package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"github.com/nubolang/nubo/version"
	"golang.org/x/net/publicsuffix"
)

const clientBucket = "http.client"

type clientState struct {
	defaults requestOptions
	jar      http.CookieJar
}

func NewInstance(dg *debug.Debug) (language.Object, error) {
	return newInstance(dg, requestOptions{})
}

func newInstance(dg *debug.Debug, defaults requestOptions) (language.Object, error) {
	instance, err := getClientType(dg).NewInstance()
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	instance.BucketSet(clientBucket, &clientState{defaults: defaults, jar: jar})
	if defaults.baseURL != nil {
		if err := instance.GetPrototype().SetObject(
			context.Background(), "baseUrl", n.String(*defaults.baseURL, dg),
		); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

func doRequest(ctx context.Context, self language.Object, method, target string, input language.Object) (language.Object, error) {
	instance, ok := self.(*language.StructInstance)
	if !ok {
		return nil, fmt.Errorf("HTTP request requires a client instance")
	}
	stored, ok := instance.BucketGet(clientBucket)
	if !ok {
		return nil, fmt.Errorf("HTTP client is not initialized")
	}
	state, ok := stored.(*clientState)
	if !ok || state == nil {
		return nil, fmt.Errorf("HTTP client state is invalid")
	}

	opts, err := readOptions(input, false)
	if err != nil {
		return nil, err
	}

	baseObj, _ := self.GetPrototype().GetObject(ctx, "baseUrl")
	baseURL := ""
	if !isNil(baseObj) {
		baseURL = unwrap(baseObj).String()
	}

	resolved, err := resolveURL(baseURL, target)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(resolved)
	if err != nil {
		return nil, err
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, fmt.Errorf("request URL must be an absolute HTTP or HTTPS URL")
	}

	query := parsedURL.Query()
	applyQuery(query, state.defaults.query)
	applyQuery(query, opts.query)
	parsedURL.RawQuery = query.Encode()

	headers := map[string]headerSetting{
		"User-Agent": {values: []string{"Nubo/" + version.Version}},
	}
	mergeHeaders(headers, state.defaults.headers)
	mergeHeaders(headers, opts.headers)

	contentType := ""
	contentHeader, explicitContentType := headers["Content-Type"]
	if explicitContentType && !contentHeader.disabled && len(contentHeader.values) != 0 {
		contentType = contentHeader.values[0]
	}

	if opts.hasBody {
		opts.body, err = resolveReadable(ctx, opts.body)
		if err != nil {
			return nil, err
		}
	}

	body, inferredType, err := encodeBody(opts.body, opts.hasBody, contentType)
	if err != nil {
		return nil, err
	}
	if inferredType != "" && (!explicitContentType || strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data")) {
		headers["Content-Type"] = headerSetting{values: []string{inferredType}}
	}

	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), parsedURL.String(), body)
	if err != nil {
		return nil, err
	}

	for name, setting := range headers {
		if setting.disabled || len(setting.values) == 0 {
			if name == "User-Agent" || name == "Accept-Encoding" {
				req.Header.Set(name, "")
			}
			continue
		}
		req.Header[name] = append([]string(nil), setting.values...)
	}
	if host := req.Header.Get("Host"); host != "" {
		req.Host = host
		req.Header.Del("Host")
	}

	timeout := 10 * time.Second
	followRedirects := true
	useJar := true
	if state.defaults.timeout != nil {
		timeout = *state.defaults.timeout
	}
	if state.defaults.followRedirects != nil {
		followRedirects = *state.defaults.followRedirects
	}
	if state.defaults.cookieJar != nil {
		useJar = *state.defaults.cookieJar
	}
	if opts.timeout != nil {
		timeout = *opts.timeout
	}
	if opts.followRedirects != nil {
		followRedirects = *opts.followRedirects
	}
	if opts.cookieJar != nil {
		useJar = *opts.cookieJar
	}

	cookies := make(map[string]*http.Cookie)
	mergeCookies(cookies, state.defaults.cookies)
	mergeCookies(cookies, opts.cookies)

	client := &http.Client{Timeout: timeout}
	if useJar {
		client.Jar = state.jar
		if len(cookies) != 0 {
			client.Jar = &requestCookieJar{
				base:      state.jar,
				origin:    parsedURL.Scheme + "://" + parsedURL.Host,
				overrides: cookies,
			}
		}
	} else {
		for _, cookie := range cookies {
			if cookie != nil {
				req.AddCookie(cookie)
			}
		}
	}
	if !followRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return createResp(resp, data, self.Debug())
}

