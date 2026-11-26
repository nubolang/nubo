package thread

import (
	"github.com/nubolang/nubo/language"
	"sync"
	"testing"
)

func TestPortalNonblocking(t *testing.T) {
	p := NewPortal(1)
	if _, ok, closed := p.TryReceive(); ok || closed {
		t.Fatal("empty")
	}
	if !p.TrySend(language.Nil) || p.TrySend(language.Nil) {
		t.Fatal("capacity")
	}
	p.Close()
	p.Close()
	value, ok, closed := p.TryReceive()
	if value != language.Nil || !ok || !closed {
		t.Fatal("nil message on closed portal")
	}
	if _, ok, closed = p.TryReceive(); ok || !closed {
		t.Fatal("closed")
	}
	if p.TrySend(language.Nil) {
		t.Fatal("send closed")
	}
}
func TestPortalConcurrentClose(t *testing.T) {
	for k := 0; k < 100; k++ {
		p := NewPortal(2)
		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for n := 0; n < 30; n++ {
					p.TrySend(language.Nil)
					p.TryReceive()
					p.Length()
					p.IsClosed()
				}
			}()
		}
		wg.Add(1)
		go func() { defer wg.Done(); _ = p.Send(language.Nil) }()
		p.Close()
		wg.Wait()
	}
}
func TestPortalDrainClosed(t *testing.T) {
	p := NewPortal(2)
	p.TrySend(language.Nil)
	p.TrySend(language.Nil)
	p.Close()
	if len(p.Drain()) != 2 || len(p.Drain()) != 0 {
		t.Fatal("drain")
	}
}
func BenchmarkPortalEmpty(b *testing.B) {
	p := NewPortal(16)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.TryReceive()
	}
}
func BenchmarkPortalRoundTrip(b *testing.B) {
	p := NewPortal(16)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.TrySend(language.Nil)
		p.TryReceive()
	}
}
