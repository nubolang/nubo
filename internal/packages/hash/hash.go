package hash

import (
	"context"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewHash(dg *debug.Debug) language.Object {
	pkg := n.NewPackage("hash", dg)
	proto := pkg.GetPrototype()
	ctx := context.Background()

	proto.SetObject(ctx, "md5", hashMd5)
	proto.SetObject(ctx, "sha1", hashSha1)
	proto.SetObject(ctx, "sha256", hashSha256)
	proto.SetObject(ctx, "sha512", hashSha512)
	proto.SetObject(ctx, "blake3", hashBlake3)

	hashBcrypt.GetPrototype().SetObject(ctx, "compare", checkBcrypt)
	proto.SetObject(ctx, "bcrypt", hashBcrypt)
	proto.SetObject(ctx, "argon2", hashArgon2)

	return pkg
}
