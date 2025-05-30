package astnode

type NodeType string

const (
	NodeTypeImport NodeType = "import"

	NodeTypeFunction         NodeType = "function"
	NodeTypeFunctionArgument NodeType = "function_argument"
	NodeTypeFunctionCall     NodeType = "function_call"
	NodeTypeReturn           NodeType = "return"

	NodeTypeType        NodeType = "type"
	NodeTypeStruct      NodeType = "struct"
	NodeTypeStructField NodeType = "struct_field"

	NodeTypeEvent         NodeType = "event"
	NodeTypeEventArgument NodeType = "event_argument"

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
)
