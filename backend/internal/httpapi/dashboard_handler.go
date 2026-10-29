package httpapi

import (
	"net/http"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/dashboard"
)

type dashboardHandler struct {
	svc *dashboard.Service
}

func newDashboardHandler(svc *dashboard.Service) *dashboardHandler {
	return &dashboardHandler{svc: svc}
}

func (h *dashboardHandler) Manager(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.Manager(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, metrics)
}

func (h *dashboardHandler) Agent(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	metrics, err := h.svc.Agent(r.Context(), p.UserID)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, metrics)
}

func (h *dashboardHandler) Funnel(w http.ResponseWriter, r *http.Request) {
	steps, err := h.svc.Funnel(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, steps)
}

func (h *dashboardHandler) QualificationBySource(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.QualificationBySource(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, rows)
}

func (h *dashboardHandler) AgentPerformance(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.AgentPerformance(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, rows)
}

func (h *dashboardHandler) LeadTrend(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	rows, err := h.svc.LeadTrend(r.Context(), period)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, rows)
}

func (h *dashboardHandler) RejectionReasons(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.RejectionReasons(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, rows)
}

func (h *dashboardHandler) LeadsByCity(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.LeadsByCity(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, rows)
}
