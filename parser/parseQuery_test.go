package parser

import (
	"testing"

	//	db "github.com/graph-sdl/db"
	lsdl "github.com/graph-sdl/lexer"
	psdl "github.com/graph-sdl/parser"
	"github.com/graphql/client"
	"github.com/graphql/lexer"
)

func setup(inputSDL string, t *testing.T) {
	l := lsdl.New(inputSDL)
	p := psdl.New(l)
	d, errs := p.ParseDocument()
	for _, v := range errs {
		t.Fatalf("Setup failed for %s: %s", t.Name(), v)
	}
	t.Log(d.String())
}

func teardown(inputSDL string, t *testing.T) {
	t.Logf(" ****  Teardown started for %s   ****", t.Name())
	l := lsdl.New(inputSDL)
	p := psdl.New(l)
	_, errs := p.ParseDocument()
	for _, v := range errs {
		t.Errorf("Setup not expected Error =[%q]", v)
	}
	if errs != nil {
		t.Fatalf(`Setup failed for %s`, t.Name())
	}

	t.Logf("Teardown completed for %s", t.Name())
}

func checkErrors(errs []error, expectedErr []string, t *testing.T) {

	for _, ex := range expectedErr {
		if len(ex) == 0 {
			break
		}
		found := false
		for _, err := range errs {
			if trimWS(err.Error()) == trimWS(ex) {
				found = true
			}
		}
		if !found {
			t.Errorf(`Expected Error = [%q]`, ex)
		}
	}
	for _, got := range errs {
		found := false
		for _, exp := range expectedErr {
			if trimWS(got.Error()) == trimWS(exp) {
				found = true
			}
		}
		if !found {
			t.Errorf(`Unexpected Error = [%q]`, got.Error())
		}
	}
}

func TestQueryArgumentValue(t *testing.T) {
	//
	// setup
	//
	{

		inputSDL := `
				schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
		type Person {
					name : String! 
					age  (  ScaleBy : Float  =3.2 ) : [[Int!]]! 
					other : [String!] 
					posts  (  resp : [Int!]  = [12 213 22] ) : [Post!] 
		}
		type Query {allPersons  (  last : Int     first : [[String!]]   ) : [Person!] }
		`
		setup(inputSDL, t)
	}

	var input = `query XYZ {
		     allPersons(last: [ 1 23 43] first: [["abc" "asdf" null] ["asdf"]]) {
		         name
		         age
		         posts {
		         	title
		         	author {
		         		namee
		         		age
		         	}
		         }
		         #other
		     }
		}
	`

	var expectedErr []string = []string{
		`Field "namee" is not a member of "author" (SDL Object "Person") at line: 8 column: 14`,
		`List cannot contain NULLs at line: 2 column: 58`,
		`Expected a Int for argument "last", got a List, at line: 2 column: 34`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}

	//p.ClearCache()
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)
	//
	// teardown
	//
	{
		inputSDL := `type Query {allPersons  (  last : Int     first : Int   ) : [Person!] }`
		teardown(inputSDL, t)
	}
}

