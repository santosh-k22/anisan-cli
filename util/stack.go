// Package util provides a collection of domain-agnostic utility functions and cross-platform helpers.
package util

// Stack implements a parameterized Last-In-First-Out (LIFO) data structure.
type Stack[T any] struct {
	items []T
}

// Push appends a new element to the top of the stack.
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop removes and returns the topmost element of the stack; returns the zero value if the stack is empty.
func (s *Stack[T]) Pop() (item T) {
	if len(s.items) == 0 {
		return
	}
	idx := len(s.items) - 1
	item = s.items[idx]
	s.items = s.items[:idx]
	return
}

// Peek returns the topmost element without removing it; returns the zero value if the stack is empty.
func (s *Stack[T]) Peek() (item T) {
	if len(s.items) == 0 {
		return
	}
	return s.items[len(s.items)-1]
}

// Len returns the total number of elements currently stored in the stack.
func (s *Stack[T]) Len() int {
	return len(s.items)
}

// Clear removes all elements from the stack, resetting it to an empty state.
func (s *Stack[T]) Clear() {
	s.items = nil
}
