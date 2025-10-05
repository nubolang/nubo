package codehighlight

import (
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/internal/lexer"
)

func (h *Highlight) highlightHtmlCode(mode Mode, tokens []*lexer.Token) (string, error) {
	var (
		sb        strings.Builder
		insideTag bool
		tagName   bool
		cTagName  string
	)

	for i, token := range tokens {
		if token.Type == lexer.TokenLessThan {
			insideTag = true
		}

		if token.Type == lexer.TokenLessThan || token.Type == lexer.TokenClosingStartTag {
			tagName = true
		}

		if insideTag && token.Type == lexer.TokenColon {
			switch mode {
			case ModeHTML:
				sb.WriteString(fmt.Sprintf("<span style=\"color:#ffd230\">%s</span>", html.EscapeString(token.Value)))
			case ModeConsole:
				sb.WriteString(color.New(color.FgHiYellow).Sprintf("%s", token.Value))
			}
			continue
		}

		if insideTag && (token.Type == lexer.TokenSelfClosingTag || token.Type == lexer.TokenGreaterThan) {
			insideTag = false
		}

		if token.Type == lexer.TokenIdentifier {
			if tagName {
				cTagName += token.Value
				if i+1 < len(tokens) && tokens[i+1].Type == lexer.TokenDot {
					cTagName += "."
					continue
				}

				parts := strings.Split(cTagName, ".")
				tagName = false
				cTagName = ""
				lastPart := parts[len(parts)-1]
				endsWithCapital := unicode.IsUpper(rune(lastPart[0]))

				for _, part := range parts[:len(parts)-1] {
					sb.WriteString(highlightIdentifier(mode, part))
					if mode == ModeHTML {
						sb.WriteString("<span style=\"color:#d1d5dc\">.</span>")
					} else {
						sb.WriteString(color.New(color.FgHiBlack).Sprint("."))
					}
				}

				switch mode {
				case ModeHTML:
					if endsWithCapital {
						sb.WriteString(fmt.Sprintf("<span style=\"color:#ad46ff\">%s</span>", html.EscapeString(lastPart)))
					} else {
						sb.WriteString(fmt.Sprintf("<span style=\"color:#2b7fff\">%s</span>", html.EscapeString(lastPart)))
					}
				case ModeConsole:
					if endsWithCapital {
						sb.WriteString(color.New(color.FgMagenta).Sprintf("%s", lastPart))
					} else {
						sb.WriteString(color.New(color.FgBlue).Sprintf("%s", lastPart))
					}
				}

				continue
			}

			switch mode {
			case ModeHTML:
				sb.WriteString(fmt.Sprintf("<span style=\"color:#ffd230\">%s</span>", html.EscapeString(token.Value)))
			case ModeConsole:
				sb.WriteString(color.New(color.FgHiYellow).Sprintf("%s", token.Value))
			}
			continue
		}

		if tagName && token.Type == lexer.TokenDot {
			continue
		}

		if token.Type == lexer.TokenString {
			sb.WriteString(highlightString(mode, token))
			continue
		}

		if token.Type == lexer.TokenHtmlText {
			if mode == ModeHTML {
				sb.WriteString(fmt.Sprintf("<span style=\"color:#e5e7eb\">%s</span>", html.EscapeString(token.Value)))
			} else {
				sb.WriteString(color.New(color.FgWhite).Sprintf("%s", token.Value))
			}
			continue
		}

		if mode == ModeHTML {
			sb.WriteString(fmt.Sprintf("<span style=\"color:#d1d5dc\">%s</span>", html.EscapeString(token.Value)))
		} else {
			sb.WriteString(color.New(color.FgHiBlack).Sprintf("%s", token.Value))
		}
	}

	return sb.String(), nil
}
