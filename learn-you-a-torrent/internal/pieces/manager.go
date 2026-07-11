package pieces

import "sync"

// Manager tracks which torrent pieces have been downloaded and verified.
type Manager struct {
	total int
	done  map[int]bool
	mu    sync.Mutex
}

// NewManager creates a manager for total pieces.
func NewManager(total int) *Manager {
	return &Manager{
		total: total,
		done:  make(map[int]bool),
	}
}

// NextMissing returns the lowest-index piece not yet marked complete.
func (m *Manager) NextMissing() (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := 0; i < m.total; i++ {
		if !m.done[i] {
			return i, true
		}
	}
	return 0, false
}

// MarkComplete records a verified piece as downloaded.
func (m *Manager) MarkComplete(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.done[index] = true
}

// Complete reports whether all pieces are downloaded.
func (m *Manager) Complete() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.done) == m.total
}

// CompletedCount returns the number of verified pieces.
func (m *Manager) CompletedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.done)
}
