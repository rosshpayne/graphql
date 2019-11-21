package lexer

import (
	"fmt"
	"testing"

	"github.com/graphql/token"
)

func TestNextToken(t *testing.T) {
	input := "\ufeff" + `

#  comment 

{
    _1use_r(id: -0) {
    id
    name
	}
}
	type Character {
		界: String!
	    appearsIn: [Episode!]!
   }
   
   query qName {
   	      ...fri世界endFields
	      user(id: -4.567E-2, mode: null) {
	       aliasid: id
	       profilePic(width: -100)
		}
   }
  
  mutation {
  sendEmail(message: """
    Hello,
      World!
    Yours,
      GraphQL.
""")}

mutation {
  sendEmail(message: "Hello,\n  World!\n\nYours,\n  GraphQL.")
}

enum Episode {
  NEWHOPE
  EMPIRE
  JEDI
}

[1, 2, -13]

`
	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.BOM, "\ufeff"},
		{token.COMMENT, "#"},
		{token.LBRACE, "{"},
		{token.IDENT, "_1use_r"},
		{token.LPAREN, "("},
		{token.IDENT, "id"},
		{token.COLON, ":"},
		{token.INT, "-0"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "id"},
		{token.IDENT, "name"}, // 10
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
		{token.TYPE, "type"},
		{token.IDENT, "Character"},
		{token.LBRACE, "{"}, //15
		{token.IDENT, "界"},
		{token.COLON, ":"},
		{token.IDENT, "String"},
		{token.BANG, "!"},
		{token.IDENT, "appearsIn"}, //20
		{token.COLON, ":"},
		{token.LBRACKET, "["},
		{token.IDENT, "Episode"},
		{token.BANG, "!"},
		{token.RBRACKET, "]"}, //25
		{token.BANG, "!"},
		{token.RBRACE, "}"},
		{token.QUERY, "query"},
		{token.IDENT, "qName"},
		{token.LBRACE, "{"}, //30
		{token.EXPAND, "..."},
		{token.IDENT, "fri世界endFields"},
		{token.IDENT, "user"},
		{token.LPAREN, "("},
		{token.IDENT, "id"}, //35
		{token.COLON, ":"},
		{token.FLOAT, "-4.567E-2"},
		{token.COMMA, ","},
		{token.IDENT, "mode"},
		{token.COLON, ":"},
		{token.NULL, "null"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "aliasid"}, // 40
		{token.COLON, ":"},
		{token.IDENT, "id"},
		{token.IDENT, "profilePic"},
		{token.LPAREN, "("},
		{token.IDENT, "width"}, // 45
		{token.COLON, ":"},
		{token.INT, "-100"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
		{token.MUTATION, "mutation"},
		{token.LBRACE, "{"},
		{token.IDENT, "sendEmail"},
		{token.LPAREN, "("},
		{token.IDENT, "message"},
		{token.COLON, ":"},
		{token.STRING, "\n    Hello,\n      World!\n    Yours,\n      GraphQL.\n"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.MUTATION, "mutation"},
		{token.LBRACE, "{"},
		{token.IDENT, "sendEmail"},
		{token.LPAREN, "("},
		{token.IDENT, "message"},
		{token.COLON, ":"},
		{token.STRING, `Hello,\n  World!\n\nYours,\n  GraphQL.`},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.ENUM, "enum"},
		{token.IDENT, "Episode"},
		{token.LBRACE, "{"},
		{token.IDENT, "NEWHOPE"},
		{token.IDENT, "EMPIRE"},
		{token.IDENT, "JEDI"},
		{token.RBRACE, "}"},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.COMMA, ","},
		{token.INT, "-13"},
		{token.RBRACKET, "]"},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		//	fmt.Printf("%v\n", tok)
		fmt.Println(tok.Literal)
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q Error: %s",
				i, tt.expectedType, tok.Type, l.Error())
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q Error: %s",
				i, tt.expectedLiteral, tok.Literal, l.Error())
		}
	}
}

func TestNextToken2(t *testing.T) {
	input := "\ufeff" + `
query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    id
    name
    profilePic(size: $devicePicSize)
}
}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.BOM, "\ufeff"},
		{token.QUERY, "query"},
		{token.IDENT, "getZuckProfile"},
		{token.LPAREN, "("},
		{token.DOLLAR, "$"},
		{token.IDENT, "devicePicSize"},
		{token.COLON, ":"},
		{token.IDENT, "Int"},
		{token.ASSIGN, "="},
		{token.INT, "1234"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "xyzalias"},
		{token.COLON, ":"},
		{token.IDENT, "user"},
		{token.LPAREN, "("},
		{token.IDENT, "id"},
		{token.COLON, ":"},
		{token.INT, "4"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "id"},
		{token.IDENT, "name"},
		{token.IDENT, "profilePic"},
		{token.LPAREN, "("},
		{token.IDENT, "size"},
		{token.COLON, ":"},
		{token.DOLLAR, "$"},
		{token.IDENT, "devicePicSize"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		//fmt.Printf("%v\n", tok)

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q Error: %s",
				i, tt.expectedType, tok.Type, l.Error())
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q Error: %s",
				i, tt.expectedLiteral, tok.Literal, l.Error())
		}
	}
}
