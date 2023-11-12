package internal

import (
	"sync"
)

type Store[T interface{}] struct {
	Data map[string]T
	mu   *sync.RWMutex
}

func NewStore[T interface{}]() *Store[T] {
	return &Store[T]{
		Data: make(map[string]T),
		mu:   &sync.RWMutex{},
	}
}

type Author[T interface{}] interface {
	GetAuthor(string) T
	SetAuthor(string, T)
}

func (s *Store[T]) GetAuthor(id string) T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Data[id]
}

func (s *Store[T]) SetAuthor(id string, client T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data[id] = client
}