func TestQuerySingleResolverLast2a(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
				schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				type Person {name : String! age  (  ScaleBy : Float =1.19  ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [1 2 3] ) : [Post!] }
				type Query {allPersons  (  last : Int     first : Int = 2  ) : [Person!] }`
		setup(inputSDL, t)
	}

	//
	// Test
	//

	var input = `query XYZ3 {
	     allPersons(last: 2 ) {
	         name 
	         age
	         WhatAmIReading: posts {
	         	title
	         	author {
	         		name
	         		age
	         	}
	         }
	         #other
	     }
	}
`
	//queryXYZ3{allPersons(last:2first:2){nameage(ScaleBy:1.3)WhatAmIReading:posts(resp:[123]){titleauthor{nameage(ScaleBy:1.2)}}}}
	expectedResult := `
	{
	 data: {
	 allPersons : [
	 {
	 name : "Jack Smith"
	 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ]
	 WhatAmIReading : [
	         {
	         title : "GraphQL for Begineers"
	         author : [
	                 {
	                 name : "Jack Smith"
	                 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ]
	                 } ]
	         }
	         {
	         title : "Holidays in Tuscany"
	         author : [
	                 {
	                 name : "Jenny Hawk"
	                 age : [  [  25 26 27 ]  [  44 45 46 ]  ]
	                 } ]
	         }
	         {
	         title : "Sweet"
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ]
	                 } ]
	         } ]
	 }
	 {
	 name : "Jenny Hawk"
	 age : [  [  25 26 27 ]  [  44 45 46 ]  ]
	 WhatAmIReading : [
	         {
	         title : "Sweet"
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ]
	                 } ]
	         }
	         {
	         title : "How to Eat"
	         author : [
	                 {
	                 name : "Kathlyn Host"
	                 age : [  [  33, 32, 31]  [ 33, 32, 31 ]  ]
	                 } ]
	         }
	         {
	         title : "Programming in GO"
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ]
	                 } ]
	         } ]
	 }  ]
	}
	}`

	var expectedErr []string

	var expectedDoc string = `query XYZ3 {
	 allPersons(last:2first:2) {
		name
		age(ScaleBy:1.19)
		WhatAmIReading : posts(resp:[123]) { 
											 title
											 author { 
		    										name
		    										age(ScaleBy:1.19)
													}
											}
								}
		}
	`
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// register resolvers - this would normally be populated by the client and resolverMap passed to server
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}
	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
		t.Log(doc.String())
	}
	//
	if len(errs) == 0 {

		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
	//
	// Teardown
	//
	{
		inputSDL := `type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}
}

