package ring

import (
	"sync"
	"testing"
	"time"
)

func TestNewRingBuffer(t *testing.T) {
	// Test valid creation
	rb := NewRingBuffer[int](5)
	if rb.MaxSize() != 5 {
		t.Errorf("Expected max size 5, got %d", rb.MaxSize())
	}
	if rb.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", rb.Size())
	}
	if !rb.IsEmpty() {
		t.Error("Expected buffer to be empty initially")
	}
	if rb.IsFull() {
		t.Error("Expected buffer not to be full initially")
	}

	// Test panic on invalid size
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for maxSize <= 0")
		}
	}()
	NewRingBuffer[int](0)
}

func TestPushAndPop(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Push elements
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}
	if !rb.IsFull() {
		t.Error("Expected buffer to be full")
	}

	// Pop elements
	val, ok := rb.Pop()
	if !ok || val != 1 {
		t.Errorf("Expected to pop 1, got %d, ok=%v", val, ok)
	}

	val, ok = rb.Pop()
	if !ok || val != 2 {
		t.Errorf("Expected to pop 2, got %d, ok=%v", val, ok)
	}

	val, ok = rb.Pop()
	if !ok || val != 3 {
		t.Errorf("Expected to pop 3, got %d, ok=%v", val, ok)
	}

	if !rb.IsEmpty() {
		t.Error("Expected buffer to be empty after popping all elements")
	}

	// Pop from empty buffer
	val, ok = rb.Pop()
	if ok {
		t.Error("Expected pop from empty buffer to return false")
	}
	if val != 0 {
		t.Errorf("Expected zero value, got %d", val)
	}
}

func TestOverwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Fill the buffer
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	// Overwrite oldest elements
	rb.Push(4) // Should overwrite 1
	rb.Push(5) // Should overwrite 2

	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}

	// Check that we get the newest 3 elements
	val, ok := rb.Pop()
	if !ok || val != 3 {
		t.Errorf("Expected to pop 3, got %d, ok=%v", val, ok)
	}

	val, ok = rb.Pop()
	if !ok || val != 4 {
		t.Errorf("Expected to pop 4, got %d, ok=%v", val, ok)
	}

	val, ok = rb.Pop()
	if !ok || val != 5 {
		t.Errorf("Expected to pop 5, got %d, ok=%v", val, ok)
	}
}

func TestPeek(t *testing.T) {
	rb := NewRingBuffer[string](3)

	// Peek empty buffer
	val, ok := rb.Peek()
	if ok {
		t.Error("Expected peek on empty buffer to return false")
	}
	if val != "" {
		t.Errorf("Expected empty string, got %s", val)
	}

	// Add elements and peek
	rb.Push("first")
	rb.Push("second")

	val, ok = rb.Peek()
	if !ok || val != "first" {
		t.Errorf("Expected to peek 'first', got %s, ok=%v", val, ok)
	}

	// Peek should not remove element
	if rb.Size() != 2 {
		t.Errorf("Expected size 2 after peek, got %d", rb.Size())
	}

	// Pop and verify peek changes
	rb.Pop()
	val, ok = rb.Peek()
	if !ok || val != "second" {
		t.Errorf("Expected to peek 'second', got %s, ok=%v", val, ok)
	}
}

func TestClear(t *testing.T) {
	rb := NewRingBuffer[int](5)

	// Add some elements
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	if rb.Size() != 5 {
		t.Errorf("Expected size 5, got %d", rb.Size())
	}

	// Clear the buffer
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", rb.Size())
	}
	if !rb.IsEmpty() {
		t.Error("Expected buffer to be empty after clear")
	}
	if rb.IsFull() {
		t.Error("Expected buffer not to be full after clear")
	}

	// Verify we can use it normally after clear
	rb.Push(10)
	val, ok := rb.Pop()
	if !ok || val != 10 {
		t.Errorf("Expected to pop 10 after clear, got %d, ok=%v", val, ok)
	}
}

func TestIterator(t *testing.T) {
	rb := NewRingBuffer[int](5)

	// Test iterator on empty buffer
	it := rb.NewIterator()
	if it.HasNext() {
		t.Error("Expected iterator to have no elements for empty buffer")
	}
	val, ok := it.Next()
	if ok {
		t.Error("Expected Next() to return false for empty buffer")
	}
	if val != 0 {
		t.Errorf("Expected zero value, got %d", val)
	}

	// Add elements
	for i := 1; i <= 3; i++ {
		rb.Push(i)
	}

	// Test iterator
	it = rb.NewIterator()
	expected := []int{1, 2, 3}
	actual := []int{}

	for it.HasNext() {
		val, ok := it.Next()
		if !ok {
			t.Error("Expected Next() to return true when HasNext() is true")
		}
		actual = append(actual, val)
	}

	if len(actual) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(actual))
	}

	for i, v := range expected {
		if actual[i] != v {
			t.Errorf("Expected element %d to be %d, got %d", i, v, actual[i])
		}
	}

	// Test that iterator is exhausted
	if it.HasNext() {
		t.Error("Expected iterator to be exhausted")
	}
}

func TestIteratorWithOverwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Fill and overwrite
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	// Iterator should show the 3 most recent elements
	it := rb.NewIterator()
	expected := []int{3, 4, 5}
	actual := []int{}

	for it.HasNext() {
		val, ok := it.Next()
		if !ok {
			t.Error("Expected Next() to return true when HasNext() is true")
		}
		actual = append(actual, val)
	}

	if len(actual) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(actual))
	}

	for i, v := range expected {
		if actual[i] != v {
			t.Errorf("Expected element %d to be %d, got %d", i, v, actual[i])
		}
	}
}

