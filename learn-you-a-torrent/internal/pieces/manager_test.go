package pieces

import "testing"

func TestManager_nextMissingAndMarkComplete(t *testing.T) {
	m := NewManager(3)

	index, ok := m.NextMissing()
	if !ok || index != 0 {
		t.Fatalf("NextMissing() = (%d, %v), want (0, true)", index, ok)
	}

	m.MarkComplete(0)
	index, ok = m.NextMissing()
	if !ok || index != 1 {
		t.Fatalf("NextMissing() after 0 = (%d, %v), want (1, true)", index, ok)
	}

	m.MarkComplete(1)
	m.MarkComplete(2)
	_, ok = m.NextMissing()
	if ok {
		t.Fatal("NextMissing() = true, want false when all complete")
	}
	if !m.Complete() {
		t.Error("Complete() = false, want true")
	}
}
