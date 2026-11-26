package thread

import (
	"fmt"
	"sync"
	"time"

	"github.com/nubolang/nubo/language"
)

// Portal serializes channel closure with sends. done wakes blocked senders
// before Close waits for their read locks.
type Portal struct {
	portal chan language.Object
	done   chan struct{}
	once   sync.Once
	mu     sync.RWMutex
}

func NewPortal(capacity int) *Portal {
	return &Portal{
		portal: make(chan language.Object, capacity),
		done:   make(chan struct{}),
	}
}

func (p *Portal) IsClosed() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

func (p *Portal) Send(obj language.Object) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.IsClosed() {
		return fmt.Errorf("[thread/Portal] portal is closed")
	}

	select {
	case p.portal <- obj:
		return nil
	case <-p.done:
		return fmt.Errorf("[thread/Portal] portal is closed")
	}
}

func (p *Portal) TrySend(obj language.Object) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.IsClosed() {
		return false
	}

	select {
	case p.portal <- obj:
		return true
	default:
		return false
	}
}

func (p *Portal) Receive() language.Object {
	return <-p.portal
}

func (p *Portal) TryReceive() (language.Object, bool, bool) {
	select {
	case obj, ok := <-p.portal:
		return obj, ok, p.IsClosed()
	default:
		return nil, false, p.IsClosed()
	}
}

func (p *Portal) Length() int {
	return len(p.portal)
}

func (p *Portal) Drain() []language.Object {
	length := p.Length()
	result := make([]language.Object, 0, length)
	for remaining := length; remaining > 0; remaining-- {
		obj, ok, _ := p.TryReceive()
		if !ok {
			break
		}
		result = append(result, obj)
	}
	return result
}

func (p *Portal) ReceiveWithTimeout(ms int) (language.Object, error) {
	if ms < 0 || int64(ms) > int64((1<<63-1)/time.Millisecond) {
		return nil, fmt.Errorf("[thread/Portal] invalid timeout")
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case obj, ok := <-p.portal:
		if !ok {
			return nil, fmt.Errorf("[thread/Portal] portal is closed")
		}
		return obj, nil
	case <-timer.C:
		return nil, fmt.Errorf("[thread/Portal] receive timeout")
	}
}

func (p *Portal) Close() {
	p.once.Do(func() {
		close(p.done)
		p.mu.Lock()
		defer p.mu.Unlock()
		close(p.portal)
	})
}

func (p *Portal) ReceiveMessage() (language.Object, error) {
	value, ok := <-p.portal
	if !ok {
		return nil, fmt.Errorf("[thread/Portal] portal is closed")
	}
	return value, nil
}
