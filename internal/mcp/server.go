// Package mcp provides MCP (Model Context Protocol) server implementation
package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/memory"
)

// Server implements an MCP server
type Server struct {
	memoryEngine *memory.Engine
	querier      db.Querier
	mux          *http.ServeMux
}

// NewServer creates a new MCP server
func NewServer(memoryEngine *memory.Engine, querier db.Querier) *Server {
	s := &Server{
		memoryEngine: memoryEngine,
		querier:      querier,
		mux:          http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/tools/memory/get", s.handleMemoryGet)
	s.mux.HandleFunc("/tools/memory/set", s.handleMemorySet)
	s.mux.HandleFunc("/tools/session/list", s.handleSessionList)
	s.mux.HandleFunc("/tools/session/get", s.handleSessionGet)
	s.mux.HandleFunc("/tools/cost/get", s.handleCostGet)
	s.mux.HandleFunc("/tools/prompt/get", s.handlePromptGet)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ToolRequest represents an MCP tool request
type ToolRequest struct {
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters"`
}

// ToolResponse represents an MCP tool response
type ToolResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents response content
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleMemoryGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		Project string `json:"project"`
		Key     string `json:"key"`
		Type    string `json:"type,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		s.sendError(w, "Invalid request body")
		return
	}

	// Query memory
	memories, err := s.memoryEngine.Recall(r.Context(), params.Project, params.Key, 5)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get memory: %v", err))
		return
	}

	var result []memory.RecallResult
	for _, m := range memories {
		if params.Type == "" || string(m.Memory.Type) == params.Type {
			result = append(result, m)
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	s.sendSuccess(w, string(data))
}

func (s *Server) handleMemorySet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		Project string  `json:"project"`
		Type    string  `json:"type"`
		Key     string  `json:"key"`
		Value   string  `json:"value"`
		Confidence float64 `json:"confidence,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		s.sendError(w, "Invalid request body")
		return
	}

	if err := s.memoryEngine.Store(r.Context(), params.Project, memory.MemoryType(params.Type), params.Key, params.Value, "mcp"); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to store memory: %v", err))
		return
	}

	s.sendSuccess(w, "Memory stored successfully")
}

func (s *Server) handleSessionList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")

	sessions, err := s.querier.ListRecentSessions(r.Context(), 50)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to list sessions: %v", err))
		return
	}

	// Filter by project if specified
	var result []db.Session
	for _, s := range sessions {
		if project == "" || s.Project == project {
			result = append(result, s)
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	s.sendSuccess(w, string(data))
}

func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		s.sendError(w, "Session ID required")
		return
	}

	session, err := s.querier.GetSession(r.Context(), sessionID)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get session: %v", err))
		return
	}

	data, _ := json.MarshalIndent(session, "", "  ")
	s.sendSuccess(w, string(data))
}

func (s *Server) handleCostGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")

	cost, err := s.querier.GetCostByProject(r.Context(), project)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get cost: %v", err))
		return
	}

	data, _ := json.MarshalIndent(cost, "", "  ")
	s.sendSuccess(w, string(data))
}

func (s *Server) handlePromptGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		s.sendError(w, "Prompt name required")
		return
	}

	prompt, err := s.querier.GetPrompt(r.Context(), name)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get prompt: %v", err))
		return
	}

	data, _ := json.MarshalIndent(prompt, "", "  ")
	s.sendSuccess(w, string(data))
}

func (s *Server) sendSuccess(w http.ResponseWriter, text string) {
	resp := ToolResponse{
		Content: []Content{{Type: "text", Text: text}},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) sendError(w http.ResponseWriter, text string) {
	resp := ToolResponse{
		Content: []Content{{Type: "text", Text: text}},
		IsError: true,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}
