package astnode

type NodeType string

const (
	NodeTypeImport NodeType = "import"

	NodeTypeFunction         NodeType = "function"
	NodeTypeFunctionArgument NodeType = "function_argument"

	NodeTypeType        NodeType = "type"
	NodeTypeStruct      NodeType = "struct"
	NodeTypeStructField NodeType = "struct_field"

	NodeTypeEvent         NodeType = "event"
	NodeTypeEventArgument NodeType = "event_argument"
)
