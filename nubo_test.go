package nubo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Default(t *testing.T) {
	inst := New()
	obj, err := inst.ExecString(`
		let x = 5
		return x
	`)

	assert.NoError(t, err, "Execute error should be nil")
	assert.Equal(t, int64(5), obj.Value(), "Value should be 5")
}

func Test_Html(t *testing.T) {
	inst := New()
	obj, err := inst.ExecString(`
		return (<div :id="4">
			Hello, World
		</div>)
	`)

	assert.NoError(t, err, "Execute error should be nil")
	fmt.Println(obj.Value())
}
