package hash

import (
	"encoding/base64"

	"github.com/nubolang/nubo/native/n"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// bcrypt hash
var hashBcrypt = n.Function(
	n.Describe(
		n.Arg("password", TStringByte),
		n.Arg("cost", n.TInt, n.Int(bcrypt.DefaultCost, nil)),
	).Returns(n.TString),
	func(a *n.Args) (any, error) {
		password := toBytes(a.Name("password"))
		cost := int(a.Name("cost").Value().(int64))
		hash, err := bcrypt.GenerateFromPassword(password, cost)
		if err != nil {
			return nil, err
		}
		return string(hash), nil
	},
)

// bcrypt compare
var checkBcrypt = n.Function(
	n.Describe(
		n.Arg("password", TStringByte),
		n.Arg("hash", TStringByte),
	).Returns(n.TBool),
	func(a *n.Args) (any, error) {
		password := toBytes(a.Name("password"))
		hash := toBytes(a.Name("hash"))
		err := bcrypt.CompareHashAndPassword(hash, password)
		return err == nil, nil
	},
)

// Argon2id
var hashArgon2 = n.Function(
	n.Describe(n.Arg("password", TStringByte), n.Arg("salt", TStringByte)).Returns(n.TString),
	func(a *n.Args) (any, error) {
		password := toBytes(a.Name("password"))
		salt := toBytes(a.Name("salt"))

		hash := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
		return base64.RawStdEncoding.EncodeToString(hash), nil
	},
)
