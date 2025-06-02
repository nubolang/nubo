package io

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func openFn(ctx native.FnCtx) (language.Object, error) {
	fileName, err := ctx.Get("file")
	if err != nil {
		return nil, err
	}
	encodingName, err := ctx.Get("encoding")
	if err != nil {
		return nil, err
	}

	fileRealPath := fileName.Value().(string)
	if !filepath.IsAbs(fileRealPath) && fileName.Debug() != nil {
		fileRealPath = filepath.Join(filepath.Dir(fileName.Debug().File), fileRealPath)
	}

	enc, err := getEncoding(encodingName.Value().(string))
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fileRealPath)
	if err != nil {
		return nil, err
	}

	var reader io.Reader = file
	if enc != encoding.Nop {
		reader = transform.NewReader(file, enc.NewDecoder())
	}

	return NewIOStream(reader), nil
}

func getEncoding(name string) (encoding.Encoding, error) {
	switch strings.ToLower(name) {
	case "utf-8":
		return encoding.Nop, nil
	case "utf-16":
		return unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM), nil
	case "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	case "latin1", "iso-8859-1":
		return charmap.ISO8859_1, nil
	case "windows-1252", "cp1252":
		return charmap.Windows1252, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", name)
	}
}