// requestCookieJar overlays one request's cookies without mutating the session jar.
type requestCookieJar struct {
	base      http.CookieJar
	origin    string
	overrides map[string]*http.Cookie
}

func (j *requestCookieJar) Cookies(target *url.URL) []*http.Cookie {
	stored := j.base.Cookies(target)
	if target.Scheme+"://"+target.Host != j.origin {
		return stored
	}
	cookies := make([]*http.Cookie, 0, len(stored)+len(j.overrides))
	for _, cookie := range stored {
		if _, overridden := j.overrides[cookie.Name]; !overridden {
			cookies = append(cookies, cookie)
		}
	}
	for _, cookie := range j.overrides {
		if cookie != nil {
			cookies = append(cookies, cookie)
		}
	}
	return cookies
}

func (j *requestCookieJar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	j.base.SetCookies(target, cookies)
}

func resolveURL(baseURL, target string) (string, error) {
	candidate, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid request URL: %w", err)
	}
	if candidate.IsAbs() || baseURL == "" {
		return candidate.String(), nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	if (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", fmt.Errorf("baseUrl must be an absolute HTTP or HTTPS URL")
	}
	if candidate.Host != "" {
		return base.ResolveReference(candidate).String(), nil
	}

	if candidate.Path != "" {
		escapedBasePath := base.EscapedPath()
		base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(candidate.Path, "/")
		base.RawPath = strings.TrimRight(escapedBasePath, "/") + "/" + strings.TrimLeft(candidate.EscapedPath(), "/")
		base.RawQuery = candidate.RawQuery
		base.ForceQuery = candidate.ForceQuery
	} else if candidate.RawQuery != "" || candidate.ForceQuery {
		base.RawQuery = candidate.RawQuery
		base.ForceQuery = candidate.ForceQuery
	}
	base.Fragment = candidate.Fragment
	base.RawFragment = candidate.RawFragment
	return base.String(), nil
}

func encodeBody(body language.Object, present bool, contentType string) (io.Reader, string, error) {
	if !present {
		return nil, "", nil
	}
	body = unwrap(body)

	if reader, ok := binaryBody(body); ok {
		if contentType == "" {
			return reader, "application/octet-stream", nil
		}
		return reader, "", nil
	}

	mediaType := ""
	params := map[string]string{}
	if contentType != "" {
		var err error
		mediaType, params, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, "", fmt.Errorf("invalid Content-Type: %w", err)
		}
	}

	switch {
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"):
		data, err := encodeJSON(body)
		return bytes.NewReader(data), "", err

	case mediaType == "application/x-www-form-urlencoded":
		if !isNil(body) && body.Type().Base() == language.ObjectTypeString {
			return strings.NewReader(body.String()), "", nil
		}
		data, err := encodeForm(body)
		return strings.NewReader(data), "", err

	case mediaType == "multipart/form-data":
		if !isNil(body) && body.Type().Base() == language.ObjectTypeString {
			if params["boundary"] == "" {
				return nil, "", fmt.Errorf("raw multipart body requires a boundary in Content-Type")
			}
			return strings.NewReader(body.String()), "", nil
		}
		return encodeMultipart(body, params)

	case mediaType == "" && isStructured(body):
		data, err := encodeJSON(body)
		return bytes.NewReader(data), "application/json", err

	default:
		if isNil(body) {
			return nil, "", nil
		}
		if isStructured(body) {
			return nil, "", fmt.Errorf("cannot encode structured body as %q", contentType)
		}
		return strings.NewReader(body.String()), "", nil
	}
}

func isStructured(obj language.Object) bool {
	if isNil(obj) {
		return false
	}
	switch unwrap(obj).Type().Base() {
	case language.ObjectTypeDict, language.ObjectTypeList, language.ObjectTypeStructInstance:
		return true
	default:
		return false
	}
}

func binaryBody(obj language.Object) (io.Reader, bool) {
	if isNil(obj) {
		return nil, false
	}
	switch value := obj.Value().(type) {
	case io.Reader:
		return value, true
	case []byte:
		return bytes.NewReader(value), true
	}
	if list, ok := obj.(*language.List); ok && len(list.Data) != 0 {
		data := make([]byte, 0, len(list.Data))
		for _, element := range list.Data {
			if isNil(element) {
				return nil, false
			}
			value, ok := unwrap(element).Value().(uint8)
			if !ok {
				return nil, false
			}
			data = append(data, value)
		}
		return bytes.NewReader(data), true
	}
	return nil, false
}

