package syncutil

import "sync"

// Guarded wraps given type in a [sync.RWMutex].
type Guarded[T any] struct {
	value T
	mu    sync.RWMutex
}

func (g *Guarded[T]) Load() T {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.value
}

func (g *Guarded[T]) Store(value T) {
	g.Modify(func(v *T) {
		*v = value
	})
}

func (g *Guarded[T]) Modify(f func(value *T)) {
	g.mu.Lock()
	defer g.mu.Unlock()

	f(&g.value)
}
