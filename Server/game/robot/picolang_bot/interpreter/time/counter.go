// Package time provides a logical clock for interpreter execution.
package time

// Counter advances by one on each Increment. Its zero value starts at zero.
// It is not safe for concurrent use.
type Counter struct {
	value uint64
}

// Value returns the current count without advancing it.
func (c *Counter) Value() uint64 {
	return c.value
}

// Increment advances the counter by one and returns the new count.
func (c *Counter) Increment() uint64 {
	c.value++
	return c.value
}
