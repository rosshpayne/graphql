package parser

import (
	"strings"
	"testing"

	"github.com/graphql/lexer"
)

func compare(doc, expected string) bool {

	return trimWS(doc) != trimWS(expected)

}

func trimWS(input string) string {

	var out strings.Builder
	for _, v := range input {
		if !(v == '\u0009' || v == '\u0020' || v == '\u000A' || v == '\u000D' || v == ',') {
			out.WriteRune(v)
		}
	}
	return out.String()

}

func TestParseQuery(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  Xid : [Int!]
				  Zname : String
    			  profilePic(size: Int) : T1
    			  }
    			  
    	type T1 { 
    			aa : String
    			bb : [[Int]!]!
    			cc : T2
    			ff : Int
    			gghi : Float
    			xyz : String
    			dog : String
    	}
    	
    	type T2 {
    	    	ddd: String
    			eee (f : Int) : Int
    	}
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb
    	cc {
    		add: ddd
    		aee: eee ( f : $devicePicSize )
    	}
    	ff
    	gghi
    	xyz
    	dog
    }
}
}
`
	var expectedDoc string = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: 1234) {
    	aa
    	bb
    	cc {
    		add: ddd
    		aee: eee ( f : 1234 )
    	}
    	ff
    	gghi
    	xyz
    	dog
    }
}
}
	`
	var parseErrs []string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}
	//

}

func TestParseMissingRPAREN(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb
    	cc {
    		add: ddd
    		aee: eee ( f : "abcdef-hij" }
    	ff
    	gghi
    	xyz
    	dog
    }
}
}
`

	parseErrs := []string{
		`Expected an argument name or a right parenthesis got "} ff" at line: 10, column: 35`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
}

func TestParseMisplacedVariable(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb($devicePicSize: Int = 1234)
    	cc {
    		add: ddd
    		aee: eee ( f : """abc
    		def-hij
  """)
    	}
    	ff
    	gghi
    	cat
    	dog
    }

}
`
	parseErrs := []string{
		`Expected an argument name or a right parenthesis got "$ devicePicSize" at line: 7, column: 9`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
}

func TestParseArgumentNull(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query getZuckProfile($devicePicSize: [Int!] = [1234 23 234 32 null] ) {
  xyzalias: user(id: 4) {
    sex
    author {
    	name
    	age
    }
  }
}
`

	parseErrs := []string{
		`List cannot contain NULLs at line: 1 column: 63`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
}

func TestParseNoName1(t *testing.T) {
	var input = `query  {
  xyzalias: user(id: 4) {
    author {
    	name
    	age
    }
  }
}
`
	var parseErrs []string

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
	}
}

func TestParseNoName2(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query ($devicePicSize: [Int!] = [1234 23 234 32] ) {
  xyzalias: user(id: 4) {
    author {
    	name
    	age
    }
  }
}
`
	var parseErrs []string

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
	}
}

func TestParseNullLiteral(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query ($devicePicSize: Int! = null ) {
  xyzalias: user(id: 4) {
    author {
		name   
		age
    }
  }
}
`
	var parseErrs []string = []string{
		`Value cannot be NULL at line: 1 column: 31`,
	}

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
	}
}

func TestParseMissingDollar(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query getZuckProfile(devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    author {
		name   
		age
    }
}
}
`
	parseErrs := []string{
		`Missing "$", at line: 1, column: 22`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)

}

func TestParseMissingDollar2(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	var input = `query getZuckProfile(@devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    author {
		name   
		age
    }
}
}
`
	parseErrs := []string{
		`Expected "$" got "@", at line: 1, column: 22`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)

}

func TestParseMissingDollar3(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query getZuckProfile(#asf adsf asdf
	$devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    author {
		name   
		age
    }
}
}
`
	var parseErrs []string

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestParseMissingDollar4(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}
	var input = `query getZuckProfile(#$devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    author {
		name   
		age
    }
}
}
`
	var parseErrs []string = []string{
		`Missing "$", at line: 2, column: 3`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)

}

func TestParseIllegalInName(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: us*er(id: 4) {
    Xid
    Zname
}
}
`
	parseErrs := []string{
		`Expected an identifier for a fragment or inlinefragment got ILLEGAL. at line: 2, column: 15`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)

}
func TestParseMissingArgName(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb( : 65.4)
    	cc {
    		add: ddd
    		aee: eee ( f : 22)
    	}
    	ff
    	gghi
    	cat
    	dog
    }

}
`
	parseErrs := []string{
		`Expected an argument name or a right parenthesis got ": 65.4" at line: 7, column: 10`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
}

func TestParseMissingColon(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb( acd  65.4)
    	cc {
    		add: ddd
    		aee: eee ( f : 22)
    	}
    	ff
    	gghi
    	cat
    	dog
    }

}
`

	parseErrs := []string{
		`Expected an argument name or a right parenthesis got "acd 65.4" at line: 7, column: 10`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
}

