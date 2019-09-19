package lexer

import (
	"unicode"
	"unicode/utf8"

	"github.com/graphql/token"
)

// pos preserves position of read position
type readPos struct {
	Line int
	Col  int
}

// Lexer parses an Input string (embedded in token pkg) and returns it as tokens - defined in token package.
type Lexer struct {
	Eloc  readPos // position of illegal char
	input string
	cpos  int    // Current rune position in input
	rpos  int    // Read rune position in input
	ch    rune   // current rune under examination, added to token during lex processings
	del   string // string delimeter
	line  int
	col   int // curren col position
	err   error
}

func (l *Lexer) Cpos() int {
	return l.cpos
}

func (l *Lexer) Input() string {
	return l.input
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readRune() // prime lexer struct
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	//	fmt.Printf("NextToken: %c\n", l.ch)
	l.skipWhitespace() // scan to next non-whitespace and return its value as a token

	switch l.ch {
	case '\ufeff':
		tok = l.newToken(token.BOM, l.ch)
	case '#':
		tok = l.newToken(token.COMMENT, l.ch)
		l.readToEol()
	case '.': // ... expand sequence
		if l.peekRune() == '.' {
			//ch := l.ch
			l.readRune()
			if l.peekRune() == '.' {
				//ch := l.ch
				l.readRune()
				literal := token.EXPAND
				tok = token.Token{Type: token.EXPAND, Literal: literal}
			} else {
				tok = l.newToken(token.ILLEGAL, l.ch)
			}
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}
	case '"':
		if l.peekRune() == '"' {
			//ch := l.ch
			l.readRune()
			if l.peekRune() == '"' {
				l.readRune()
				l.del = token.RAWSTRINGDEL
				tok = l.readString()
			} else {
				tok = l.newToken(token.ILLEGAL, l.ch)
			}
		} else {
			l.del = token.STRINGDEL
			tok = l.readString()
		}
	case ':':
		tok = l.newToken(token.COLON, l.ch)
	case ',':
		tok = l.newToken(token.COMMA, l.ch)
	case '{':
		tok = l.newToken(token.LBRACE, l.ch)
	case '}':
		tok = l.newToken(token.RBRACE, l.ch)
	case '(':
		tok = l.newToken(token.LPAREN, l.ch)
	case ')':
		tok = l.newToken(token.RPAREN, l.ch)
	case '[':
		tok = l.newToken(token.LBRACKET, l.ch)
		tok.Cat = token.VALUE
	case ']':
		tok = l.newToken(token.RBRACKET, l.ch)
	case '$':
		tok = l.newToken(token.DOLLAR, l.ch) // cat VALUE
		tok.Cat = token.VALUE                // maybe a VAL when not in Variable def otherwise is an IDENT. Default to a VALUE
	case '=':
		tok = l.newToken(token.ASSIGN, l.ch)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if unicode.IsLetter(l.ch) || l.ch == '_' {
			tok = l.readIdentifier()
			tok.Type, tok.Cat = token.LookupIdent(tok.Literal) // IDENT,nil or <keyword>,<VALUE | NONVALUE>
		} else if unicode.IsDigit(l.ch) || l.ch == '-' || l.ch == '+' {
			tok = l.readNumber()
		} else {
			tok = l.newToken(token.ILLEGAL, l.ch)
		}
		return tok
	}
	if tok.Type != "ILLEGAL" {
		l.readRune() // prime l.ch
	}
	return tok
}

func (l *Lexer) skipWhitespace() {
	// Horizontal Tab (U+0009) Space (U+0020)
	// LineTerminator :: New Line (U+000A)
	//  Carriage Return (U+000D) [lookahead ≠ New Line (U+000A)] Carriage Return (U+000D) New Line (U+000A)
	for l.ch == '\u0009' || l.ch == '\u0020' || l.ch == '\u000A' || l.ch == '\u000D' || l.ch == ',' {
		if l.ch == '\n' { // linefeed
			l.line++
			l.col = 0
		}
		l.readRune()
	}
}

