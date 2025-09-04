package ring

import (
	"fmt"
	"sync"
)

// RingBuffer is a thread-safe generic ring buffer implementation
type RingBuffer[T any] struct {
	buffer  []T
	head    int
	tail    int
	size    int
	maxSize int
	isFull  bool
	mu      sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified maximum size
func NewRingBuffer[T any](maxSize int) *RingBuffer[T] {
	if maxSize <= 0 {
		panic("maxSize must be greater than 0")
	}
	return &RingBuffer[T]{
		buffer:  make([]T, maxSize),
		maxSize: maxSize,
	}
}

// Push adds an element to the ring buffer
// If the buffer is full, it overwrites the oldest element
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.maxSize

	if rb.isFull {
		rb.head = (rb.head + 1) % rb.maxSize
	} else {
		rb.size++
		if rb.size == rb.maxSize {
			rb.isFull = true
		}
	}
}

// Pop removes and returns the oldest element from the buffer
// Returns the element and true if successful, zero value and false if empty
func (rb *RingBuffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var zero T
	if rb.size == 0 {
		return zero, false
	}

	item := rb.buffer[rb.head]
	rb.buffer[rb.head] = zero // Clear the slot to avoid memory leaks
	rb.head = (rb.head + 1) % rb.maxSize
	rb.size--
	rb.isFull = false

	return item, true
}

// Peek returns the oldest element without removing it
// Returns the element and true if successful, zero value and false if empty
func (rb *RingBuffer[T]) Peek() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var zero T
	if rb.size == 0 {
		return zero, false
	}

	return rb.buffer[rb.head], true
}

// Size returns the current number of elements in the buffer
func (rb *RingBuffer[T]) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// MaxSize returns the maximum capacity of the buffer
func (rb *RingBuffer[T]) MaxSize() int {
	return rb.maxSize
}

// IsEmpty returns true if the buffer is empty
func (rb *RingBuffer[T]) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size == 0
}

// IsFull returns true if the buffer is full
func (rb *RingBuffer[T]) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.isFull
}

// Clear removes all elements from the buffer
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var zero T
	for i := range rb.buffer {
		rb.buffer[i] = zero
	}
	rb.head = 0
	rb.tail = 0
	rb.size = 0
	rb.isFull = false
}

// Iterator represents an iterator over the ring buffer
type Iterator[T any] struct {
	rb       *RingBuffer[T]
	current  int
	count    int
	snapshot []T
}

// NewIterator creates a new iterator for the ring buffer
// It takes a snapshot of the current state to ensure thread-safe iteration
func (rb *RingBuffer[T]) NewIterator() *Iterator[T] {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	snapshot := make([]T, rb.size)
	if rb.size > 0 {
		for i := 0; i < rb.size; i++ {
			idx := (rb.head + i) % rb.maxSize
			snapshot[i] = rb.buffer[idx]
		}
	}

	return &Iterator[T]{
		rb:       rb,
		snapshot: snapshot,
	}
}

// HasNext returns true if there are more elements to iterate over
func (it *Iterator[T]) HasNext() bool {
	return it.count < len(it.snapshot)
}

// Next returns the next element in the iteration
// Returns the element and true if successful, zero value and false if no more elements
func (it *Iterator[T]) Next() (T, bool) {
	var zero T
	if !it.HasNext() {
		return zero, false
	}

	item := it.snapshot[it.count]
	it.count++
	return item, true
}

// ToSlice returns a slice containing all elements in the buffer in order (oldest to newest)
func (rb *RingBuffer[T]) ToSlice() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []T{}
	}

	result := make([]T, rb.size)
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.maxSize
		result[i] = rb.buffer[idx]
	}
	return result
}

// String returns a string representation of the ring buffer
func (rb *RingBuffer[T]) String() string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	return fmt.Sprintf("RingBuffer{size: %d, maxSize: %d, elements: %v}",
		rb.size, rb.maxSize, rb.ToSlice())
}
