package nubo

import (
	"net/http"

	"github.com/nubolang/nubo/language"
)

type Map map[string]any

type Renderer struct {
	err error

	object language.Object
}

func Render(code string, data Map) *Renderer {
	ctx := New()
	value, err := ctx.ExecString(code)
	renderer := &Renderer{object: value, err: err}
	return renderer
}

func (r *Renderer) Err() error {
	return r.err
}

func (r *Renderer) Value() any {
	if r.err != nil {
		return nil
	}
	return r.object.Value()
}

func (r *Renderer) String() string {
	if r.err != nil {
		return ""
	}
	return r.object.String()
}

func (r *Renderer) Response(w http.ResponseWriter, req *http.Request) error {
	_, err := w.Write([]byte(r.String()))
	return err
}