func TestQuerySingleResolverLast2b(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				type Person {name : String! age  (  ScaleBy : Float =1.3  ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [1 2 3] ) : [Post!] }
				type Query {allPersons  (  last : Int     first : Int = 2  ) : [Person!] }
				`
		setup(inputSDL, t)
	}

	//
	// Test
	//

	var input = `query {
	     allPersons(last: 2 ) {
	         name 
	         age
	         WhatAmIReading: posts {
	         	author {
	         		name
	         	}
	         }
	     }
	}`

	expectedResult := `
	{
	 data: {
	 allPersons : [
	 {
	 name : "Jack Smith"
	 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ]
	 WhatAmIReading : [
	         {
	         author : [
	                 {
	                 name : "Jack Smith"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Jenny Hawk"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 } ]
	         } ]
	 }
	 {
	 name : "Jenny Hawk"
	 age : [  [  25 26 27 ]  [  44 45 46 ]  ]
	 WhatAmIReading : [
	         {
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Kathlyn Host"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 } ]
	         } ]
	 }  ]
	}
	}`

	var expectedErr []string
	var expectedDoc = `query{allPersons(last:2first:2){nameage(ScaleBy:1.3)WhatAmIReading:posts(resp:[123]){author{name}}}}`
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// register resolvers - this would normally be populated by the client and resolverMap passed to server
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}
	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(expectedDoc, doc.String()) {
			t.Logf("Got:      [%s] \n", trimWS(expectedDoc))
			t.Logf("Expected: [%s] \n", trimWS(doc.String()))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}
	//
	if len(errs) == 0 {

		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestQuerySingleResolverLast1(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				type Person {name : String! age  (  ScaleBy : Float =1.3  ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [1 2 3] ) : [Post!] }
				type Query {allPersons  (  last : Int     first : Int = 2  ) : [Person!] }
				`
		setup(inputSDL, t)
	}

	//
	// Test
	//

	var input = `query {
	     allPersons(last: 1 ) {
	         name 
	         age
	         WhatAmIReading: posts {
	         	author {
	         		name
	         	}
	         }
	     }
	}`

	expectedResult := `
	{
	 data: {
	 allPersons : [
	 {
	 name : "Jack Smith"
	 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ]
	 WhatAmIReading : [
	         {
	         author : [
	                 {
	                 name : "Jack Smith"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Jenny Hawk"
	                 } ]
	         }
	         {
	         author : [
	                 {
	                 name : "Sabastian Jackson"
	                 } ]
	         } ]
	 }]}
	}`

	var expectedErr []string
	var expectedDoc string = `query{allPersons(last:1first:2){nameage(ScaleBy:1.3)WhatAmIReading:posts(resp:[123]){author{name}}}}`
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// register resolvers - this would normally be populated by the client and resolverMap passed to server
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}
	doc, errs := p.ParseDocument(schema)
	//
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}
	//
	if len(errs) == 0 {

		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
	//
	// Teardown
	//
	{
		inputSDL := `type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}
}

func TestQueryTwoResolver_43(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `
				schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				type Post {title : String! title2 : String! author : [Person!]! }
				type Person {name : String! age  (  ScaleBy : Float =3.2  ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [1 2 3] ) : [Post!] }
				type Query {allPersons  (  last : Int     first : Int   ) : [Person!] }`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ {
	     allPersons(first : 3 last: 2 ) {
	         name 
	         age
	         WhatAmIReading: posts { 
	         	title
	         	author  {
	         		name
	         		age
	         	}
	         }
	         #age
	     }
	}
`

	expectedErr := []string{} //`Expected single value got List for Post at line: 6 column: 27`}

	expectedResult := `        {
        data: {
         allPersons : [ 
         {
         name : "Jack Smith"
         age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ] 
         WhatAmIReading : [ 
                 {
                 title : "GraphQL for Begineers"
                 author : [ 
                         {
                         name : "Jack Smith"
                         age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ] 
                         } ] 
                 }
                 {
                 title : "Holidays in Tuscany"
                 author : [ 
                         {
                         name : "Jenny Hawk"
                         age : [  [  25 26 27 ]  [  44 45 46 ]  ] 
                         } ] 
                 }
                 {
                 title : "Sweet"
                 author : [ 
                         {
                         name : "Sabastian Jackson"
                         age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                         } ] 
                 }  ]
         }
         {
         name : "Jenny Hawk"
         age : [  [  25 26 27 ]  [  44 45 46 ]  ] 
         WhatAmIReading : [ 
                 {
                 title : "Sweet"
                 author : [ 
                         {
                         name : "Sabastian Jackson"
                         age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                         } ] 
                 }
                 {
                 title : "How to Eat"
                 author : [ 
                         {
                         name : "Kathlyn Host"
                         age : [  [  33 32 31 ]  [  33 32 31 ]  ] 
                         } ] 
                 }
                 {
                 title : "Programming in GO"
                 author : [ 
                         {
                         name : "Sabastian Jackson"
                         age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                         } ] 
                 }  ]
         }  ]
        }
        }`
	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolvePartial); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/allPersons/posts", client.ResolvePosts); err != nil {
		p.addErr(err.Error())
	}
	_, errs := p.ParseDocument(schema)
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
	//
	// Teardown
	//
	{
		inputSDL := `type Query {allPersons(last : Int  first : Int) : [Person!]}`
		teardown(inputSDL, t)
	}
}

func TestQueryTwinResolverX(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `
				schema {
				query : Query 
				mutation : Mutation
				subscription : Subscription
				}
				type Query {allPersons  (  last : Int     first : Int =2  ) : Person! }
				type Person {name : String! age  (  ScaleBy : Float   ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [2 3 4] ) : [Post!] }`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query XYZ {
	     allPersons(last: 2 ) {
	         name 
	         age(ScaleBy: 10.)
	         WhatAmIReading: posts { # error here... Type definition lists single value, resolver returns List	  posts (resp: [Int!]) : Post! 
	         	title
	         	author  { # type Person, AST List_ of Object (person)
	         		name
	         		age(ScaleBy: 3.)
	         	}
	         }
	         #other
	     }
	}
`

	var expectedErr []string
	var expectedResult string

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolvePartial); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/allPersons/posts", client.ResolvePosts); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/allPersons/age", client.ResolveAge); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/allPersons/posts/author/age", client.ResolveAge); err != nil {
		p.addErr(err.Error())
	}
	_, errs := p.ParseDocument(schema)
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {

		expectedErr := []string{`Resolver returned a list, expected a single item for Person at line: 2 column: 7`}

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestQueryFieldCheckWithWrongName(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
						type Query {allPersons  (  last : Int =3    first : Int =2  ) : Person! }
				type Person {name : String! age  (  ScaleBy : Float =33.3  ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  = [2 3 4] ) : [Post!] }`
		setup(inputSDL, t)
	}

	var input = `query XYZ {
	     allPersons(last: 2 ) {
	         name 
	         age
	         posts {
	         	title
	         	author {
	         		namee
	         	}
	         }
	     }
	}
`

	expectedErr := []string{`Field "namee" is not a member of "author" (SDL Object "Person") at line: 8 column: 13`}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
}

