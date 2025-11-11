package thread

import (
	"fmt"
	"time"

	"github.com/nubolang/nubo/language"
)

type Portal struct {
	portal   chan language.Object
	capacity int
	closed   bool
}

func NewPortal(capacity int) *Portal {
	return &Portal{
		portal:   make(chan language.Object, capacity),
		capacity: capacity,
		closed:   false,
	}
}

func (p *Portal) Send(obj language.Object) {
	p.portal <- obj
}

func (p *Portal) Receive() language.Object {
	return <-p.portal
}

func (p *Portal) ReceiveWithTimeout(ms int) (language.Object, error) {
	select {
	case obj := <-p.portal:
		return obj, nil
	case <-time.After(time.Duration(ms) * time.Millisecond):
		return nil, fmt.Errorf("[thread/Portal] receive timeout")
	}
}

func (p *Portal) Close() {
	close(p.portal)
	p.closed = true
}
