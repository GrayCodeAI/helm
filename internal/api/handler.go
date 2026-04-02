// Package api provides REST API handlers
package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/memory"
)

// Handler handles API requests
type Handler struct {
	querier      db.Querier
	memoryEngine *memory.Engine
}

// NewHandler creates a new API handler
func NewHandler(querier db.Querier, memoryEngine *memory.Engine) *Handler {
	return &Handler{
		querier:      querier,
		memoryEngine: memoryEngine,
	}
}

// RegisterRoutes registers API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/sessions/", h.handleSessionDetail)
	mux.HandleFunc("/api/cost", h.handleCost)
	mux.HandleFunc("/api/memory", h.handleMemory)
	mux.HandleFunc("/api/prompts", h.handlePrompts)
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSessions(w, r)
	case http.MethodPost:
		h.createSession(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.querier.ListRecentSessions(r.Context(), 100)
	if err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, sessions)
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Project  string `json:"project"`
		Prompt   string `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, err, http.StatusBadRequest)
		return
	}

	session := db.CreateSessionParams{
		ID:       generateID(),
		Provider: req.Provider,
		Model:    req.Model,
		Project:  req.Project,
		Prompt:   sql.NullString{String: req.Prompt, Valid: req.Prompt != ""},
		Status:   "running",
	}

	if _, err := h.querier.CreateSession(r.Context(), session); err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, session)
}

func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/sessions/"):]

	switch r.Method {
	case http.MethodGet:
		session, err := h.querier.GetSession(r.Context(), id)
		if err != nil {
			h.sendError(w, err, http.StatusNotFound)
			return
		}
		h.sendJSON(w, session)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	if project == "" {
		http.Error(w, "Project required", http.StatusBadRequest)
		return
	}

	cost, err := h.querier.GetCostByProject(r.Context(), project)
	if err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, cost)
}

func (h *Handler) handleMemory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMemory(w, r)
	case http.MethodPost:
		h.createMemory(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listMemory(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		http.Error(w, "Project required", http.StatusBadRequest)
		return
	}

	memories, err := h.memoryEngine.List(r.Context(), project)
	if err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, memories)
}

func (h *Handler) createMemory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Project string `json:"project"`
		Type    string `json:"type"`
		Key     string `json:"key"`
		Value   string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.memoryEngine.Store(r.Context(), req.Project, memory.MemoryType(req.Type), req.Key, req.Value, "api"); err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, map[string]string{"status": "stored"})
}

func (h *Handler) handlePrompts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prompts, err := h.querier.ListPrompts(r.Context())
	if err != nil {
		h.sendError(w, err, http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, prompts)
}

func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func generateID() string {
	return time.Now().Format("20060102150405") + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
