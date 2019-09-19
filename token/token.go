package token

type TokenType string
type TokenCat string

// type Token_ struct {
// 	Type TokenType
// 	Cat  TokenCat
// }

const (
	IDENT TokenType = "IDENT"
)
const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals

	INT       = "Int"    // 1343456
	FLOAT     = "Float"  // 3.42
	STRING    = "String" // contents between " or """
	RAWSTRING = "Raw"
	LIST      = "LIST"
	BOOL      = "Boolean"

	// Category
	VALUE    = "VALUE"
	NONVALUE = "NONVALUE"

	// Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"

	// Punctuator :: one of ! $ ( ) ... : = @ [ ] { | }
	COMMA      = ","
	SEMICOLON  = ";"
	COLON      = ":"
	COMMENT    = "#"
	UNDERSSCRE = "_"
	DOLLAR     = "$"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	EXPAND = "..."
	// delimiters
	RAWSTRINGDEL = `"""`

	STRINGDEL = `"`

	BOM = "BOM"

	// Keywords
	QUERY        = "QUERY"
	MUTATION     = "MUTATION"
	SUBSCRIPTION = "SUBCRIPTION"
	TYPE         = "TYPE"
	FRAGMENT     = "FRAGMENT"
	ON           = "ON"
	NULL         = "NULL"
	TRUE         = "true"
	FALSE        = "false"
	ENUM         = "ENUM"
)

type Pos struct {
	Line int
	Col  int
}

// Token is exposed via token package so lexer can create new instanes of this type as required.
type Token struct {
	Cat      TokenCat
	Type     TokenType
	Literal  string // string value of token - rune, string, int, float, bool
	Position Pos    // start position of token
	Illegal  bool
}

var keywords = map[string]struct {
	Type TokenType
	Cat  TokenCat
}{
	"query":        {QUERY, NONVALUE},
	"mutation":     {MUTATION, NONVALUE},
	"subscription": {SUBSCRIPTION, NONVALUE},
	"enum":         {ENUM, NONVALUE},
	"fragment":     {FRAGMENT, NONVALUE},
	"on":           {ON, NONVALUE},
	"type":         {TYPE, NONVALUE},
	"null":         {NULL, VALUE},
	"true":         {TRUE, VALUE},
	"false":        {FALSE, VALUE},
}

func LookupIdent(ident string) (TokenType, TokenCat) {
	if tok, ok := keywords[ident]; ok {
		return tok.Type, tok.Cat
	}
	return IDENT, NONVALUE
}
