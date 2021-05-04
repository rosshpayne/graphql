package parser

import (
	"testing"

	"github.com/rosshpayne/graphql/lexer"
)

func TestValidateArgs(t *testing.T) {

	//
	// Setup
	//
	{
		inputSDL := `
		type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}
		
		type Person {name : String! age(ScaleBy : Float = 1.2) : [[Int!]]! other : [String!] posts(resp :  Int! ) : [Post!]}
		
		type Post {title : String! title2 : String! author : [Person!]!}
		
		directive @include  (  if : Boolean  = false  ) on | INLINE_FRAGMENT| FIELD
		`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ ($expandedInfo: Boolean = true) {
	     allPersons(last: 2) {
	         name 
	         age (ScaleBy: 1) 
	         ... @include (if: $expandedInfo) {						# inline fragment
	         	posts (resp: 3.5) {
	         		 author {
	         	 		name
	         	 		age 										# (ScaleBy: 1.2) created by parser
	         	     }
	         	     address
	        	 }
	           }
	     }
	}
`

	var expectedErr []string = []string{
		`Required type for argument "ScaleBy" is Float, got Int at line: 5 column: 16`,
		`Argument "first" must be defined (type "[[String!]]") at line: 3, column: 7`,
		`Field "address" is not a member of "posts" (SDL Object "Post") at line: 12 column: 17`,
		`Required type for argument "resp" is Int, got Float at line: 7 column: 19`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//	p.ClearCache()
	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)

}

func TestValidateDirectives(t *testing.T) {

	//
	// Setup
	//
	{
		inputSDL := `
		type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}
		
		type Person {name : String! age(ScaleBy : Float = 2.3) : [[Int!]]! other : [String!] posts(resp :  Int! ) : [Post!]}
		
		type Post {title : String! title2 : String! author : [Person!]!}
		
		directive @include (arg1: Int = 34 arg2: Float ) on FIELD | INLINE_FRAGMENT |QUERY
		`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ ($expandedInfo: Boolean = true)  @ include(arg1: 2 arg2: 3.2) {
	     allPersons(last: 2 first : [["abc", "def" ]["def"]]) {
	         name @  include(arg1:3 arg2: 4.5 )
	         age (ScaleBy: 1.5) 
	         ... @  include(arg2: 4.5 ) {					
	         	posts (resp: 3) {
	         		 author {
	         	 		name
	         	 		age 										# (ScaleBy: 1.2) created by parser
	         	     }
	        	 }
	           }
	     }
	}
`

	var expectedErr []string
	var expectedDoc string = `         
	   query XYZ ( $expandedInfo : Boolean = true) @include (arg1:2 arg2:3.2){ 
                        allPersons(last:2 first:[["abc" "def"]  ["def"] ] ) {
                                name@include(arg1:3 arg2:4.5) 
                                age(ScaleBy:1.5)
                                ... on Person@include(arg2:4.5 arg1:34)  {
                                        posts(resp:3) {
                                                author {
                                                        name
                                                        age(ScaleBy:2.3)
                                                        }
                                                }
                                        }
                                }
                }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//	p.ClearCache()
	d, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(d.String(), expectedDoc) {
			t.Errorf("Got:      [%s] \n", trimWS(d.String()))
			t.Errorf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
	}

}

func TestValidateDirectiveWrongLoc(t *testing.T) {

	//
	// Setup
	//
	{
		inputSDL := `
		type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}

		type Person {name : String! age(ScaleBy : Float = 1.2) : [[Int!]]! other : [String!] posts(resp :  Int! ) : [Post!]}

		type Post {title : String! title2 : String! author : [Person!]!}

		directive @include (arg1: Int = 34 arg2: Float ) on FIELD
		`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ ($expandedInfo: Boolean = true) {
	     allPersons(last: 2 first : [["abc", "def" ]["def"]]) {
	         name @  include(arg1:3 arg2: 4.5 )
	         age (ScaleBy: 1.5) 
	         ... @  include(arg2: 4.5 ) {					
	         	posts (resp: 3) {
	         		 author {
	         	 		name
	         	 		age 										# (ScaleBy: 1.2) created by parser
	         	     }
	        	 }
	           }
	     }
	}
`

	var expectedErr []string = []string{
		`Directive "@include" is not defined for INLINE_FRAGMENT (see schema doc, DefaultDoc), at line: 6 column: 18`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//	p.ClearCache()
	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)

}