func encodeJSON(obj language.Object) ([]byte, error) {
	if isNil(obj) {
		return []byte("null"), nil
	}
	value, err := language.ToValue(obj, true)
	if err != nil {
		return nil, fmt.Errorf("JSON body: %w", err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("JSON body: %w", err)
	}
	return data, nil
}

func encodeForm(body language.Object) (string, error) {
	values := make(url.Values)
	err := forEachDict(body, "form body", func(name string, item language.Object) error {
		if list, ok := item.(*language.List); ok {
			for _, entry := range list.Data {
				if err := addFormValue(values, name, entry); err != nil {
					return err
				}
			}
			return nil
		}
		return addFormValue(values, name, item)
	})
	return values.Encode(), err
}

func addFormValue(values url.Values, name string, item language.Object) error {
	if isNil(item) {
		return nil
	}
	if isStructured(item) {
		data, err := encodeJSON(item)
		if err != nil {
			return err
		}
		values.Add(name, string(data))
		return nil
	}
	values.Add(name, item.String())
	return nil
}

func encodeMultipart(body language.Object, params map[string]string) (io.Reader, string, error) {
	buffer := new(bytes.Buffer)
	writer := multipart.NewWriter(buffer)
	if boundary := params["boundary"]; boundary != "" {
		if err := writer.SetBoundary(boundary); err != nil {
			return nil, "", err
		}
	}

	err := forEachDict(body, "multipart body", func(name string, item language.Object) error {
		if list, ok := item.(*language.List); ok {
			if _, binary := binaryBody(item); !binary {
				for _, entry := range list.Data {
					if err := writeMultipartField(writer, name, entry); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return writeMultipartField(writer, name, item)
	})
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(buffer.Bytes()), writer.FormDataContentType(), nil
}

func writeMultipartField(writer *multipart.Writer, name string, value language.Object) error {
	if isNil(value) {
		return nil
	}

	if _, ok := unwrap(value).Value().(*language.OrderedMap); ok {
		filename, found, _ := dictField(value, "filename")
		data, hasData, _ := dictField(value, "data")
		if found && hasData {
			contentType, _, _ := dictField(value, "contentType")
			if isNil(filename) || filename.Type().Base() != language.ObjectTypeString {
				return fmt.Errorf("multipart field %q: filename must be a string", name)
			}
			mediaType := "application/octet-stream"
			if !isNil(contentType) {
				mediaType = contentType.String()
			}
			disposition := mime.FormatMediaType("form-data", map[string]string{
				"name": name, "filename": filename.String(),
			})
			if strings.ContainsAny(mediaType, "\r\n") {
				return fmt.Errorf("multipart field %q: invalid content type", name)
			}
			headers := textproto.MIMEHeader{
				"Content-Disposition": {disposition},
				"Content-Type":        {mediaType},
			}
			part, err := writer.CreatePart(headers)
			if err != nil {
				return err
			}
			reader, binary := binaryBody(data)
			if !binary {
				if isStructured(data) {
					encoded, err := encodeJSON(data)
					if err != nil {
						return err
					}
					reader = bytes.NewReader(encoded)
				} else if isNil(data) {
					reader = bytes.NewReader(nil)
				} else {
					reader = strings.NewReader(data.String())
				}
			}
			_, err = io.Copy(part, reader)
			return err
		}
	}

	part, err := writer.CreateFormField(name)
	if err != nil {
		return err
	}
	if reader, binary := binaryBody(value); binary {
		_, err = io.Copy(part, reader)
		return err
	}
	if isStructured(value) {
		data, err := encodeJSON(value)
		if err != nil {
			return err
		}
		_, err = part.Write(data)
		return err
	}
	_, err = io.WriteString(part, value.String())
	return err
}

func dictField(obj language.Object, name string) (language.Object, bool, error) {
	var value language.Object
	found := false
	err := forEachDict(obj, "dictionary", func(key string, item language.Object) error {
		if key == name {
			value = item
			found = true
		}
		return nil
	})
	return value, found, err
}

func resolveReadable(ctx context.Context, obj language.Object) (language.Object, error) {
	obj = unwrap(obj)
	if isNil(obj) {
		return obj, nil
	}

	proto := obj.GetPrototype()
	if proto == nil {
		return obj, nil
	}

	method, exists := proto.GetObject(ctx, "__read__")
	if !exists {
		return obj, nil
	}

	fn, ok := unwrap(method).(*language.Function)
	if !ok || len(fn.ArgTypes) != 0 {
		return nil, fmt.Errorf("__read__ must be a zero-argument function")
	}

	result, err := fn.Call(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("__read__ failed: %w", err)
	}

	return unwrap(result), nil
}
