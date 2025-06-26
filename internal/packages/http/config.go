package http

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var configStruct *language.Struct

func config(dg *debug.Debug) *language.Struct {
	if configStruct == nil {
		configStruct = language.NewStruct("config", []language.StructField{
			{Name: "body", Type: n.TAny},
			{Name: "headers", Type: n.NewDictType(n.TString, n.TAny)},
			{Name: "timeout", Type: n.TInt},
		}, dg)
	}

	return configStruct
}
