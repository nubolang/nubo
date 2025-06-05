package language

func TypeCheck(shouldBe *Type, is *Type) bool {
	return shouldBe.Compare(is)
}