func TestQueryFieldCheckWithInlineFragmentNoDirective_44(t *testing.T) {

	//
	// Setup
	//
	{
		inputSDL := `
		type Query {allPersons(last : Int = 1 first : Float = 3.2) : [Person!]}
		
		type Person {name : String! age(ScaleBy : Float = 3.2) : [[Int!]]! other : [String!] posts(resp :  Int! = 3) : [Post!]}
		
		type Post {title : String! title2 : String! author : [Person!]!}`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ ($expandedInfo: Boolean = true) {
	     allPersons(last: 2) {
	         name 
	         age
	         ... @include {						# inline fragment
	         	posts (resp: 3) {
	         		 author {
	         	 		name
	         	 		age
	         	     }
	         	     address
	        	 }
	           }
	     }
	}
`

	var expectedErr []string = []string{
		`Field "address" is not a member of "posts" (SDL Object "Post") at line: 12 column: 17`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	// if len(errs) == 0 {
	// 	if compare(doc.String(), expectedDoc) {
	// 		t.Logf("Got:      [%s] \n", trimWS(doc.String()))
	// 		t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
	// 		t.Errorf(`Unexpected document for %s. `, t.Name())
	// 	}
	// }
	//
	// Teardown
	//
	{
		inputSDL := `type Query {allPersons(last : [Int]  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}
}

func TestQueryCoerceInt2ListDepth1_45(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `	type Query {allPersons(last : Int = 1  first : Int = 3 ) : [Person!]}`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query XYZ {
	     allPersons(last: 2) {
	         name 
	     }
	}
`
	var expectedDoc string = `
		query XYZ {
		     allPersons(last:2 first:3)  {
		         name
		     }
		}
	`
	var expectedErr []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	doc, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
		t.Log(doc.String())
	}
	//
	// TearDown
	//
	{
		inputSDL := `	type Query {allPersons(last : Int = 1  first : Int = 3 ) : [Person!]}`
		teardown(inputSDL, t)
	}
}

func TestQueryCoerceInt2ListDepth2_46(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `	type Query {allPersons(last : Int = 1 first : Int = 3 ) : [Person!]}`
		setup(inputSDL, t)
	}

	var input string = `
	query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: 2) {
	         name 
	     }
	}
`
	var expectedDoc string = `
		query XYZ ($expandedInfo: Boolean = false) {
		     allPersons(last: 2 first :3) {
		         name
		     }
		}
	`
	var expectedErr []string = []string{}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}

	//p.ClearCache()
	doc, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}

}

func TestQueryCoerceInt2ListDepth3_47(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `	type Query {allPersons(last : Int = 1 first : Int = 3 ) : [Person!]}`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input string = `
	query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: 3) {
	         name 
	     }
	}
`
	var expectedDoc string = `
		query XYZ ($expandedInfo: Boolean = false) {
		     allPersons(last: 3 first: 3) {
		         name
		     }
		}
	`
	var expectedErr []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	//p.ClearCache()
	doc, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(doc.String(), expectedDoc) {
			t.Logf("Got:      [%s] \n", trimWS(doc.String()))
			t.Logf("Expected: [%s] \n", trimWS(expectedDoc))
			t.Errorf(`Unexpected document for %s. `, t.Name())
		}
	}
	//
	// Teardown
	//
	{

		inputSDL := `	type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}

}

