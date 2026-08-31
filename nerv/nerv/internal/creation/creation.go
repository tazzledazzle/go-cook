// Package creation implements the Module Creation Service: orchestrates
// dependency resolution, template rendering, and CI wiring for a new
// module, then hands the resulting metadata to the Module Directory
// Service (package directory) for registration — exactly once, and only
// after generation has fully succeeded. Creation never registers or
// indexes anything itself; that's Directory's job.
package creation

import (
	"fmt"
	"path/filepath"
	"time"

	"tazzledazzle/nerv/nerv/internal/cihook"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/directory"
	"tazzledazzle/nerv/nerv/internal/metrics"
	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/template"
)

// Request describes a module to create.
type Request struct {
	Name     string
	Language string
	Deps     []depgraph.Dependency
}

// Result is what Create returns on success.
type Result struct {
	Project        registry.Project
	ResolvedDeps   map[string]string
	DepsCacheHit   bool
	PipelineConfig string
}

// Service is the Module Creation Service.
type Service struct {
	Engine    *template.Engine
	Resolver  *depgraph.Resolver
	Hook      cihook.CIHook
	Directory *directory.Service
	Metrics   *metrics.Metrics // optional; nil is fine, calls are skipped
	DestRoot  string
}

// Create resolves dependencies, renders the module's template, wires
// CI, and registers the resulting metadata with the Directory Service —
// in that order, aborting before any disk/registry side effect if an
// earlier step fails.
func (s *Service) Create(req Request) (Result, error) {
	if req.Name == "" || req.Language == "" {
		return Result{}, fmt.Errorf("creation: name and language are required")
	}

	projectID := fmt.Sprintf("%s-%d", req.Name, time.Now().UnixNano())
	projectPath := filepath.Join(s.DestRoot, req.Name)

	var resolvedDeps map[string]string
	var cacheHit bool
	if len(req.Deps) > 0 {
		result, err := s.Resolver.Resolve(projectID, req.Deps)
		if err != nil {
			return Result{}, fmt.Errorf("creation: resolving dependencies: %w", err)
		}
		resolvedDeps = result.Resolved
		cacheHit = result.CacheHit
		if s.Metrics != nil {
			s.Metrics.RecordResolution(cacheHit)
		}
	}

	start := time.Now()
	if err := s.Engine.Render("service", projectPath, map[string]interface{}{"ServiceName": req.Name}); err != nil {
		return Result{}, fmt.Errorf("creation: rendering template: %w", err)
	}
	if s.Metrics != nil {
		s.Metrics.ObserveRenderDuration(time.Since(start).Seconds())
	}

	pipelineRef, err := s.Hook.TriggerPipeline(projectPath, req.Language)
	if err != nil {
		return Result{}, fmt.Errorf("creation: wiring CI hook: %w", err)
	}

	project := registry.Project{
		ID:              projectID,
		Name:            req.Name,
		Language:        req.Language,
		TemplateName:    "service",
		TemplateVersion: "v1",
		Path:            projectPath,
		CreatedAt:       time.Now().UTC(),
	}

	// Metadata registration happens LAST, and exactly once — only after
	// resolution, rendering, and CI wiring have all already succeeded.
	if err := s.Directory.Register(project); err != nil {
		return Result{}, fmt.Errorf("creation: registering with directory service: %w", err)
	}
	if s.Metrics != nil {
		s.Metrics.RecordProjectGenerated(req.Language)
	}

	return Result{
		Project:        project,
		ResolvedDeps:   resolvedDeps,
		DepsCacheHit:   cacheHit,
		PipelineConfig: pipelineRef,
	}, nil
}
