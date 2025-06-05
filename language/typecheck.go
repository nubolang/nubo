package language

import (
	"fmt"
	"strings"
)

func TypeCheck(shouldBe ObjectComplexType, is ObjectComplexType) bool {
	return shouldBe.Compare(is)
}

type UnionType struct {
	Types []ObjectComplexType
}

func NewUnionType(types ...ObjectComplexType) *UnionType {
	return &UnionType{Types: types}
}

func (ut *UnionType) Base() ObjectType {
	return TypeAny
}

func (ut *UnionType) String() string {
	var types = make([]string, len(ut.Types))
	for i, typ := range ut.Types {
		types[i] = typ.String()
	}

	return fmt.Sprintf("union(%s)", strings.Join(types, "|"))
}

func (ut *UnionType) Compare(other ObjectComplexType) bool {
	for _, t := range ut.Types {
		if t.Compare(other) {
			return true
		}
	}
	return false
}
