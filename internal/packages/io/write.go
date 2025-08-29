package io

import (
	"os"

	"github.com/nubolang/nubo/native/n"
)

func writeFile(args *n.Args) (any, error) {
	file := args.Name("file").String()
	data := args.Name("data").String()
	perm := args.Name("perm").Value().(int64)

	return nil, os.WriteFile(file, []byte(data), os.FileMode(perm))
}
