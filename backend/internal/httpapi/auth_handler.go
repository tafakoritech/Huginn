package httpapi

import (
	"net/http"

	"github.com/tafakorpardazan/huginn/internal/auth"
)

type authHandler struct {
	svc   *auth.Service
	cookie string
}

func newAuthHandler(svc *auth.Service, cookieName string) *authHandler {
	return &authHandler{svc: svc, cookie: cookieName}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Device   string `json:"device"`
}

func (h *authHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.Login == "" || req.Password == "" {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "login and password are required"})
		return
	}

	result, err := h.svc.Login(r.Context(), auth.LoginParams{
		Login:     req.Login,
		Password:  req.Password,
		Device:    req.Device,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials, auth.ErrUserInactive:
			RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "invalid credentials"})
		default:
			internalErr(w, err)
		}
		return
	}

	respondLogin(w, h.cookie, result)
}

func (h *authHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refresh := r.Header.Get("X-Refresh-Token")
	if refresh == "" {
		refresh = r.URL.Query().Get("refresh_token")
	}
	if refresh == "" {
		RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "refresh token required"})
		return
	}
	result, err := h.svc.Refresh(r.Context(), refresh, r.Header.Get("User-Agent"), clientIP(r), r.UserAgent())
	if err != nil {
		RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "invalid or expired refresh token"})
		return
	}
	respondLogin(w, h.cookie, result)
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refresh := r.Header.Get("X-Refresh-Token")
	if refresh == "" {
		refresh = r.URL.Query().Get("refresh_token")
	}
	_ = h.svc.Logout(r.Context(), refresh)
	RespondNoContent(w)
}

func respondLogin(w http.ResponseWriter, cookieName string, result *auth.LoginResult) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    result.Tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // set true behind TLS in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(result.Tokens.ExpiresIn.Seconds()),
	})
	RespondJSON(w, http.StatusOK, map[string]any{
		"user":          result.User,
		"access_token":  result.Tokens.AccessToken,
		"refresh_token": result.Tokens.RefreshToken,
		"expires_in":    result.Tokens.ExpiresIn.Seconds(),
		"token_type":    "Bearer",
	})
}

func internalErr(w http.ResponseWriter, err error) {
	RespondError(w, http.StatusInternalServerError, &APIError{Code: CodeInternalError, Message: "internal server error"})
}
