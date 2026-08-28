// Package api exposes Nerv's project-generation workflow over HTTP,
// composing the registry, template, depgraph, cihook, and metrics
// packages behind a small REST surface. Unlike the CLI (one process
// per invocation), this server is long-running, so the depgraph
// resolution cache persists across requests.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"tazzledazzle/nerv/nerv/internal/cihook"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/metrics"
	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/template"
)

// Server bundles every dependency the HTTP handlers need.
type Server struct {
	Reg      registry.Registrar
	Engine   *template.Engine
	Resolver *depgraph.Resolver
	Hook     cihook.CIHook
	Metrics  *metrics.Metrics
	DestRoot string // root directory generated projects are written under
}

// NewProjectRequest is the JSON body for POST /projects.
type NewProjectRequest struct {
	Name     string       `json:"name"`
	Language string       `json:"language"`
	Deps     []DepRequest `json:"deps,omitempty"`
}

// DepRequest is one entry in NewProjectRequest.Deps.
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

// errorResponse is the JSON body for any non-2xx response.
type errorResponse struct {
	Error string `json:"error"`
}

// Router builds the *http.ServeMux exposing every Nerv HTTP endpoint,
// including the Prometheus /metrics endpoint from the metrics package.
func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("GET /projects", s.handleListProjects)
	mux.Handle("GET /metrics", s.Metrics.Handler())
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
	if req.Name == "" || req.Language == "" {
		writeError(w, http.StatusBadRequest, "name and language are required")
		return
	}

	projectID := fmt.Sprintf("%s-%d", req.Name, time.Now().UnixNano())
	projectPath := filepath.Join(s.DestRoot, req.Name)

	var resolvedDeps map[string]string
	var cacheHit bool
	if len(req.Deps) > 0 {
		deps := make([]depgraph.Dependency, len(req.Deps))
		for i, d := range req.Deps {
			deps[i] = depgraph.Dependency{Name: d.Name, Constraint: d.Version}
		}

		result, err := s.Resolver.Resolve(projectID, deps)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		resolvedDeps = result.Resolved
		cacheHit = result.CacheHit
		s.Metrics.RecordResolution(cacheHit)
	}

	start := time.Now()
	err := s.Engine.Render("service", projectPath, map[string]interface{}{"ServiceName": req.Name})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("rendering template: %v", err))
		return
	}
	s.Metrics.ObserveRenderDuration(time.Since(start).Seconds())

	pipelineRef, err := s.Hook.TriggerPipeline(projectPath, req.Language)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("wiring CI hook: %v", err))
		return
	}

	project := registry.Project{
		ID:              projectID,
		Name:            req.Name,
		Language:        req.Language,
		TemplateName:    "service",
		TemplateVersion: "v1",
		Path:            projectPath,
	}
	if err := s.Reg.Register(project); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("registering project: %v", err))
		return
	}
	s.Metrics.RecordProjectGenerated(req.Language)

	writeJSON(w, http.StatusCreated, NewProjectResponse{
		Project:        project,
		ResolvedDeps:   resolvedDeps,
		DepsCacheHit:   cacheHit,
		PipelineConfig: pipelineRef,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")

	projects, err := s.Reg.List(lang)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing projects: %v", err))
		return
	}
	if projects == nil {
		projects = []registry.Project{}
	}

	writeJSON(w, http.StatusOK, projects)
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
