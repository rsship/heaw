package internal

import (
	"errors"
	"sync"
)

type Storer[K any, V any] interface {
	Get(K) (*V, error)
	Set(K, V) bool
	Update(K, V) V
	Delete(K) bool
	ForEach(fn func(K, V))
}

type InternalStore[K comparable, V any] struct {
	data map[K]V
	mut  *sync.RWMutex
}

func NewInternalStore[K comparable, V any]() *InternalStore[K, V] {
	return &InternalStore[K, V]{
		data: make(map[K]V),
		mut:  &sync.RWMutex{},
	}
}

func (s *InternalStore[K, V]) Get(key K) (*V, error) {
	s.mut.RLock()
	defer s.mut.RUnlock()
	data, ok := s.data[key]
	if !ok {
		return nil, errors.New("NOT FOUND")
	}
	return &data, nil
}

func (s *InternalStore[K, V]) Set(key K, val V) bool {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.data[key] = val

	return true
}

func (s *InternalStore[K, V]) Update(key K, val V) V {

	s.mut.Lock()
	defer s.mut.Unlock()

	data := s.data[key]
	s.data[key] = val

	return data
}

func (s *InternalStore[K, V]) Delete(key K) bool {

	s.mut.Lock()
	defer s.mut.Unlock()

	delete(s.data, key)

	return true
}

func (s *InternalStore[K, V]) ForEach(fn func(K, V)) {

	for k, v := range s.data {
		go fn(k, v)
	}
}
