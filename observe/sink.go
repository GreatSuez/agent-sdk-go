package observe

import (
	"context"
	"sync"
	"time"
)

type Sink interface {
	Emit(ctx context.Context, event Event) error
}

type SinkFunc func(ctx context.Context, event Event) error

func (f SinkFunc) Emit(ctx context.Context, event Event) error {
	if f == nil {
		return nil
	}
	return f(ctx, event)
}

type NoopSink struct{}

func (NoopSink) Emit(ctx context.Context, event Event) error {
	_ = ctx
	_ = event
	return nil
}

type MultiSink struct {
	sinks []Sink
}

func NewMultiSink(sinks ...Sink) Sink {
	filtered := make([]Sink, 0, len(sinks))
	for _, s := range sinks {
		if s == nil {
			continue
		}
		filtered = append(filtered, s)
	}
	if len(filtered) == 0 {
		return NoopSink{}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return &MultiSink{sinks: filtered}
}

func (m *MultiSink) Emit(ctx context.Context, event Event) error {
	if m == nil {
		return nil
	}
	for _, sink := range m.sinks {
		if err := sink.Emit(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

type AsyncSink struct {
	downstream Sink
	queue      chan Event
	done       chan struct{}
	wg         sync.WaitGroup
	once       sync.Once
}

func NewAsyncSink(downstream Sink, buffer int) *AsyncSink {
	if downstream == nil {
		downstream = NoopSink{}
	}
	if buffer <= 0 {
		buffer = 256
	}
	as := &AsyncSink{
		downstream: downstream,
		queue:      make(chan Event, buffer),
		done:       make(chan struct{}),
	}
	as.wg.Add(1)
	go as.loop()
	return as
}

func (s *AsyncSink) Emit(ctx context.Context, event Event) error {
	if s == nil {
		return nil
	}
	event.Normalize()
	select {
	case <-s.done:
		return nil // sink is closing, drop silently
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return nil
	case s.queue <- event:
		return nil
	default:
		// Drop on pressure to avoid blocking runtime hot path.
		return nil
	}
}

func (s *AsyncSink) Close() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		close(s.done)  // signal loop to drain and exit
		close(s.queue) // unblock range loop
		s.wg.Wait()    // wait for loop goroutine to finish
	})
}

func (s *AsyncSink) loop() {
	defer s.wg.Done()
	for event := range s.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = s.downstream.Emit(ctx, event)
		cancel()
	}
}
