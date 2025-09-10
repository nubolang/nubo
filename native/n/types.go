package n

import "github.com/nubolang/nubo/language"

var (
	TInt    = language.TypeInt
	TFloat  = language.TypeFloat
	TBool   = language.TypeBool
	TString = language.TypeString
	THtml   = language.TypeHtml
	TChar   = language.TypeChar
	TByte   = language.TypeByte
	TList   = language.TypeList
	TDict   = language.TypeDict
	TStruct = language.TypeStructInstance
	TNil    = language.TypeNil
	TAny    = language.TypeAny
	TVoid   = language.TypeVoid
)

func TTFn(returnType *language.Type, argsType ...*language.Type) *language.Type {
	return language.NewFunctionType(returnType, argsType...)
}

func TTList(elementType *language.Type) *language.Type {
	return language.NewListType(elementType)
}

func NewDictType(key *language.Type, value *language.Type) *language.Type {
	return language.NewDictType(key, value)
}

func Nullable(typ *language.Type) *language.Type {
	return language.Nullable(typ)
}

func TUnion(types ...*language.Type) *language.Type {
	return language.NewUnionType(types...)
}
