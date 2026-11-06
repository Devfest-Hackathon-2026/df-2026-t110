// Package event collects interpreter events for delivery to the recorder.
package event

import robotevent "github.com/dvpico/Programming-Learning-AR/Server/game/protocol/robot_event"

// Collector buffers one interpreter's events in emission order. Its zero value
// is ready to use. Calls must be sequential; events must not be modified after Add.
type Collector struct {
	events []robotevent.PicolangEvent
}

// Add appends an event to the pending batch.
func (c *Collector) Add(event robotevent.PicolangEvent) {
	c.events = append(c.events, event)
}

// Collect transfers all pending events to the caller and empties the collector.
// It returns nil when empty. Later additions do not modify the returned batch,
// which can be assigned directly to RobotOutput.Events for delivery via the hub.
func (c *Collector) Collect() []robotevent.PicolangEvent {
	events := c.events
	c.events = nil
	return events
}
