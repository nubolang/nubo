package language

func TypeCheck(shouldBe ObjectComplexType, is ObjectComplexType) bool {
	if shouldBe != TypeAny && shouldBe != is {
		return false
	}
	return true
}
