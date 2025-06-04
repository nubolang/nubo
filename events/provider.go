package events

import (
	"errors"
	"slices"
	"sync"
	"time"
)

const MaxWorkers = 10

type DefaultProvider struct {
	mu      sync.RWMutex
	events  []*Event
	subs    map[string][]chan TransportData
	buffer  map[string][]TransportData
	closed  bool
	workers int
}

func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{
		subs:   make(map[string][]chan TransportData),
		buffer: make(map[string][]TransportData),
	}
}

func (p *DefaultProvider) Events() []*Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	eventsCopy := make([]*Event, len(p.events))
	copy(eventsCopy, p.events)
	return eventsCopy
}

func (p *DefaultProvider) GetEvent(id string) *Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, e := range p.events {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func (p *DefaultProvider) AddEvent(e *Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *DefaultProvider) Publish(topic string, data TransportData) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("provider closed")
	}

	subs, ok := p.subs[topic]
	if !ok || len(subs) == 0 {
		return nil
	}

	sent := false
	for _, ch := range subs {
		select {
		case ch <- data:
			sent = true
		default:
		}
	}

	if !sent {
		p.buffer[topic] = append(p.buffer[topic], data)

		if p.workers < MaxWorkers {
			p.workers++
			go p.bufferWorker(topic)
		}
	}

	return nil
}

func (p *DefaultProvider) bufferWorker(topic string) {
	defer func() {
		p.mu.Lock()
		p.workers--
		p.mu.Unlock()
	}()

	for {
		p.mu.Lock()
		buf := p.buffer[topic]
		if len(buf) == 0 || p.closed {
			delete(p.buffer, topic)
			p.mu.Unlock()
			return
		}
		data := buf[0]
		p.buffer[topic] = buf[1:]
		subs := p.subs[topic]
		p.mu.Unlock()

		sent := false
		for _, ch := range subs {
			select {
			case ch <- data:
				sent = true
			default:
			}
		}

		if !sent {
			p.mu.Lock()
			p.buffer[topic] = append([]TransportData{data}, p.buffer[topic]...)
			p.mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (p *DefaultProvider) Subscribe(topic string, handler func(TransportData)) (UnsubscribeFunc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("provider closed")
	}

	ch := make(chan TransportData, 10)
	p.subs[topic] = append(p.subs[topic], ch)

	go func() {
		for msg := range ch {
			handler(msg)
		}
	}()

	unsub := func() error {
		p.mu.Lock()
		defer p.mu.Unlock()

		subs := p.subs[topic]
		for i, c := range subs {
			if c == ch {
				p.subs[topic] = slices.Delete(subs, i, i+1)
				close(c)

				if len(p.subs[topic]) == 0 {
					delete(p.subs, topic)
					delete(p.buffer, topic)
				}
				return nil
			}
		}
		return errors.New("channel not found")
	}

	return unsub, nil
}

func (p *DefaultProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	for _, chans := range p.subs {
		for _, ch := range chans {
			close(ch)
		}
	}
	p.subs = nil
	p.buffer = nil
	return nil
}
