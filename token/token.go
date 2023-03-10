package token

type Type string

type Token struct {
	Type           // 类型
	Literal string // 实际含义
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"
	// IDENT 标识符
	IDENT  = "IDENT" // 变量名，函数名
	INT    = "INT"
	STRING = "STRING"
	// ASSIGN 操作符
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	SLASH    = "/"
	BANG     = "!"
	LT       = "<"
	GT       = ">"
	EQ       = "=="
	NEQ      = "!="
	// COMMA 分隔符
	COMMA     = ","
	SEMICOLON = ";"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
	COLON     = ":"
	// FUNCTION 关键词
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
)

var Keywords = map[string]Type{
	"fn":     FUNCTION,
	"let":    LET,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
}

func LookupIdent(ident string) Type {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	// 不是关键字，那就是普通标识符
	return IDENT
}
