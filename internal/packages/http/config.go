package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nubolang/nubo/language"
)

type headerSetting struct {
	values   []string
	disabled bool
}

type requestOptions struct {
	baseURL         *string
	body            language.Object
	hasBody         bool
	headers         map[string]headerSetting
	query           map[string][]string
	cookies         map[string]*http.Cookie
	timeout         *time.Duration
	followRedirects *bool
	cookieJar       *bool
}

func unwrap(obj language.Object) language.Object {
	for {
		wrapped, ok := obj.(*language.Any)
		if !ok {
			return obj
		}
		obj = wrapped.Data
	}
}

func isNil(obj language.Object) bool {
	obj = unwrap(obj)
	return obj == nil || obj.Type().Base() == language.ObjectTypeNil
}

func forEachDict(obj language.Object, name string, fn func(string, language.Object) error) error {
	if isNil(obj) {
		return nil
	}

	entries, ok := unwrap(obj).Value().(*language.OrderedMap)
	if !ok {
		return fmt.Errorf("%s must be a dictionary", name)
	}

	return entries.IterateErr(func(key, value language.Object) error {
		key = unwrap(key)
		if isNil(key) || key.Type().Base() != language.ObjectTypeString {
			return fmt.Errorf("%s keys must be strings", name)
		}
		return fn(key.String(), unwrap(value))
	})
}

func readOptions(obj language.Object, creating bool) (requestOptions, error) {
	opts := requestOptions{
		headers: make(map[string]headerSetting),
		query:   make(map[string][]string),
		cookies: make(map[string]*http.Cookie),
	}

	err := forEachDict(obj, "HTTP options", func(name string, value language.Object) error {
		switch name {
		case "baseUrl":
			if !creating {
				return fmt.Errorf("baseUrl can only be set when creating a client")
			}
			if isNil(value) {
				empty := ""
				opts.baseURL = &empty
				return nil
			}
			if value.Type().Base() != language.ObjectTypeString {
				return fmt.Errorf("baseUrl must be a string")
			}
			baseURL := value.String()
			opts.baseURL = &baseURL

		case "body":
			if creating {
				return fmt.Errorf("body is only valid for individual requests")
			}
			opts.body = value
			opts.hasBody = true

		case "headers":
			return forEachDict(value, "headers", func(key string, item language.Object) error {
				if key == "" || strings.ContainsAny(key, " \t\r\n:") {
					return fmt.Errorf("invalid HTTP header name %q", key)
				}
				key = http.CanonicalHeaderKey(key)
				if isNil(item) {
					opts.headers[key] = headerSetting{disabled: true}
					return nil
				}

				setting := headerSetting{}
				if list, ok := item.(*language.List); ok {
					for _, entry := range list.Data {
						if isNil(entry) {
							return fmt.Errorf("header %q cannot contain nil values", key)
						}
						setting.values = append(setting.values, unwrap(entry).String())
					}
				} else {
					setting.values = []string{item.String()}
				}
				opts.headers[key] = setting
				return nil
			})

		case "query":
			return forEachDict(value, "query", func(key string, item language.Object) error {
				if isNil(item) {
					opts.query[key] = nil
					return nil
				}
				if list, ok := item.(*language.List); ok {
					values := make([]string, 0, len(list.Data))
					for _, entry := range list.Data {
						if !isNil(entry) {
							values = append(values, unwrap(entry).String())
						}
					}
					opts.query[key] = values
					return nil
				}
				opts.query[key] = []string{item.String()}
				return nil
			})

		case "cookies":
			return forEachDict(value, "cookies", func(key string, item language.Object) error {
				if isNil(item) {
					opts.cookies[key] = nil
					return nil
				}
				cookie := &http.Cookie{Name: key, Value: item.String()}
				if err := cookie.Valid(); err != nil {
					return fmt.Errorf("invalid cookie %q: %w", key, err)
				}
				opts.cookies[key] = cookie
				return nil
			})

		case "timeout":
			if isNil(value) {
				return nil
			}
			seconds, ok := value.Value().(int64)
			if !ok || seconds < 0 || seconds > int64((1<<63-1)/int64(time.Second)) {
				return fmt.Errorf("timeout must be a non-negative integer in seconds")
			}
			timeout := time.Duration(seconds) * time.Second
			opts.timeout = &timeout

		case "cookieJar", "followRedirects":
			if isNil(value) {
				return nil
			}
			enabled, ok := value.Value().(bool)
			if !ok {
				return fmt.Errorf("%s must be a boolean", name)
			}
			if name == "cookieJar" {
				opts.cookieJar = &enabled
			} else {
				opts.followRedirects = &enabled
			}

		default:
			return fmt.Errorf("unknown HTTP option %q", name)
		}
		return nil
	})

	return opts, err
}

func mergeHeaders(dst map[string]headerSetting, src map[string]headerSetting) {
	for key, value := range src {
		dst[key] = value
	}
}

func mergeCookies(dst map[string]*http.Cookie, src map[string]*http.Cookie) {
	for key, value := range src {
		dst[key] = value
	}
}

func applyQuery(dst url.Values, src map[string][]string) {
	for key, values := range src {
		if values == nil {
			dst.Del(key)
		} else {
			dst[key] = append([]string{}, values...)
		}
	}
}
