package terminal

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type keyEvent struct {
	Key, Text        string
	Ctrl, Alt, Shift bool
	Type             string
	protocol         bool
	flags            int
	identity         string
}

func (k keyEvent) value() map[string]any {
	kind := k.Type
	if kind == "" {
		kind = "press"
	}
	return map[string]any{"action": kind, "key": k.Key, "text": k.Text, "ctrl": k.Ctrl, "alt": k.Alt, "shift": k.Shift}
}

// decode preserves incomplete UTF-8 and escape sequences until more input arrives.
func decode(data []byte, expired bool) (keyEvent, int) {
	if len(data) == 0 {
		return keyEvent{}, 0
	}
	if data[0] == 27 {
		if len(data) == 1 {
			if expired {
				return keyEvent{Key: "escape"}, 1
			}
			return keyEvent{}, 0
		}
		if data[1] == '[' || data[1] == 'O' {
			end := 2
			for end < len(data) && (data[end] < 0x40 || data[end] > 0x7e) {
				end++
			}
			if end == len(data) {
				if expired {
					return keyEvent{Key: "escape"}, 1
				}
				return keyEvent{}, 0
			}
			final := data[end]
			body := string(data[2:end])
			if final == 'u' && strings.HasPrefix(body, "?") {
				flags, err := strconv.Atoi(body[1:])
				if err != nil {
					flags = -1
				}
				return keyEvent{Key: "protocol", flags: flags}, end + 1
			}
			if (final == 'I' || final == 'O') && body == "" {
				return keyEvent{Key: "focus", Type: map[bool]string{true: "in", false: "out"}[final == 'I']}, end + 1
			}

			params := strings.Split(string(data[2:end]), ";")
			eventType := "press"
			modifier := 1
			if len(params) > 1 {
				fields := strings.Split(params[1], ":")
				modifier, _ = strconv.Atoi(fields[0])
				if len(fields) > 1 {
					switch fields[1] {
					case "2":
						eventType = "repeat"
					case "3":
						eventType = "release"
					}
				}
			}
			if final == 'u' {
				code, err := strconv.Atoi(strings.Split(params[0], ":")[0])
				if err != nil || code < 0 || code > utf8.MaxRune {
					return keyEvent{Key: "unknown"}, end + 1
				}
				key := map[int]string{27: "escape", 13: "enter", 9: "tab", 127: "backspace", 57344: "escape", 57345: "enter", 57346: "tab", 57347: "backspace", 57348: "insert", 57349: "delete", 57350: "left", 57351: "right", 57352: "up", 57353: "down", 57354: "pageUp", 57355: "pageDown", 57356: "home", 57357: "end"}[code]
				if code >= 57364 && code <= 57375 {
					key = "f" + strconv.Itoa(code-57363)
				}
				text := ""
				identity := key
				if key == "" {
					if code >= 57344 && code <= 63743 {
						key = "unknown"
					} else {
						key = "text"
						text = string(rune(code))
						identity = text
					}
				}
				event := keyEvent{Key: key, Text: text, Type: eventType, protocol: true, identity: identity}
				bits := modifier - 1
				if bits >= 0 {
					event.Shift = bits&1 != 0
					event.Alt = bits&2 != 0
					event.Ctrl = bits&4 != 0
				}
				return event, end + 1
			}
			key := map[byte]string{'A': "up", 'B': "down", 'C': "right", 'D': "left", 'H': "home", 'F': "end", 'P': "f1", 'Q': "f2", 'R': "f3", 'S': "f4", 'Z': "tab"}[final]
			if final == '~' {
				key = map[string]string{"1": "home", "2": "insert", "3": "delete", "4": "end", "5": "pageUp", "6": "pageDown", "11": "f1", "12": "f2", "13": "f3", "14": "f4", "15": "f5", "17": "f6", "18": "f7", "19": "f8", "20": "f9", "21": "f10", "23": "f11", "24": "f12"}[params[0]]
			}
			if key == "" {
				key = "unknown"
			}
			event := keyEvent{Key: key, Shift: final == 'Z', Type: eventType, protocol: data[1] == '[', identity: key}
			bits := modifier - 1
			if bits >= 0 {
				event.Shift = event.Shift || bits&1 != 0
				event.Alt = bits&2 != 0
				event.Ctrl = bits&4 != 0
			}

			return event, end + 1
		}
		event, n := decode(data[1:], expired)
		if n == 0 {
			return event, 0
		}
		event.Alt = true
		return event, n + 1
	}
	switch data[0] {
	case 13, 10:
		return keyEvent{Key: "enter"}, 1
	case 9:
		return keyEvent{Key: "tab"}, 1
	case 127, 8:
		return keyEvent{Key: "backspace"}, 1
	}

	if data[0] < 32 {
		return keyEvent{Key: string(rune(data[0] + 96)), Ctrl: true}, 1
	}

	if !utf8.FullRune(data) {
		return keyEvent{}, 0
	}

	r, n := utf8.DecodeRune(data)
	return keyEvent{Key: "text", Text: string(r)}, n
}
