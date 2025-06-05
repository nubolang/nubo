package interpreter

import (
	"errors"
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
)

func newErr(base error, err string, d ...*debug.Debug) error {
	if len(d) > 0 {
		return debug.NewError(base, err, d[0])
	}

	return fmt.Errorf("%w: %v", base, err)
}

func isTypeErr(err error) bool {
	var de *debug.DebugErr
	if errors.As(err, &de) {
		return de.Unwrap() == ErrTypeMismatch
	}

	return err == ErrTypeMismatch
}

var (
	ErrUnknownNode       = fmt.Errorf("Unknown node")
	ErrImmutableVariable = fmt.Errorf("Variable is immutable")
	ErrExpression        = fmt.Errorf("Expression error")
	ErrImportError       = fmt.Errorf("Import error")
	ErrUndefinedVariable = fmt.Errorf("Undefined variable")
	ErrUnsupported       = fmt.Errorf("Unsupported operation")
	ErrUndefinedFunction = fmt.Errorf("Undefined function")
	ErrExpectedFunction  = fmt.Errorf("Expected function")
	ErrTypeMismatch      = fmt.Errorf("Type mismatch")
	ErrAst               = fmt.Errorf("Ast error")
	ErrValueError        = fmt.Errorf("Value error")
	ErrInvalid           = fmt.Errorf("Invalid syntax")
)
