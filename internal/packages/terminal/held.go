package terminal

import (
	"errors"
	"os"
	"time"

	"github.com/nubolang/nubo/language"
)

// setKeyEvents negotiates Kitty flags 1|2|8. Only a confirmed terminal response
// enables held state. Call on the owning interpreter thread, like pollKey.
func (s *session) setKeyEvents(enabled bool) (bool, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false, errors.New("terminal session is closed")
	}

	if !enabled {
		err := s.disableKeyEvents()
		s.mu.Unlock()
		return false, err
	}

	if s.keyEvents {
		s.mu.Unlock()
		return true, nil
	}

	if !s.raw {
		s.mu.Unlock()
		return false, errors.New("key events require raw mode")
	}

	if s.keyPushed {
		if err := s.disableKeyEvents(); err != nil {
			s.mu.Unlock()
			return false, err
		}
	}

	s.keyFlags = -1
	s.down = make(map[string]bool)
	s.keyPushed = true
	_, err := os.Stdout.WriteString("\x1b[>11u\x1b[?u")
	s.mu.Unlock()

	if err != nil {
		return false, err
	}

	deadline := time.Now().Add(150 * time.Millisecond)

	for time.Now().Before(deadline) {
		value, err := s.pollInput(time.Until(deadline).Milliseconds() + 1)
		if err != nil {
			s.mu.Lock()
			_ = s.disableKeyEvents()
			s.mu.Unlock()
			return false, err
		}

		if value != language.Nil {
			if len(s.queued) >= 256 {
				s.mu.Lock()
				_ = s.disableKeyEvents()
				s.mu.Unlock()
				return false, errors.New("too much input during keyboard negotiation")
			}
			s.queued = append(s.queued, value)
		}

		s.mu.Lock()
		if s.keyFlags >= 0 {
			if s.keyFlags&11 == 11 {
				s.keyEvents = true
				_, err = os.Stdout.WriteString("\x1b[?1004h")
				s.mu.Unlock()
				return true, err
			}

			err = s.disableKeyEvents()
			s.mu.Unlock()
			return false, err
		}

		s.mu.Unlock()
	}

	s.mu.Lock()
	err = s.disableKeyEvents()
	s.mu.Unlock()
	return false, err
}

// Caller holds mu. Pop on the same screen on which the mode was pushed.
func (s *session) disableKeyEvents() error {
	s.keyEvents = false
	clear(s.down)

	if !s.keyPushed {
		return nil
	}

	_, err := os.Stdout.WriteString("\x1b[<u\x1b[?1004l")
	if err == nil {
		s.keyPushed = false
	}

	return err
}
