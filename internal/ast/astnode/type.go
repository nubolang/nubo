package astnode

type NodeType string

const (
	NodeTypeImport NodeType = "import"

	NodeTypeFunction         NodeType = "function"
	NodeTypeInlineFunction   NodeType = "inline_function"
	NodeTypeFunctionArgument NodeType = "function_argument"
	NodeTypeFunctionCall     NodeType = "function_call"
	NodeTypeReturn           NodeType = "return"

	NodeTypeType        NodeType = "type"
	NodeTypeStruct      NodeType = "struct"
	NodeTypeStructField NodeType = "struct_field"
	NodeTypeImpl        NodeType = "impl"

	NodeTypeEvent         NodeType = "event"
	NodeTypeEventArgument NodeType = "event_argument"
	NodeTypeSubscribe     NodeType = "subscribe"
	NodeTypePublish       NodeType = "publish"

	NodeTypeExpression NodeType = "expression"
	NodeTypeOperator   NodeType = "operator"
	NodeTypeValue      NodeType = "value"

	NodeTypeIncrement NodeType = "increment"
	NodeTypeDecrement NodeType = "decrement"
	NodeTypeAssign    NodeType = "assign"

	NodeTypeVariableDecl NodeType = "variable_decl"

	NodeTypeElement            NodeType = "element"
	NodeTypeElementRawText     NodeType = "element_raw_text"
	NodeTypeElementAttribute   NodeType = "element_attribute"
	NodeTypeElementDynamicText NodeType = "element_dynamic_text"

	NodeTypeList      NodeType = "list"
	NodeTypeDict      NodeType = "dict"
	NodeTypeDictField NodeType = "dict_field"

	NodeTypeWhile  NodeType = "while"
	NodeTypeIf     NodeType = "if"
	NodeTypeElse   NodeType = "else"
	NodeTypeFor    NodeType = "for"
	NodeTypeSignal NodeType = "signal"
	NodeTypeTry    NodeType = "try"
)
