package events

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nubolang/nubo/config"
)

type DefaultProvider struct {
	eventsMu sync.RWMutex
	events   []*Event

	topics sync.Map // map[string]*topicState
	closed atomic.Bool
}

type topicState struct {
	mu      sync.Mutex
	subs    []chan TransportData
	buffer  []TransportData
	workers int
}

func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{}
}

func (p *DefaultProvider) AddEvent(e *Event) {
	p.eventsMu.Lock()
	defer p.eventsMu.Unlock()
	p.events = append(p.events, e)
}

func (p *DefaultProvider) Events() []*Event {
	p.eventsMu.RLock()
	defer p.eventsMu.RUnlock()
	return slices.Clone(p.events)
}

func (p *DefaultProvider) GetEvent(id string) *Event {
	p.eventsMu.RLock()
	defer p.eventsMu.RUnlock()
	for _, e := range p.events {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func (p *DefaultProvider) getOrCreateTopic(topic string) *topicState {
	actual, _ := p.topics.LoadOrStore(topic, &topicState{})
	return actual.(*topicState)
}

func (p *DefaultProvider) Publish(topic string, data TransportData) error {
	if p.closed.Load() {
		return errors.New("provider closed")
	}

	ts := p.getOrCreateTopic(topic)

	ts.mu.Lock()
	defer ts.mu.Unlock()

	allSent := true
	for _, ch := range ts.subs {
		select {
		case ch <- data:
		default:
			allSent = false
		}
	}

	if allSent {
		return nil
	}

	ts.buffer = append(ts.buffer, data)
	if ts.workers < config.Current.Runtime.Events.MaxWorkersPerTopic {
		ts.workers++
		go p.bufferWorker(ts)
	}

	return nil
}

func (p *DefaultProvider) bufferWorker(ts *topicState) {
	backoff := 10 * time.Millisecond

	for {
		ts.mu.Lock()
		if p.closed.Load() || len(ts.buffer) == 0 {
			ts.workers--
			ts.mu.Unlock()
			return
		}

		data := ts.buffer[0]
		ts.buffer = slices.Delete(ts.buffer, 0, 1)
		subs := slices.Clone(ts.subs)
		ts.mu.Unlock()

		allSent := true
		for _, ch := range subs {
			select {
			case ch <- data:
			default:
				allSent = false
			}
		}

		if !allSent {
			ts.mu.Lock()
			ts.buffer = append([]TransportData{data}, ts.buffer...)
			ts.mu.Unlock()
			time.Sleep(backoff)
			if backoff < 200*time.Millisecond {
				backoff *= 2
			}
		} else {
			backoff = 10 * time.Millisecond
		}
	}
}

func (p *DefaultProvider) Subscribe(topic string, handler func(TransportData)) (UnsubscribeFunc, error) {
	if p.closed.Load() {
		return nil, errors.New("provider closed")
	}

	ts := p.getOrCreateTopic(topic)
	ch := make(chan TransportData, config.Current.Runtime.Events.ChannelBufferSize)

	ts.mu.Lock()
	ts.subs = append(ts.subs, ch)
	ts.mu.Unlock()

	go func() {
		for msg := range ch {
			handler(msg)
		}
	}()

	return func() error {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		for i, c := range ts.subs {
			if c == ch {
				ts.subs = slices.Delete(ts.subs, i, i+1)
				close(ch)
				return nil
			}
		}
		return errors.New("channel not found")
	}, nil
}

func (p *DefaultProvider) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.topics.Range(func(_, v any) bool {
		ts := v.(*topicState)
		ts.mu.Lock()
		for _, ch := range ts.subs {
			close(ch)
		}
		ts.subs = nil
		ts.buffer = nil
		ts.mu.Unlock()
		return true
	})

	return nil
}
