package parsers

import (
	"context"

	"github.com/nubolang/nubo/internal/ast/astnode"
)

func parseTemplateLiteral(_ context.Context, sn HTMLAttrValueParser, str *astnode.Node) (*astnode.Node, error) {
	content := splitWithDynamic(str.Value.(string))

	s := &astnode.Node{
		Type:    astnode.NodeTypeTemplateLiteral,
		Content: str.Value.(string),
		Debug:   str.Debug,
	}

	contentDebug := *str.Debug
	contentDebug.Column++ // skip the opening quote
	for _, part := range content {
		dg := debugOffset(&contentDebug, str.Value.(string), part.offset)
		if part.Dynamic {
			partNode, err := sn.ParseHTMLAttrValue(dg, part.Text)
			if err != nil {
				return nil, err
			}
			dynamic := &astnode.Node{
				Type:  astnode.NodeTypeDynamicText,
				Value: partNode,
				Debug: dg,
			}
			dynamic.Flags.Append("NODEVALUE")
			s.Children = append(s.Children, dynamic)
		} else {
			s.Children = append(s.Children, &astnode.Node{
				Type:    astnode.NodeTypeRawText,
				Content: part.Text,
				Debug:   dg,
			})
		}
	}

	return s, nil
}

type TemplatePart struct {
	Text    string
	Dynamic bool
	offset  int
}

func splitWithDynamic(s string) []TemplatePart {
	var parts []TemplatePart
	var buf []rune
	depth := 0
	var quote rune
	escape := false
	dynamicMode := false
	start := 0

	flush := func() {
		if len(buf) > 0 {
			parts = append(parts, TemplatePart{
				Text:    string(buf),
				Dynamic: dynamicMode,
				offset:  start,
			})
			buf = nil
		}
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if escape {
			buf = append(buf, r)
			escape = false
			continue
		}

		if quote != 0 { // inside string
			buf = append(buf, r)
			if r == '\\' {
				escape = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}

		if r == '"' || r == '\'' {
			buf = append(buf, r)
			quote = r
			continue
		}

		if r == '$' && i+1 < len(runes) && runes[i+1] == '{' {
			if depth == 0 {
				flush()
				dynamicMode = true
				start = i + 2
			} else {
				buf = append(buf, r)
			}
			depth++
			i++ // skip '{' since we already handled it
			continue
		}

		if dynamicMode && r == '{' {
			depth++
		}
		if dynamicMode && r == '}' {
			depth--
			if depth == 0 {
				flush()
				dynamicMode = false
				start = i + 1
				continue
			}
			buf = append(buf, r)
			continue
		}

		buf = append(buf, r)
	}

	flush()
	return parts
}
