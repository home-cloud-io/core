package set

import (
	"maps"
	"slices"
	"sync"
)

type Set[T comparable] struct {
	mu sync.RWMutex
	m  map[T]struct{}
}

func New[T comparable]() *Set[T] {
	return &Set[T]{
		mu: sync.RWMutex{},
		m: map[T]struct{}{},
	}
}

func (s *Set[T]) Add(key T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = struct{}{}
}

func (s *Set[T]) Remove(key T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

func (s *Set[T]) Has(key T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[key]
	return ok
}

func (s *Set[T]) Empty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m) == 0
}

func (s *Set[T]) Items() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Collect(maps.Keys(s.m))
}
