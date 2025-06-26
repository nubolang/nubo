package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"github.com/nubolang/nubo/version"
)

func NewInstance(dg *debug.Debug) (language.Object, error) {
	inst, err := httpStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	proto := inst.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	proto.SetObject("request", n.Function(n.Describe(n.Arg("method", n.TString), n.Arg("url", n.TString), n.Arg("config", config(dg).Type(), language.Nil)).Returns(getResp(dg).Type()), func(a *n.Args) (any, error) {
		base, _ := proto.GetObject("baseUrl")
		baseUrl := base.String()

		method := a.Name("method").String()
		url := a.Name("url").String()

		if baseUrl != "" && !isAbsoluteUrl(url) {
			newUrl, err := joinURL(baseUrl, url)
			if err != nil {
				return nil, err
			}
			url = newUrl
		}

		var body io.Reader
		var headers map[string]any = make(map[string]any)
		headers["User-Agent"] = fmt.Sprintf("Nubo/%s", version.Version)

		timeout := 10 * time.Second

		config := a.Name("config")
		if config.Type().Base() != language.ObjectTypeNil {
			configProto := config.GetPrototype()
			if bd, ok := configProto.GetObject("body"); ok {
				if bd.Type().Base() != language.ObjectTypeNil {
					body = strings.NewReader(bd.String())
				}
			}

			if tm, ok := configProto.GetObject("timeout"); ok {
				if tm.Type().Base() != language.ObjectTypeNil {
					tmInt := tm.Value().(int64)
					if tmInt != 0 {
						timeout = time.Duration(tmInt) * time.Second
					}
				}
			}

			if hd, ok := configProto.GetObject("headers"); ok {
				if hd.Type().Base() != language.ObjectTypeNil {
					hds := hd.Value().(map[language.Object]language.Object)
					for k, v := range hds {
						headers[k.String()] = v.String()
					}
				}
			}
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, err
		}

		// Add headers
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
			}
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		bodyb, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return createResp(struct {
			Url     string
			Status  int
			Headers http.Header
			Body    []byte
		}{
			Url:     url,
			Status:  resp.StatusCode,
			Headers: resp.Header,
			Body:    bodyb,
		}, dg)
	}))

	return inst, nil
}

func isAbsoluteUrl(u string) bool {
	parsed, err := url.Parse(u)
	return err == nil && parsed.IsAbs()
}

func joinURL(baseStr, relStr string) (string, error) {
	base, err := url.Parse(baseStr)
	if err != nil {
		return "", err
	}

	// clean base path
	basePath := strings.TrimSuffix(base.Path, "/")

	// clean rel path
	relStr = strings.TrimPrefix(relStr, "/")

	// join
	base.Path = basePath + "/" + relStr

	return base.String(), nil
}
