package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/imports"
)

type importHandler struct {
	imports   *imports.Service
	documents *documents.Service
}

func newImportHandler(imports *imports.Service, documents *documents.Service) *importHandler {
	return &importHandler{imports: imports, documents: documents}
}

func (h *importHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string            `json:"name"`
		Description *string           `json:"description"`
		SourceType  imports.SourceType `json:"source_type"`
		ProjectID   *string           `json:"project_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.SourceType == "" {
		req.SourceType = imports.SourceRawText
	}
	p, _ := auth.FromContext(r.Context())
	b, err := h.imports.CreateBatch(r.Context(), imports.CreateBatchParams{
		Name: req.Name, Description: req.Description, SourceType: req.SourceType, CreatedBy: &p.UserID, ProjectID: req.ProjectID,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusCreated, b)
}

func (h *importHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	statusStr := r.URL.Query().Get("status")
	var status *imports.Status
	if statusStr != "" {
		s := imports.Status(statusStr)
		status = &s
	}
	batches, err := h.imports.ListBatches(r.Context(), status, limit, offset)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"batches": batches, "count": len(batches)})
}

func (h *importHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "batchID")
	b, err := h.imports.GetBatch(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "batch not found"})
		return
	}
	// Include lead_count so the dashboard document table never renders undefined.
	docs, _ := h.imports.Repo().ListDocumentsWithCounts(r.Context(), id)
	RespondOK(w, map[string]any{"batch": b, "documents": docs})
}

func (h *importHandler) Queue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "batchID")
	if err := h.imports.QueueBatch(r.Context(), id); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondOK(w, map[string]any{"queued": true, "batch_id": id})
}

func (h *importHandler) AddDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "batchID")
	r.Body = http.MaxBytesReader(w, r.Body, 25*1024*1024)
	file, header, err := r.FormFile("file")
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: "file field required"})
		return
	}
	defer file.Close()
	p, _ := auth.FromContext(r.Context())

	doc, err := h.imports.AddDocument(r.Context(), id, p.UserID, header.Filename, header.Header.Get("Content-Type"), header.Size, file)
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusCreated, doc)
}

func (h *importHandler) AddRawText(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "batchID")
	var req struct {
		Text string `json:"text"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.Text == "" {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "text is required"})
		return
	}
	p, _ := auth.FromContext(r.Context())
	doc, err := h.imports.AddRawText(r.Context(), id, p.UserID, req.Text)
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusCreated, doc)
}

func (h *importHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "docID")
	doc, err := h.imports.GetDocument(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "document not found"})
		return
	}
	RespondOK(w, doc)
}