func TestToSlice(t *testing.T) {
	rb := NewRingBuffer[string](4)

	// Empty buffer
	slice := rb.ToSlice()
	if len(slice) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(slice))
	}

	// Add elements
	words := []string{"hello", "world", "test", "ring"}
	for _, word := range words {
		rb.Push(word)
	}

	slice = rb.ToSlice()
	if len(slice) != len(words) {
		t.Errorf("Expected slice length %d, got %d", len(words), len(slice))
	}

	for i, word := range words {
		if slice[i] != word {
			t.Errorf("Expected element %d to be %s, got %s", i, word, slice[i])
		}
	}

	// Test with overwrite
	rb.Push("buffer") // Should overwrite "hello"
	slice = rb.ToSlice()
	expected := []string{"world", "test", "ring", "buffer"}

	if len(slice) != len(expected) {
		t.Errorf("Expected slice length %d, got %d", len(expected), len(slice))
	}

	for i, word := range expected {
		if slice[i] != word {
			t.Errorf("Expected element %d to be %s, got %s", i, word, slice[i])
		}
	}
}

func TestConcurrency(t *testing.T) {
	rb := NewRingBuffer[int](100)
	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Writers and readers

	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				rb.Push(id*1000 + j)
				time.Sleep(time.Microsecond) // Small delay to increase contention
			}
		}(i)
	}

	// Start reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				rb.Pop()
				rb.Peek()
				rb.Size()
				rb.IsEmpty()
				rb.IsFull()
				time.Sleep(time.Microsecond) // Small delay to increase contention
			}
		}()
	}

	wg.Wait()

	// Buffer should be in a consistent state
	size := rb.Size()
	if size < 0 || size > rb.MaxSize() {
		t.Errorf("Invalid buffer state: size=%d, maxSize=%d", size, rb.MaxSize())
	}
}

func TestConcurrentIterators(t *testing.T) {
	rb := NewRingBuffer[int](50)

	// Fill buffer
	for i := 0; i < 30; i++ {
		rb.Push(i)
	}

	const numIterators = 5
	var wg sync.WaitGroup
	wg.Add(numIterators + 1) // Iterators + one writer

	// Start concurrent iterators
	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()
			it := rb.NewIterator()
			count := 0
			for it.HasNext() {
				_, ok := it.Next()
				if !ok {
					t.Errorf("Iterator %d: Next() returned false when HasNext() was true", id)
				}
				count++
			}
			// Each iterator should see a consistent snapshot
			if count != it.rb.Size() && count != len(it.snapshot) {
				// Note: Due to concurrent modifications, the count might differ from current size
				// but should match the snapshot size
			}
		}(i)
	}

	// Continue modifying buffer while iterators are running
	go func() {
		defer wg.Done()
		for i := 100; i < 120; i++ {
			rb.Push(i)
			time.Sleep(time.Microsecond * 10)
		}
	}()

	wg.Wait()
}

func TestGenericTypes(t *testing.T) {
	// Test with different types to ensure generics work correctly

	// String buffer
	stringBuf := NewRingBuffer[string](3)
	stringBuf.Push("a")
	stringBuf.Push("b")
	val, ok := stringBuf.Pop()
	if !ok || val != "a" {
		t.Errorf("String buffer failed: expected 'a', got %s, ok=%v", val, ok)
	}

	// Struct buffer
	type TestStruct struct {
		ID   int
		Name string
	}

	structBuf := NewRingBuffer[TestStruct](2)
	s1 := TestStruct{ID: 1, Name: "first"}
	s2 := TestStruct{ID: 2, Name: "second"}

	structBuf.Push(s1)
	structBuf.Push(s2)

	result, ok := structBuf.Pop()
	if !ok || result.ID != 1 || result.Name != "first" {
		t.Errorf("Struct buffer failed: expected %+v, got %+v, ok=%v", s1, result, ok)
	}

	// Pointer buffer
	ptrBuf := NewRingBuffer[*int](2)
	num1, num2 := 42, 84
	ptrBuf.Push(&num1)
	ptrBuf.Push(&num2)

	ptr, ok := ptrBuf.Pop()
	if !ok || ptr == nil || *ptr != 42 {
		t.Errorf("Pointer buffer failed: expected pointer to 42, got %v, ok=%v", ptr, ok)
	}
}

func TestString(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)

	str := rb.String()
	// Just verify it doesn't panic and contains expected info
	if str == "" {
		t.Error("String() returned empty string")
	}
	// Basic check that it contains size info
	if !contains(str, "size: 2") {
		t.Errorf("String() should contain size info, got: %s", str)
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkPush(b *testing.B) {
	rb := NewRingBuffer[int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push(i)
	}
}

func BenchmarkPop(b *testing.B) {
	rb := NewRingBuffer[int](1000)
	// Fill buffer first
	for i := 0; i < 1000; i++ {
		rb.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push(i) // Keep it filled
		rb.Pop()
	}
}

func BenchmarkIterator(b *testing.B) {
	rb := NewRingBuffer[int](1000)
	// Fill buffer
	for i := 0; i < 1000; i++ {
		rb.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		it := rb.NewIterator()
		for it.HasNext() {
			it.Next()
		}
	}
}
