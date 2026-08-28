package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordProjectGeneratedAppearsInOutput(t *testing.T) {
	m := New()
	m.RecordProjectGenerated("go")
	m.RecordProjectGenerated("go")
	m.RecordProjectGenerated("python")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	m.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `nerv_projects_generated_total{language="go"} 2`) {
		t.Errorf("metrics output missing go=2 counter, got:\n%s", body)
	}
	if !strings.Contains(body, `nerv_projects_generated_total{language="python"} 1`) {
		t.Errorf("metrics output missing python=1 counter, got:\n%s", body)
	}
}

func TestRecordResolutionLabelsHitAndMiss(t *testing.T) {
	m := New()
	m.RecordResolution(true)
	m.RecordResolution(false)
	m.RecordResolution(false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	m.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `nerv_dependency_resolution_total{result="hit"} 1`) {
		t.Errorf("metrics output missing hit=1 counter, got:\n%s", body)
	}
	if !strings.Contains(body, `nerv_dependency_resolution_total{result="miss"} 2`) {
		t.Errorf("metrics output missing miss=2 counter, got:\n%s", body)
	}
}