func TestQueryCoerceDiffTypeListDepth3_48(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `	type Query {allPersons(last : String first : Int = 4 ) : [Person!]}`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input string = `
	query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: 12.2 ) {
	         name 
	     }
	}
`
	var expectedErr []string = []string{
		`Required type for argument "last" is String, got Float at line: 3 column: 18`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	//
	// Teardown
	//
	{

		inputSDL := `	type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}

}

func TestQueryDiffTypeListDepth3_49(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `type Query {allPersons(last : [[[String!]]]  first : [[String!]] ) : [Person!]}`
		setup(inputSDL, t)
	}

	var input string = `
	query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: [[[4.4]]] first : [["ABC"]]) {
	         name 
	     }
	}
`
	var expectedErr []string = []string{
		`Required type "String", got "Float" at line: 3 column: 27`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	//
	// Teardown
	//
	{

		inputSDL := `	type Query {allPersons(last : Int  first : [[String!]] ) : [Person!]}`
		teardown(inputSDL, t)
	}

}

func TestQueryInvalidArguments_45(t *testing.T) {

	{
		//
		// Setup for 44
		//
		inputSDL := `type Person {name : String! age(ScaleBy : Float ) : [[Int!]]! other : [String!] posts(resp :  [Int!] ) : [Post!]}
					type Query {allPersons(last : [Int]  first : [[String!]] ) : [Person!]}`
		setup(inputSDL, t)
	}

	var input = `query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: 2) {
	         name 
	         age
	         ... {						# inline fragment
	         	posts (author: $expandedInfo) {
	         		author (name: "abc" age: 234) {
	         	 		name
	         	 		age
	         	     }
	         	  address
	        	 }
	           }
	     }
	}
`
	expectedErr := []string{
		`Argument "first" must be defined (type "[[String!]]") at line: 2 column: 7`,
		`Argument "ScaleBy" must be defined (type "Float") at line: 4 column: 11`,
		`Enclosing type for an inline fragment field must be an Interface or Union if type on-condition specified or Object type if none. Got "Object" at line: 5 column: 13`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)

}

func TestQueryFieldCheckWithFragmentSpreadDirective(t *testing.T) {
	{
		//
		// Setup for 44
		//
		inputSDL := `type Person {name : String! age  (  ScaleBy : Float ) : [[Int!]]! other : [String!] posts  (  resp : [Int!]  ) : [Post!] }
					type Query {allPersons(last : [Int]  first : [[String!]] ) : [Person!]}`
		setup(inputSDL, t)
	}
	var input = `query XYZ ($expandedInfo: Boolean = true) {
	     allPersons(last: 2) {
	         name 
	         ... @include(if: $expandedInfo){
	         	age
	         	posts {
	         	 author {
	         	 	name
	         	 	age
	         	        }
	         	  address
	         	      }
	            }
	     }
	}
`

	expectedErr := []string{
		`Field "address" is not a member of "posts" (SDL Object "Post") at line: 11 column: 14`,
		`Argument "first" must be defined (type "[[String!]]") at line: 2 column: 7`,
		`Argument "ScaleBy" must be defined (type "Float") at line: 5 column: 12`,
		`Argument "resp" must be defined (type "[Int!]") at line: 6 column: 12`,
		`Argument "ScaleBy" must be defined (type "Float") at line: 9 column: 14`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	_, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)

}
func TestQueryWithFragmentSpreadDirectiveFALSE(t *testing.T) {
	//
	// Setup
	//
	{

		inputSDL := `type Query {allPersons(last : Int first : Int ) : [Person!]}`
		setup(inputSDL, t)
	}

	var input = `query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(first: 3 last: 2) {
	         aliasN: name 
	         ... @include(if: $expandedInfo) {
	         	age (ScaleBy : 1.2)
	         	posts (resp: [ 12 32]){
	         	 title
	         	 author {
	         	 	name
	         	 	age(ScaleBy : 1.2)
	         	 }
	         	}
	         }
	     }
	}
