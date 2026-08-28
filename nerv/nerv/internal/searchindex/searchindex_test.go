package searchindex

import "testing"

func TestSearchFindsExactNameMatch(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "widget-api", Language: "go", TemplateName: "service"})
	idx.Add(Doc{ID: "2", Name: "auth-lib", Language: "go", TemplateName: "library"})

	results := idx.Search("widget")
	if len(results) != 1 || results[0].ID != "1" {
		t.Errorf("Search(widget) = %v, want exactly doc 1", results)
	}
}

func TestSearchMatchesLanguageField(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "widget-api", Language: "go", TemplateName: "service"})
	idx.Add(Doc{ID: "2", Name: "data-pipeline", Language: "python", TemplateName: "service"})

	results := idx.Search("python")
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("Search(python) = %v, want exactly doc 2", results)
	}
}

func TestSearchRanksMoreMatchingTokensHigher(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "widget-api", Language: "go", TemplateName: "service"})
	idx.Add(Doc{ID: "2", Name: "widget-service-go", Language: "go", TemplateName: "service"})

	// "widget go service" matches doc 2 on all three tokens, doc 1 on two.
	results := idx.Search("widget go service")
	if len(results) != 2 {
		t.Fatalf("Search() returned %d results, want 2", len(results))
	}
	if results[0].ID != "2" {
		t.Errorf("top result = %+v, want doc 2 (more matching tokens)", results[0])
	}
}

func TestSearchIsCaseAndSeparatorInsensitive(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "Widget-API", Language: "go", TemplateName: "service"})

	results := idx.Search("WIDGET_api")
	if len(results) != 1 {
		t.Errorf("Search() = %v, want doc 1 matched regardless of case/separator", results)
	}
}

func TestSearchReturnsEmptyForNoMatch(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "widget-api", Language: "go", TemplateName: "service"})

	results := idx.Search("nonexistent-term")
	if len(results) != 0 {
		t.Errorf("Search() = %v, want empty", results)
	}
}

func TestAddReplacesExistingEntry(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "old-name", Language: "go", TemplateName: "service"})
	idx.Add(Doc{ID: "1", Name: "new-name", Language: "python", TemplateName: "service"})

	if results := idx.Search("old-name"); len(results) != 0 {
		t.Errorf("Search(old-name) = %v, want empty after re-Add", results)
	}
	if results := idx.Search("new-name"); len(results) != 1 {
		t.Errorf("Search(new-name) = %v, want 1 match after re-Add", results)
	}
}

func TestRemoveDropsDoc(t *testing.T) {
	idx := NewIndex()
	idx.Add(Doc{ID: "1", Name: "widget-api", Language: "go", TemplateName: "service"})
	idx.Remove("1")

	if results := idx.Search("widget"); len(results) != 0 {
		t.Errorf("Search() after Remove = %v, want empty", results)
	}
}
