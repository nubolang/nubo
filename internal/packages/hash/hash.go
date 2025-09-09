package hash

import (
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewHash(dg *debug.Debug) language.Object {
	pkg := n.NewPackage("hash", dg)
	proto := pkg.GetPrototype()

	proto.SetObject("md5", hashMd5)
	proto.SetObject("sha1", hashSha1)
	proto.SetObject("sha256", hashSha256)
	proto.SetObject("sha512", hashSha512)
	proto.SetObject("blake3", hashBlake3)

	hashBcrypt.GetPrototype().SetObject("compare", checkBcrypt)
	proto.SetObject("bcrypt", hashBcrypt)
	proto.SetObject("argon2", hashArgon2)

	return pkg
}
