package pubsub

import (
	"errors"
	"slices"
	"sync"
)

type DefaultProvider struct {
	mu     sync.RWMutex
	events []Event
	subs   map[string][]chan TransportData
	queues map[string]chan TransportData
	closed bool
}

func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{
		subs:   make(map[string][]chan TransportData),
		queues: make(map[string]chan TransportData),
	}
}

func (p *DefaultProvider) Events() []Event {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return slices.Clone(p.events)
}

func (p *DefaultProvider) AddEvent(e Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *DefaultProvider) Publish(topic string, data TransportData) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("provider closed")
	}
	queue, ok := p.queues[topic]
	p.mu.RUnlock()

	if !ok {
		p.mu.Lock()
		// kettős ellenőrzés
		queue, ok = p.queues[topic]
		if !ok {
			queue = make(chan TransportData, 100)
			p.queues[topic] = queue
			go p.startQueueDispatcher(topic, queue)
		}
		p.mu.Unlock()
	}

	select {
	case queue <- data:
		return nil
	default:
		return errors.New("queue full")
	}
}

func (p *DefaultProvider) startQueueDispatcher(topic string, queue chan TransportData) {
	for data := range queue {
		p.mu.RLock()
		chans := p.subs[topic]
		p.mu.RUnlock()

		var wg sync.WaitGroup
		wg.Add(len(chans))

		for _, ch := range chans {
			go func(c chan TransportData) {
				defer wg.Done()
				c <- data
			}(ch)
		}

		wg.Wait()
	}
}

func (p *DefaultProvider) Subscribe(topic string, handler func(TransportData)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("provider closed")
	}
	ch := make(chan TransportData, 10)
	p.subs[topic] = append(p.subs[topic], ch)

	go func() {
		for msg := range ch {
			handler(msg)
		}
	}()
	return nil
}

func (p *DefaultProvider) Unsubscribe(topic string, handlerCh chan TransportData) {
	p.mu.Lock()
	defer p.mu.Unlock()

	chans := p.subs[topic]
	for i, ch := range chans {
		if ch == handlerCh {
			close(ch)
			p.subs[topic] = slices.Delete(p.subs[topic], i, i+1)
			break
		}
	}

	// Delete the channel if there are no more subscribers
	if len(p.subs[topic]) == 0 {
		if queue, ok := p.queues[topic]; ok {
			close(queue)
			delete(p.queues, topic)
		}
		delete(p.subs, topic)
	}
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

	for _, queue := range p.queues {
		close(queue)
	}

	p.subs = nil
	p.queues = nil
	return nil
}