func (l *Lexer) readRune() {
	// get next byte in string
	if l.rpos >= len(l.input) {
		l.ch = 0
	} else {
		var size int
		// TODO: check token type. Only comment and string need rune reads all others simple ascii will suffice
		l.ch, size = utf8.DecodeRuneInString(l.input[l.rpos:])
		l.cpos = l.rpos
		l.rpos += size
		if !(l.ch == '\n' || l.ch == '\r') {
			l.col++
		}
	}
	//	fmt.Printf("readRune: %c\n", l.ch)

}

func (l *Lexer) peekRune() rune {
	if l.rpos >= len(l.input) {
		return 0
	} else {
		rn, _ := utf8.DecodeRuneInString(l.input[l.rpos:])
		return rn
	}
}

func (l *Lexer) readIdentifier() token.Token {
	start := token.Pos{l.line, l.col}
	pos := l.cpos
	for unicode.IsLetter(l.ch) || l.ch == '_' || unicode.IsDigit(l.ch) {
		l.readRune()
	}
	return token.Token{Cat: token.NONVALUE, Type: token.STRING, Literal: l.input[pos:l.cpos], Position: start}
}

func (l *Lexer) readNumber() token.Token {
	var tokenT token.TokenType = token.INT
	var illegalT bool
	spos := l.cpos
	start := token.Pos{l.line, l.col}
	if l.ch == '-' {
		//l.skipWhitespace()
		l.readRune()
	}
	for unicode.IsDigit(l.ch) {
		l.readRune()
	}
	tokenT = token.INT
	switch l.ch {
	case '.':
		tokenT = token.FLOAT
		l.readRune()
		for unicode.IsDigit(l.ch) {
			l.readRune()
		}
		if l.ch == 'e' || l.ch == 'E' {
			l.readRune()
			if l.ch == '-' || l.ch == '+' {
				l.readRune()
			}
			for unicode.IsDigit(l.ch) {
				l.readRune()
			}
		}

	case 'e', 'E':
		tokenT = token.FLOAT
		l.readRune()
		if l.ch == '-' || l.ch == '+' {
			l.readRune()
		}
		for unicode.IsDigit(l.ch) {
			l.readRune()
		}

	default: // all letters other than e E
		if unicode.IsLetter(l.ch) {
			l.Eloc = readPos{l.line, l.col}
			// token is now interpreted as an illegal IDENT
			illegalT = true
			tokenT = token.IDENT
			for !(unicode.IsSpace(l.ch)) {
				l.readRune()
			}
		}
	}
	return token.Token{Cat: token.VALUE, Type: tokenT, Literal: l.input[spos:l.cpos], Illegal: illegalT, Position: start}

}

func (l *Lexer) readString() token.Token {

	pos := l.cpos + 1
	start := token.Pos{l.line, l.col}
	//fmt.Println("pos: ", pos)
	for {
		l.readRune()
		if l.ch == '"' { // "
			if l.del == token.STRINGDEL {
				break
			} else {
				l.readRune()
				if l.ch == '"' { // "
					if l.del == token.STRINGDEL {
						return l.newToken(token.ILLEGAL, l.ch)
					}
					l.readRune()
					if l.ch == '"' { // "
						break
					}
				}
				return l.newToken(token.ILLEGAL, l.ch)
			}
		}
		if l.del == token.RAWSTRINGDEL && (l.ch == 10) { // linefeed
			l.line++
			l.col = 0
		}
	}
	//fmt.Println("l.cpos: ", l.cpos)
	var epos int
	if l.del == token.RAWSTRINGDEL {
		epos = 2
		return token.Token{Cat: token.VALUE, Type: token.RAWSTRING, Literal: l.input[pos : l.cpos-epos], Position: start}
	}
	return token.Token{Cat: token.VALUE, Type: token.STRING, Literal: l.input[pos : l.cpos-epos], Position: start}
}

func (l *Lexer) readToEol() {
	for {
		l.readRune()
		if l.ch == '\u000D' || l.ch == '\u000A' {
			//l.skipWhitespace()
			break
		}
	}
}

func (l *Lexer) newToken(tokenType token.TokenType, ch rune, pos ...token.Pos) token.Token {
	if len(pos) > 0 {
		return token.Token{Cat: token.NONVALUE, Type: tokenType, Literal: string(ch), Position: pos[0]}
	}
	return token.Token{Cat: token.NONVALUE, Type: tokenType, Literal: string(ch), Position: token.Pos{l.line, l.col}}
}
