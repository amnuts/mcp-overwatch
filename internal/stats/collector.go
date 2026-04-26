package stats

import (
	"sync"
	"time"
)

// Collector buffers events and periodically flushes them to a Store.
type Collector struct {
	store    *Store
	buffer   []Event
	mu       sync.Mutex
	interval time.Duration
	stopCh   chan struct{}
	done     chan struct{}
}

// NewCollector creates a new Collector that flushes at the given interval.
func NewCollector(store *Store, flushInterval time.Duration) *Collector {
	c := &Collector{
		store:    store,
		interval: flushInterval,
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
	}
	go c.loop()
	return c
}

// Record adds an event to the buffer.
func (c *Collector) Record(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	c.mu.Lock()
	c.buffer = append(c.buffer, e)
	c.mu.Unlock()
}

// Flush writes all buffered events to the store.
func (c *Collector) Flush() error {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return nil
	}
	batch := c.buffer
	c.buffer = nil
	c.mu.Unlock()

	return c.store.InsertBatch(batch)
}

// Stop stops the periodic flush loop and flushes remaining events.
func (c *Collector) Stop() error {
	close(c.stopCh)
	<-c.done
	return c.Flush()
}

func (c *Collector) loop() {
	defer close(c.done)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Flush()
		case <-c.stopCh:
			return
		}
	}
}
