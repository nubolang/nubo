package interpreter

import (
	"github.com/nubolang/nubo/language"
)

// ComplexVariable represents a variable that is outside of the current interpreter. (For example: otherFile.otherVariable)
type ComplexVariable struct {
	// Scope is the scope of the variable.
	Scope interface {
		GetObject(name string) (language.Object, bool)
		BindObject(name string, value language.Object, mutable bool) error
	}

	// Object is the object that the variable represents.
	Object language.Object
}