`

	var expectedErr []string

	expectedResult := `{
        data: {
         allPersons : [ 
         {
         aliasN : "Jack Smith"
         }
         {
         aliasN : "Jenny Hawk"
         }  ]
        }
        }	`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// assign Resolver
	//

	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
			p.addErr(err.Error())
		}
		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestQueryInlineFragmentWithSameAttributeTrue(t *testing.T) {

	//
	// Setup
	//
	{
		inputSDL := `type Query {allPersons(last : Int = 2 first : Int = 5 ) : [Person!]}
		type Person {name : String! age(ScaleBy : Float =2.3) : [[Int!]]! other : [String!] posts(resp :  Int! ) : [Post!]}
		type Post {title : String! title2 : String! author : [Person!]!}`
		setup(inputSDL, t)
	}
	//
	//	Test
	//
	var input = `query XYZ ($expandedInfo: Boolean = true) {
		     allPersons(last: 2) {
		         aliasN: name
		         ... @include(if: $expandedInfo) {
		         	age
		         	posts (resp: 3){
		         	 title
		         	 author {
		         	 	name
		         	 	age
		         	 }
		         	}
		         }
		         age
		     }
		}
	`

	var expectedErr []string = []string{
		`Field "Person.Query/allPersons/age" has already been specified at line: 14 column: 12`,
	}

	expectedResult := ``

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// assign Resolver
	//
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
			p.addErr(err.Error())
		}
		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}

}
func TestQueryInlineFragmentWithSameAttributeFALSE(t *testing.T) {

	var input = `query XYZ ($expandedInfo: Boolean = false) {
		     allPersons(last: 2) {
		         aliasN: name
		         ... @include(if: $expandedInfo) {
		         	age
		         	posts {
		         	 title
		         	 author {
		         	 	name
		         	 	age
		         	 }
		         	}
		         }
		         age
		     }
		}
	`

	var expectedErr []string = []string{
		`Field "Person.Query/allPersons/age" has already been specified at line: 14 column: 12`,
		`Argument "resp" must be defined (type "Int!") at line: 6 column: 13`,
	}

	expectedResult := ``

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// assign Resolver
	//
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
			p.addErr(err.Error())
		}
		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}

}

func TestQueryWithFragmentSpreadDirectiveTRUE(t *testing.T) {

	var input = `query XYZ ($expandedInfo: Boolean = true) {
		     allPersons(last: 2) {
		         aliasN: name
		         ... @include(if: $expandedInfo) {
		         	age
		         	posts (resp: 34){
		         	 title
		         	 author {
		         	 	name
		         	 	age
		         	 }
		         	}
		         }
		     }
		}
	`

	var expectedErr []string

	expectedResult := `        {
        data: {
         allPersons : [ 
         {
         aliasN : "Jack Smith"
                 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ] 
                 posts : [ 
                         {
                         title : "GraphQL for Begineers"
                         author : [ 
                                 {
                                 name : "Jack Smith"
                                 age : [  [  53 54 55 56 ]  [  25 26 28 27 ]  ] 
                                 } ] 
                         }
                         {
                         title : "Holidays in Tuscany"
                         author : [ 
                                 {
                                 name : "Jenny Hawk"
                                 age : [  [  25 26 27 ]  [  44 45 46 ]  ] 
                                 } ] 
                         }
                         {
                         title : "Sweet"
                         author : [ 
                                 {
                                 name : "Sabastian Jackson"
                                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                                 } ] 
                         } ] 
         }
         {
         aliasN : "Jenny Hawk"
                 age : [  [  25 26 27 ]  [  44 45 46 ]  ] 
                 posts : [ 
                         {
                         title : "Sweet"
                         author : [ 
                                 {
                                 name : "Sabastian Jackson"
                                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                                 } ] 
                         }
                         {
                         title : "How to Eat"
                         author : [ 
                                 {
                                 name : "Kathlyn Host"
                                 age : [  [  33 32 31 ]  [  33 32 31 ]  ] 
                                 } ] 
                         }
                         {
                         title : "Programming in GO"
                         author : [ 
                                 {
                                 name : "Sabastian Jackson"
                                 age : [  [  44 45 46 ]  [  54 55 56 57 ]  ] 
                                 } ] 
                         } ] 
         }  ]
        }
        }`
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//
	// assign Resolver
	//
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
			p.addErr(err.Error())
		}
		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}

}
