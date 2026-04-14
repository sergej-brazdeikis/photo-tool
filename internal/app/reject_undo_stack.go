package app

import "sync"

// maxReviewRejectUndoIDs caps in-memory undo ids for a long Review session (Story 2.6 AC8).
// When full, the oldest ids are dropped; Undo still pops LIFO from the tail.
const maxReviewRejectUndoIDs = 128

// reviewRejectUndoStack is a Review-scoped LIFO of asset ids for session undo (FR-30).
// Push only when RejectAsset reports changed==true.
type reviewRejectUndoStack struct {
	mu  sync.Mutex
	ids []int64
}

func (s *reviewRejectUndoStack) Len() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.ids)
}

func (s *reviewRejectUndoStack) Push(id int64) {
	if s == nil || id <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, id)
	s.trimLocked()
}

func (s *reviewRejectUndoStack) trimLocked() {
	for len(s.ids) > maxReviewRejectUndoIDs {
		// Drop oldest (front); keep LIFO pops from the end.
		s.ids = append([]int64(nil), s.ids[1:]...)
	}
}

// Pop removes and returns the most recently pushed id, if any.
func (s *reviewRejectUndoStack) Pop() (id int64, ok bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.ids)
	if n == 0 {
		return 0, false
	}
	id = s.ids[n-1]
	s.ids = s.ids[:n-1]
	return id, true
}

// Clear drops all undo state (shell left Review or app exit).
func (s *reviewRejectUndoStack) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = nil
}