func TestParseLeadingDoubleUnderscore(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: __use_r(id: 4) {
    Xid
    Zname
  }
}
`
	parseErrs := []string{
		`identifer "__use_r" cannot start with two underscores at line: 2, column: 13`,
		`Field "__use_r" is not a member of "Query" at line: 2 column: 13`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)
}

func TestParseLeadingSingleUnderscore(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: 4) {
    Xid
    Zname
  }
}
`

	var parseErrs []string = []string{
		`Field "_use_r" is not a member of "Query" at line: 2 column: 13`,
	}

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	checkErrors(errs, parseErrs, t)

}

func TestParseStmtVariableNoDefault(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int) {
  xyzalias: user(id: 33) {
    sex
    author
  }
}
`

	var parseErrs []string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), input) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}

}

func TestParseBoolArgValue(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: false) {
    sex
    author
  }
}
`
	var parseErrs []string = []string{
		`Required type for argument "id" is Int, got Boolean at line: 2 column: 18`,
	}

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), input) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}

}

func TestParseMultiArgValue(t *testing.T) {
	//
	// setup
	//
	{
		input := `	type Query {
			user (id : Int name: String) : User
		}`
		setup(input, t)
	}

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 23 name: """Ross""" ) {
    sex
    author
  }
}
`

	var parseErrs []string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), input) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}

	//
	// teardown
	//
	{
		input := `	type Query {
			user (id : Int) : User
		}`
		teardown(input, t)
	}

}
func TestParseBooleanVarType(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Boolean = false) {
  xyzalias: user(id: $devicePicSize) {
    sex
    author
  }
}`

	var parseErrs []string = []string{
		`Required type for argument "id" is Int, got Boolean at line: 2 column: 18`,
	}

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), input) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}

}

func TestParseVariableReference(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: $devicePicSize) {
    sex
    author
  }
}`

	var parseErrs []string
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	_, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
}

func TestParseWrongVariableNameInArgument(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  sex : [Int!]
				  author : Person
    			  }
    			  `
		setup(inputSDL, t)
	}

	//
	// Test
	//
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
	  xyzalias: user(id: $ePicSize) {
	    sex
	    author
	  }
	}
`
	var parseErrs []string = []string{
		`Variable, ePicSize not defined  at line: 2, column: 24`,
	}
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	_, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
}

func TestParseNullValue(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: null) {
    sex
    author
  }
}
`
	var parseErrs []string
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	_, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
}

func TestParseList0(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: [1 2 34 56.78]) {
    sex
    author
  }
}
`

	var parseErrs []string = []string{`Expected a Int for argument "id", got a List, at line: 2 column: 35`}
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	_, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)

}

func TestParseList1(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: user(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"]) {
    sex
    author
  }
}
`
	var parseErrs []string = []string{`Expected a Int for argument "id", got a List, at line: 2 column: 72`}
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	_, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)

}

func TestParseUseOfCommas(t *testing.T) {
	//
	// setup
	//
	{
		input := `	type Query {
			user (id : [Int]) : User
		}`
		setup(input, t)
	}
	//
	// test
	//
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: [1 2,, 34, ,56 , ]) {
    sex
    author
  }
}
`
	expectedDoc := `	query getZuckProfile ( $devicePicSize : Int = 1234) { 
                        xyzalias : user (id:[1 2 34 56] ) {
                                sex
                                author
                     } }`

	var parseErrs []string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
		t.Log("doc.String(): " + doc.String())
	}
	//
	// teardown
	//
	{
		input := `	type Query {
			user (id : Int!) : User
		}`
		teardown(input, t)
	}
}

func TestParserObject1(t *testing.T) {
	//
	// setup
	//
	{
		input := `	type Query {
			user (id : Objx) : User
		}
		input Objx {
			id  : Float
			cat : Int
			food : [Int!]
		}
			`

		setup(input, t)
	}

	var input = `query getZuckProfile($devicePicSize: Objx = { id : 23.4 cat : 32 food : [1,2,3,4 55 ]}) {
   xyzalias: user(id: $devicePicSize) {
     sex
     author
  }
}
`

	var expectedDoc string = `query getZuckProfile ($devicePicSize:Objx={id:23.4cat:32food:[123455]})
	   { xyzalias:user(id:{id:23.4 cat:32 food: [ 1 2 3 4 55]}) {
	   sex
	   author
	   }
	   }`
	var parseErrs []string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)

	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}
	//
	// teardown
	//
	{
		inputSDL := `
		schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				
		type Query {
			user (id : Int) : User
		}
		
		type User {
				  Xid : [Int!]
				  Zname : String
    			  profilePic(size: Int) : T1
    			  }
    			  
    	type T1 { 
    			aa : String
    			bb : [[Int]!]!
    			cc : T2
    			ff : Int
    			gghi : Float
    			xyz : String
    			dog : String
    	}
    	
    	type T2 {
    	    	ddd: String
    			eee (f : Int) : Int
    	}
`

		teardown(inputSDL, t)
	}
}
