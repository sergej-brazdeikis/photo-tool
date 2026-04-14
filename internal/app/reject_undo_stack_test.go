package app

import "testing"

func TestReviewRejectUndoStack_LIFO(t *testing.T) {
	var s reviewRejectUndoStack
	s.Push(1)
	s.Push(2)
	s.Push(3)
	id, ok := s.Pop()
	if !ok || id != 3 {
		t.Fatalf("Pop: got %v ok=%v want 3 true", id, ok)
	}
	id, ok = s.Pop()
	if !ok || id != 2 {
		t.Fatalf("Pop: got %v ok=%v want 2 true", id, ok)
	}
	id, ok = s.Pop()
	if !ok || id != 1 {
		t.Fatalf("Pop: got %v ok=%v want 1 true", id, ok)
	}
	_, ok = s.Pop()
	if ok {
		t.Fatal("expected empty stack")
	}
}

func TestReviewRejectUndoStack_trimOldestWhenOverCap(t *testing.T) {
	var s reviewRejectUndoStack
	for i := int64(1); i <= maxReviewRejectUndoIDs+10; i++ {
		s.Push(i)
	}
	if g := s.Len(); g != maxReviewRejectUndoIDs {
		t.Fatalf("Len after push: got %d want %d", g, maxReviewRejectUndoIDs)
	}
	// Oldest were 1..10 dropped; tail should end at max+10
	last, ok := s.Pop()
	if !ok || last != maxReviewRejectUndoIDs+10 {
		t.Fatalf("Pop tail: got %v ok=%v", last, ok)
	}
	second, ok := s.Pop()
	if !ok {
		t.Fatal("expected more ids")
	}
	if second != maxReviewRejectUndoIDs+9 {
		t.Fatalf("second pop: got %d want %d", second, maxReviewRejectUndoIDs+9)
	}
}

func TestReviewRejectUndoStack_Clear(t *testing.T) {
	var s reviewRejectUndoStack
	s.Push(42)
	s.Clear()
	if s.Len() != 0 {
		t.Fatalf("after Clear Len=%d", s.Len())
	}
	_, ok := s.Pop()
	if ok {
		t.Fatal("pop after clear")
	}
}

func TestReviewRejectUndoStack_nilSafe(t *testing.T) {
	var s *reviewRejectUndoStack
	s.Push(1)
	if s.Len() != 0 {
		t.Fatal("nil stack len")
	}
	if _, ok := s.Pop(); ok {
		t.Fatal("nil pop")
	}
	s.Clear()
}

func TestReviewRejectUndoStack_pushNonPositiveIgnored(t *testing.T) {
	var s reviewRejectUndoStack
	s.Push(0)
	s.Push(-1)
	if s.Len() != 0 {
		t.Fatalf("non-positive push should not grow stack: Len=%d", s.Len())
	}
	s.Push(1)
	if s.Len() != 1 {
		t.Fatalf("Len: got %d want 1", s.Len())
	}
}
