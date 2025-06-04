package lexer

type TokenType string

const (
	TokenFn     TokenType = "fn"
	TokenStruct TokenType = "struct"
	TokenImpl   TokenType = "impl"

	TokenLet   TokenType = "let"
	TokenConst TokenType = "const"

	TokenEvent  TokenType = "event"
	TokenServer TokenType = "server"
	TokenClient TokenType = "client"

	TokenPub     TokenType = "pub"
	TokenSub     TokenType = "sub"
	TokenRequest TokenType = "request"

	TokenImport TokenType = "import"
	TokenFrom   TokenType = "from"

	TokenStatic   TokenType = "static"
	TokenIf       TokenType = "if"
	TokenElse     TokenType = "else"
	TokenFor      TokenType = "for"
	TokenWhile    TokenType = "while"
	TokenBreak    TokenType = "break"
	TokenContinue TokenType = "continue"
	TokenReturn   TokenType = "return"

	TokenIdentifier TokenType = "identifier"
	TokenString     TokenType = "string"
	TokenNumber     TokenType = "number"
	TokenBool       TokenType = "bool"
	TokenNil        TokenType = "nil"

	// Symbols
	TokenAssign         TokenType = "="
	TokenComma          TokenType = ","
	TokenColon          TokenType = ":"
	TokenSemicolon      TokenType = ";"
	TokenDot            TokenType = "."
	TokenOpenParen      TokenType = "("
	TokenCloseParen     TokenType = ")"
	TokenOpenBrace      TokenType = "{"
	TokenUnescapedBrace TokenType = "#{"
	TokenCloseBrace     TokenType = "}"
	TokenOpenBracket    TokenType = "["
	TokenCloseBracket   TokenType = "]"

	// Operators
	TokenPlus            TokenType = "+"
	TokenIncrement       TokenType = "++"
	TokenMinus           TokenType = "-"
	TokenDecrement       TokenType = "--"
	TokenAsterisk        TokenType = "*"
	TokenPower           TokenType = "**"
	TokenSlash           TokenType = "/"
	TokenBackSlash       TokenType = "\\"
	TokenPercent         TokenType = "%"
	TokenSelfClosingTag  TokenType = "/>"
	TokenClosingStartTag TokenType = "</"
	TokenArrow           TokenType = "=>"
	TokenFnReturnArrow   TokenType = "->"

	// Comparison
	TokenEqual        TokenType = "=="
	TokenNotEqual     TokenType = "!="
	TokenLessThan     TokenType = "<"
	TokenLessEqual    TokenType = "<="
	TokenGreaterThan  TokenType = ">"
	TokenGreaterEqual TokenType = ">="
	TokenQuestion     TokenType = "?"
	TokenIn           TokenType = "in"

	// Logical
	TokenAnd TokenType = "&&"
	TokenOr  TokenType = "||"
	TokenNot TokenType = "!"

	//
	TokenSingleLineComment = "<comment>"
	TokenMultiLineComment  = "<multi-comment>"
	TokenNewLine           = "<newline>"
	TokenWhiteSpace        = "<whitespace>"

	TokenUnknown = "<unknown>"
)

func Keywords() []TokenType {
	return []TokenType{
		TokenFn,
		TokenStatic,
		TokenImpl,
		TokenStruct,

		TokenLet,
		TokenConst,

		TokenEvent,
		TokenServer,
		TokenClient,

		TokenPub,
		TokenSub,
		TokenRequest,

		TokenImport,
		TokenFrom,

		TokenIf,
		TokenElse,
		TokenFor,
		TokenWhile,
		TokenBreak,
		TokenContinue,
		TokenReturn,

		TokenNil,

		TokenIn,
	}
}

func Operators() []TokenType {
	return []TokenType{
		TokenAssign,
		TokenComma,
		TokenColon,
		TokenSemicolon,
		TokenDot,
		TokenOpenParen,
		TokenCloseParen,
		TokenOpenBrace,
		TokenCloseBrace,
		TokenOpenBracket,
		TokenCloseBracket,

		// Operators
		TokenPlus,
		TokenMinus,
		TokenAsterisk,
		TokenSlash,
		TokenBackSlash,
		TokenPercent,

		// Comparison
		TokenEqual,
		TokenNotEqual,
		TokenLessThan,
		TokenLessEqual,
		TokenGreaterThan,
		TokenGreaterEqual,
		TokenSelfClosingTag,
		TokenQuestion,

		// Logical
		TokenAnd,
		TokenOr,
		TokenNot,
	}
}
