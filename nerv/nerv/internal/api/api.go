// Package api exposes Nerv's Module Creation Service and Module
// Directory Service over HTTP. Handlers here are deliberately thin:
// they translate HTTP <-> Go types and delegate all real work to
// creation.Service and directory.Service.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"tazzledazzle/nerv/nerv/internal/creation"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/directory"
	"tazzledazzle/nerv/nerv/internal/registry"
)

// Server bundles the two services and the project-to-project graph.
type Server struct {
	Creation  *creation.Service
	Directory *directory.Service
	Graph     *depgraph.Graph
}

// NewProjectRequest is the JSON body for POST /projects.
type NewProjectRequest struct {
	Name     string       `json:"name"`
	Language string       `json:"language"`
	Deps     []DepRequest `json:"deps,omitempty"`
}

type DepRequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewProjectResponse is returned on a successful POST /projects.
type NewProjectResponse struct {
	Project        registry.Project  `json:"project"`
	ResolvedDeps   map[string]string `json:"resolved_deps,omitempty"`
	DepsCacheHit   bool              `json:"deps_cache_hit"`
	PipelineConfig string            `json:"pipeline_config"`
}

// DependsOnRequest is the JSON body for POST /projects/{id}/depends-on.
type DependsOnRequest struct {
	DependsOnID string `json:"depends_on_id"`
}

// SearchResponse is returned by GET /search.
type SearchResponse struct {
	Query           string             `json:"query"`
	Modules         []registry.Project `json:"modules"`
	CodeSearchLinks map[string]string  `json:"code_search_links,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Router builds the *http.ServeMux exposing every Nerv HTTP endpoint.
func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("GET /projects", s.handleListProjects)
	mux.HandleFunc("POST /projects/{id}/depends-on", s.handleAddDependency)
	mux.HandleFunc("GET /projects/{id}/dependents", s.handleGetDependents)
	mux.HandleFunc("GET /search", s.handleSearch)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req NewProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	deps := make([]depgraph.Dependency, len(req.Deps))
	for i, d := range req.Deps {
		deps[i] = depgraph.Dependency{Name: d.Name, Constraint: d.Version}
	}

	result, err := s.Creation.Create(creation.Request{
		Name:     req.Name,
		Language: req.Language,
		Deps:     deps,
	})
	if err != nil {
		// Policy violations (dependency resolution failures) are the
		// client's fault; anything else is treated as a server error.
		status := http.StatusInternalServerError
		if req.Name == "" || req.Language == "" {
			status = http.StatusBadRequest
		} else if len(req.Deps) > 0 {
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, NewProjectResponse{
		Project:        result.Project,
		ResolvedDeps:   result.ResolvedDeps,
		DepsCacheHit:   result.DepsCacheHit,
		PipelineConfig: result.PipelineConfig,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")

	projects, err := s.Directory.List(lang)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing projects: %v", err))
		return
	}
	if projects == nil {
		projects = []registry.Project{}
	}

	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter \"q\" is required")
		return
	}

	modules, err := s.Directory.Search(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("searching: %v", err))
		return
	}
	if modules == nil {
		modules = []registry.Project{}
	}

	writeJSON(w, http.StatusOK, SearchResponse{
		Query:           query,
		Modules:         modules,
		CodeSearchLinks: s.Directory.CodeSearchLinks(query),
	})
}

func (s *Server) handleAddDependency(w http.ResponseWriter, r *http.Request) {
	fromID := r.PathValue("id")

	var req DependsOnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.DependsOnID == "" {
		writeError(w, http.StatusBadRequest, "depends_on_id is required")
		return
	}

	if _, err := s.Directory.Get(fromID); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found: %v", fromID, err))
		return
	}
	if _, err := s.Directory.Get(req.DependsOnID); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found: %v", req.DependsOnID, err))
		return
	}

	if err := s.Graph.AddEdge(fromID, req.DependsOnID); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"from":       fromID,
		"depends_on": req.DependsOnID,
	})
}

func (s *Server) handleGetDependents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := s.Directory.Get(id); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found: %v", id, err))
		return
	}

	dependents := s.Graph.Dependents(id)
	if dependents == nil {
		dependents = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id": id,
		"dependents": dependents,
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("api: encoding response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
