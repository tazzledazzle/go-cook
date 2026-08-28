// Package searchindex implements a small in-memory inverted index over
// module metadata (name, language, template) — the search half of what
// real Nerv's "look up a GitLab project's info" search function did.
// This indexes METADATA only; searching a module's actual source code
// is handled separately by directory.CodeSearchProvider, which links
// out to a real code-search tool (OpenGrok, Sourcegraph) rather than
// reimplementing code indexing here.
package searchindex

import (
	"sort"
	"strings"
	"sync"
)

// Doc is one indexed module's searchable metadata.
type Doc struct {
	ID           string
	Name         string
	Language     string
	TemplateName string
}

// Index is a token -> doc-ID inverted index, safe for concurrent use.
type Index struct {
	mu       sync.RWMutex
	docs     map[string]Doc
	postings map[string]map[string]struct{} // token -> set of doc IDs
}

func NewIndex() *Index {
	return &Index{
		docs:     make(map[string]Doc),
		postings: make(map[string]map[string]struct{}),
	}
}

// Add indexes (or re-indexes) a doc. Safe to call again for the same ID
// to refresh its entry.
func (idx *Index) Add(d Doc) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.removeLocked(d.ID) // clear any prior postings for this ID first
	idx.docs[d.ID] = d

	for _, tok := range tokenize(d.Name, d.Language, d.TemplateName, d.ID) {
		if idx.postings[tok] == nil {
			idx.postings[tok] = make(map[string]struct{})
		}
		idx.postings[tok][d.ID] = struct{}{}
	}
}

// Remove drops a doc from the index entirely.
func (idx *Index) Remove(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.removeLocked(id)
}

func (idx *Index) removeLocked(id string) {
	if _, ok := idx.docs[id]; !ok {
		return
	}
	delete(idx.docs, id)
	for tok, ids := range idx.postings {
		delete(ids, id)
		if len(ids) == 0 {
			delete(idx.postings, tok)
		}
	}
}

// scoredDoc pairs a Doc with how many query tokens matched it.
type scoredDoc struct {
	doc   Doc
	score int
}

// Search tokenizes the query and returns matching docs, ranked by number
// of matching tokens (descending), then by name (ascending) for stable
// ordering among ties.
func (idx *Index) Search(query string) []Doc {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	scores := make(map[string]int)
	for _, tok := range tokenize(query) {
		for id := range idx.postings[tok] {
			scores[id]++
		}
	}

	results := make([]scoredDoc, 0, len(scores))
	for id, score := range scores {
		results = append(results, scoredDoc{doc: idx.docs[id], score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].doc.Name < results[j].doc.Name
	})

	docs := make([]Doc, len(results))
	for i, r := range results {
		docs[i] = r.doc
	}
	return docs
}

// tokenize lowercases and splits on any non-alphanumeric rune, so
// "widget-api" and "widget_api" and "Widget API" all produce the same
// token set: ["widget", "api"].
func tokenize(fields ...string) []string {
	var tokens []string
	for _, f := range fields {
		if f == "" {
			continue
		}
		for _, tok := range strings.FieldsFunc(strings.ToLower(f), func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
		}) {
			if tok != "" {
				tokens = append(tokens, tok)
			}
		}
	}
	return tokens
}
