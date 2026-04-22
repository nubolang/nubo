package astnode

type NodeType int

const (
	NodeTypeImport NodeType = iota
	NodeTypeInclude
	NodeTypeFunction
	NodeTypeInlineFunction
	NodeTypeFunctionArgument
	NodeTypeFunctionCall
	NodeTypeReturn
	NodeTypeType
	NodeTypeStruct
	NodeTypeStructField
	NodeTypeImpl
	NodeTypeTypeKW
	NodeTypeEvent
	NodeTypeEventArgument
	NodeTypeSubscribe
	NodeTypePublish
	NodeTypeExpression
	NodeTypeOperator
	NodeTypeValue
	NodeTypeIncrement
	NodeTypeDecrement
	NodeTypeAssign
	NodeTypeVariableDecl
	NodeTypeElement
	NodeTypeElementRawText
	NodeTypeElementAttribute
	NodeTypeElementDynamicText
	NodeTypeList
	NodeTypeDict
	NodeTypeDictField
	NodeTypeWhile
	NodeTypeIf
	NodeTypeElse
	NodeTypeFor
	NodeTypeSignal
	NodeTypeTry
	NodeTypeTemplateLiteral
	NodeTypeRawText
	NodeTypeDynamicText
	NodeTypeDefer
	NodeTypeSpawn
	NodeTypeBlock
)
