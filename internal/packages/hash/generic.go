package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"github.com/zeebo/blake3"
)

var TStringByte = n.TUnion(n.TString, n.TTList(n.TByte))

// helper
func toBytes(value language.Object) ([]byte, error) {
	if value.Type() == language.TypeString {
		return []byte(value.String()), nil
	}

	listValues, ok := value.Value().([]language.Object)
	if !ok {
		return nil, fmt.Errorf("expected []byte-like list, got %s", value.Type())
	}

	bytes := make([]byte, len(listValues))
	for i, b := range listValues {
		byteObj, ok := b.(*language.Byte)
		if !ok {
			return nil, fmt.Errorf("list item %d is not byte, got %s", i, b.Type())
		}
		bytes[i] = byteObj.Data
	}
	return bytes, nil
}

// MD5
var hashMd5 = n.Function(
	n.Describe(n.Arg("value", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		bytes, err := toBytes(a.Name("value"))
		if err != nil {
			return nil, err
		}
		sum := md5.Sum(bytes)
		return hex.EncodeToString(sum[:]), nil
	},
)

// SHA-1
var hashSha1 = n.Function(
	n.Describe(n.Arg("value", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		bytes, err := toBytes(a.Name("value"))
		if err != nil {
			return nil, err
		}
		sum := sha1.Sum(bytes)
		return hex.EncodeToString(sum[:]), nil
	},
)

// SHA-256
var hashSha256 = n.Function(
	n.Describe(n.Arg("value", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		bytes, err := toBytes(a.Name("value"))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(bytes)
		return hex.EncodeToString(sum[:]), nil
	},
)

// SHA-512
var hashSha512 = n.Function(
	n.Describe(n.Arg("value", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		bytes, err := toBytes(a.Name("value"))
		if err != nil {
			return nil, err
		}
		sum := sha512.Sum512(bytes)
		return hex.EncodeToString(sum[:]), nil
	},
)

// BLAKE3
var hashBlake3 = n.Function(
	n.Describe(n.Arg("value", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		bytes, err := toBytes(a.Name("value"))
		if err != nil {
			return nil, err
		}
		sum := blake3.Sum256(bytes)
		return hex.EncodeToString(sum[:]), nil
	},
)
