package token

import (
	"fmt"
)

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
	RAWSTRING = "RAWSTRING"
	LIST      = "List"
	BOOLEAN   = "Boolean"
	OBJECT    = "Object"

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
	ATSIGN     = "@"

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
	Cat          TokenCat
	Type         TokenType
	IsScalarType bool
	Literal      string // string value of token - rune, string, int, float, bool
	Loc          Pos    // start location (line,col) of token
	Illegal      bool
}

func (t *Token) AtPosition() string {
	return fmt.Sprintf("at line: %d, column: %d\n", t.Loc.Line, t.Loc.Col)
}

var keywords = map[string]struct {
	Type         TokenType
	Cat          TokenCat
	IsScalarType bool
}{
	"Int":          {INT, NONVALUE, true},
	"Float":        {FLOAT, NONVALUE, true},
	"String":       {STRING, NONVALUE, true},
	"Boolean":      {BOOLEAN, NONVALUE, true},
	"query":        {QUERY, NONVALUE, false},
	"mutation":     {MUTATION, NONVALUE, false},
	"subscription": {SUBSCRIPTION, NONVALUE, false},
	"fragment":     {FRAGMENT, NONVALUE, false},
	"enum":         {ENUM, NONVALUE, false},
	"on":           {ON, NONVALUE, false},
	"type":         {TYPE, NONVALUE, false},
	"null":         {NULL, VALUE, false},
	"true":         {TRUE, VALUE, false},
	"false":        {FALSE, VALUE, false},
}

func LookupIdent(ident string) (TokenType, TokenCat, bool) {
	if tok, ok := keywords[ident]; ok {
		return tok.Type, tok.Cat, tok.IsScalarType
	}
	return IDENT, NONVALUE, false
}
