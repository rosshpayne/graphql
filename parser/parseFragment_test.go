package parser

import (
	"testing"

	"github.com/rosshpayne/graph-sdl/db"
	"github.com/rosshpayne/graphql/client"
	"github.com/rosshpayne/graphql/lexer"
)

func TestFragmentx(t *testing.T) {
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
				
		type Query { hero(episode: Episode): [Character] 
					 droid(id: ID!): [Character] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

`
	var expectedResult string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] } `

	var expectedErr []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {

		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
			t.Log(d.String())
		} else {
			t.Log(errs)
		}
	}
}

//TODO: create a test for "... {" inline fragment using an object (?) enclosing type

func TestFragmentEmbeddedInlineWithWrongTypeCond(t *testing.T) {
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
				
		type Query { hero(episode: Episode): [Character] 
					 droid(id: Int): [Character] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: droid(id: 1 ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
  ... on Human {
  	totalCredits
  }
  ... on Person {
  	primaryFunction
  }
}

`

	var parseErrs []string = []string{
		`On condition type "Person" does not implement interface "Character", at line: 22 column: 5`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/droid", client.ResolverDroid); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
}

func TestFragmentEmbeddedInlineWithTypeCond(t *testing.T) {
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
				
		type Query { hero(episode: Episode): [Character] 
					 droid(id: Int): [Character] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: droid(id: 1 ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
  ... on Human {
  	totalCredits
  }
  ... on Droid {
  	primaryFunction
  }
}

`
	var expectedResult string = ` { data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
                 totalCredits : 5532
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
                 totalCredits : 2532
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
                 totalCredits : 5532
         appearsIn : [  NEWHOPE JEDI ] 
         }  ]
         rightComparison : [ 
         {
         name : "Dro-RK9"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
                         primaryFunction : "Diplomat"
         }
         {
         name : "Dro-P78"
         friends : [ 
                 {
                 name : "R2-D2"
                 }
                 {
                 name : "C-3PO"
                 } ] 
                         primaryFunction : "Multifunction"
         }  ] 
         ] }`

	var expectedErr []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/droid", client.ResolverDroid); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {

		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
			t.Log(d.String())
		} else {
			t.Log(errs)
		}
	}
}

func TestFragmentAttributeRepeated(t *testing.T) {

	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human] 
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}

	var input = ` query XYS
	{
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

	fragment comparisonFields on Character {
		 name
		 friends {
				  name
				}
		 appearsIn
	}

`

	expectedErr := []string{
		//	`Field "XXX" is not a member of "Character" (SDL Interface "Character") at line: 17 column: 4`,
		`Field "Human.Query/hero(middleComparision)/appearsIn" has already been specified at line: 8 column: 6`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		if compare(d.String(), input) {
			t.Errorf("Got:      [%s] \n", trimWS(d.String()))
			t.Errorf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected: program.String() wrong. `)
		}
	}

}

func TestFragmentInlineWithUnionEnclosingType(t *testing.T) {

	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode):  USearchResult
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }

		union USearchResult = Human | Droid

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`

		setup(inputSDL, t)
	}

	var input = ` query
	{
	  hero(episode: NEWHOPE) {
	  #name               - common fields should be specified outside of inline fragments. name is not in this case.
	  ... on Human {
	    name
	  	totalCredits
	  }
	... on Droid {
	  	name
	  	primaryFunction
	  }
	... {
	id
	}

	  }
	}
`

	var parseErrs []string = []string{
		`Expected a type on-condition as enclosing type, "USearchResult", is a Union, at line: 13 column: 4`,
		`Field "Droid.Query/hero/name" has already been specified at line: 10 column: 5`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")

	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	//
	// teardown
	//
	{
		input := `	type Query { hero(episode: Episode):  SearchResult
					 droid(id: ID!): [Droid] 
					}`
		teardown(input, t)
	}
}

func TestFragmentResolveRetList(t *testing.T) {

	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): Human 
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`

		setup(inputSDL, t)
	}

	var input = ` query
	{
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

	fragment comparisonFields on Character {
		 name
		 friends {
				  name
				}
	}

`

	var execErrs []string = []string{
		`Resolver returned a list, expected a single item for Human at line: 3 column: 20`,
		`Resolver returned a list, expected a single item for Human at line: 6 column: 23`,
		`Resolver returned a list, expected a single item for Human at line: 10 column: 21`,
	}
	var parseErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")

	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestMultiStmtConcurrent1(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: ID!): [Droid]
					}

		enum Episode { NEWHOPE EMPIRE JEDI }

		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query ABC {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	    totalCredits
	    starships
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query XYZ {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}


fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

`

	var execErrs []string
	var parseErrs []string
	var expectedResult1 string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
                 totalCredits : 5532
         starships : [ 
                 {
                 }
                 {
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         totalCredits : 2532
         starships : [ 
                 {
                 } ] 
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] 
		} `
	var expectedResult2 string = ` { data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] } `

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")

	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)

	if len(errs) == 0 {

		p.SetExecStmt("ABC")

		result, errs := p.ExecuteDocument()
		checkErrors(errs, execErrs, t)

		if compare(result, expectedResult1) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult1))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)

		t.Log("========================================================================================")
		p.SetExecStmt("XYZ")
		result, errs = p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult2) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult2))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentWithInterface(t *testing.T) {

	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Character] 
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`

		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

`
	var expectedResult string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] } `

	var expectedErr []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, expectedErr, t)
	if len(errs) == 0 {
		t.Log(d.String())
		var expectedErr []string

		result, errs := p.ExecuteDocument()

		checkErrors(errs, expectedErr, t)
		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentWithInterfaceNotSupported(t *testing.T) {

	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Character] 
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human  {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`

		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields 
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Human {
  name
  friends {
    name
  }
}`

	var parseErrs []string = []string{
		`Enclosing interface "Character" is not implemented in fragment "Human", at line: 3 column: 9`,
		`Enclosing interface "Character" is not implemented in fragment "Human", at line: 6 column: 9`,
		`Enclosing interface "Character" is not implemented in fragment "Human", at line: 10 column: 9`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)
		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
	//
	// Reset
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode): [Character] 
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human  implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		teardown(inputSDL, t)
	}

}

func TestFragmentDirectives(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Character]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	//
	// test
	//

	var input = `query ABCDEF {
	  leftComparison: hero(episode: NEWHOPE)  {
	    ...comparisonFields 
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}`

	var execErrs []string
	var parseErrs []string
	var expectedResult string = `        {
        data: {
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
        }
        }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestMultiStmtDuplicates(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
	query xyz {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query xyz2 {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query xyz {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query xyz2 {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}


fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

`

	var parseErrs []string = []string{
		`Duplicate statement name "xyz" at line: 26 column: 8`,
		`Duplicate statement name "xyz2" at line: 38 column: 8`,
		`Duplicate fragment name "comparisonFields" at line: 59 column: 10`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)

}

func TestMultiStmtNoDups(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	var input = `
	query xyz {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields2
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields2
	  }
	}
	query xyz1 {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query xyz1a {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}
	query xyz2 {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields2
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}


fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

fragment comparisonFields2 on Character {
  name
  friends {
    name
  }
}

`

	var expectedResult string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ]
         middleComparision : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] }`
	var parseErrs []string
	var execErrs []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
		p.SetExecStmt("xyz2")

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentNestedx(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode):[Human]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment nestedField2 on Character {
	name
}
fragment nestedField1 on Character {
	appearsIn
}
fragment comparisonFields on Character {
  ...nestedField1
  friends {
    name
  }
  ...nestedField2
}
`

	var parseErrs []string
	var execErrs []string
	var expectedResult string = ` { data : [ 
         leftComparison : [ 
         {
         appearsIn : [  NEWHOPE JEDI ] 
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         name : "Luke Skywalker"
         }
         {
         appearsIn : [  NEWHOPE EMPIRE ] 
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         name : "Leia Organa"
         }  ]
         middleComparision : [ 
         {
         appearsIn : [  NEWHOPE JEDI ] 
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         name : "Luke Skywalker"
         }  ]
         rightComparison : [ 
         {
         appearsIn : [  NEWHOPE EMPIRE ] 
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         name : "Leia Organa"
         }  ] 
         ] }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentNestedWithDupField(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode):[Human]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFields
	    appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment nestedField2 on Character {
	name
}
fragment nestedField1 on Character {
	appearsIn
}
fragment comparisonFields on Character {
  ...nestedField1
  friends {
    name
  }
  ...nestedField2
}
`

	var parseErrs []string = []string{
		`Field "Human.Query/hero(middleComparision)/appearsIn" has already been specified at line: 7 column: 6`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentNestedWrongFrag(t *testing.T) {
	{
		//
		// Setup
		//
		// note resolving of fragment spreads is performed during checkField as only then are all fragment stmts parsed.
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: ID!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	  }
	  middleComparision: hero(episode: JEDI ) {
	    ...comparisonFieldx1
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFieldx2
	  }
	}

fragment nestedField2 on Character {
	name
}
fragment nestedField1 on Character {
	appearsIn
}
fragment comparisonFields on Character {
  ...nestedField1
  friends {
    name
  }
  ...nestedField2
}
`

	var parseErrs []string = []string{
		`Fragment definition "comparisonFieldx1" not found at line: 6 column: 9`,
		`Fragment definition "comparisonFieldx2" not found at line: 9 column: 9`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentTypeCond1x(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: Int!): [Droid]
					}
		`
		setup(inputSDL, t)
	}

	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: EMPIRE) {
	   ...comparisonHuman
	}
	rightComparison: droid(id: 1) {
	   ...comparisonDroid
	}
	}


fragment comparisonHuman on Human {							
  ...comparisonCharacter
   totalCredits
}

fragment comparisonDroid on Droid {						
  ...comparisonCharacter
  primaryFunction
}

fragment comparisonCharacter on Character {
  name
  friends {
  	friendsName: name
  }
  appearsIn
}
`

	var parseErrs []string
	var execErrs []string
	var expectedResult string = ` { data : [ 
         leftComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 friendsName : "Luke Skywalker"
                 }
                 {
                 friendsName : "C-3PO"
                 }
                 {
                 friendsName : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE EMPIRE ] 
         totalCredits : 2532
         }  ]
         rightComparison : [ 
         {
         name : "Dro-RK9"
         friends : [ 
                 {
                 friendsName : "Leia Organa"
                 }
                 {
                 friendsName : "C-3PO"
                 }
                 {
                 friendsName : "R2-D2"
                 } ] 
         appearsIn : [  DRTYPE ] 
         primaryFunction : "Diplomat"
         }
         {
         name : "Dro-P78"
         friends : [ 
                 {
                 friendsName : "R2-D2"
                 }
                 {
                 friendsName : "C-3PO"
                 } ] 
         appearsIn : [  DRTYPE ] 
         primaryFunction : "Multifunction"
         }  ] 
         ] } `

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/droid", client.ResolverDroid); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentTypeCond1withErrs(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: ID!): [Droid]
					}
				
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}

	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: DRTYPE) {
		name
		...comparisonHuman
		#...comparisonDroid
		appearsIn
		}
	}
	
fragment comparisonHuman on Human {							
  ...comparisonCharacter
   totalCredits
}

fragment comparisonDroid on Droid {						
  ...comparisonCharacter
  primaryFunction
}

fragment comparisonCharacter on Character {
  name
  friends {
  	friendsName: name
  }
  appearsIn
}
`

	var parseErrs []string = []string{
		`"DRTYPE" is not a member of Enum type Episode at line: 2 column: 32`,
		`Field "Human.Query/hero(leftComparison)/appearsIn" has already been specified at line: 6 column: 3`,
		`Field "Human.Query/hero(leftComparison)/name" has already been specified at line: 21 column: 3`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	t.Log("Errors: ", errs)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestFragmentTypeCond2(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Human]
					 droid(id: Int!): [Droid]
					}
		`
		setup(inputSDL, t)
	}
	var input = `query ($expandedInfo: Boolean = true) {
	HumanComparison: hero(episode: JEDI) {
	   ...comparisonHuman
	}
	DroidComparison: droid(id : 1) {
	   ...comparisonDroid
	}	
	}


fragment comparisonHuman on Human {							
  ...comparisonCharacter
   totalCredits
}

fragment comparisonDroid on Droid {						
  ...comparisonCharacter
  primaryFunction
}

fragment comparisonCharacter on Character {
  name
  friends {
  	friendsName: name
  }
  appearsIn
}
`

	var parseErrs []string
	var execErrs []string
	var expectedResult string = `{ data : [ 
         HumanComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 friendsName : "Leia Organa"
                 }
                 {
                 friendsName : "C-3PO"
                 }
                 {
                 friendsName : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         totalCredits : 5532
         }  ]
         DroidComparison : [ 
         {
         name : "Dro-RK9"
         friends : [ 
                 {
                 friendsName : "Leia Organa"
                 }
                 {
                 friendsName : "C-3PO"
                 }
                 {
                 friendsName : "R2-D2"
                 } ] 
         appearsIn : [  DRTYPE ] 
         primaryFunction : "Diplomat"
         }
         {
         name : "Dro-P78"
         friends : [ 
                 {
                 friendsName : "R2-D2"
                 }
                 {
                 friendsName : "C-3PO"
                 } ] 
         appearsIn : [  DRTYPE ] 
         primaryFunction : "Multifunction"
         }  ] 
         ] } `

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/droid", client.ResolverDroid); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
}

func TestInlineFragmentTypeCondInterface(t *testing.T) {
	{
		//
		// Setup
		//
		inputSDL := `
		type Query { hero(episode: Episode): [Character]
					 droid(id: Int!): [Character]
					}
		type Droid {						# does not implement any interfaces
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}
		`
		setup(inputSDL, t)
	}
	var input = `query ($expandedInfo: Boolean = true) {
		leftComparison: hero(episode: EMPIRE) {
	   ...on Human { 
					name
					 friends {
  							friendsName: name
					}
					appearsIn
					totalCredits
					}
	}
		rightComparison: droid(id: 1) {
	   ...on Droid {
					name
					 friends {
  							friendsName: name
					}
					appearsIn
					primaryFunction	
			}
	}
	}
`

	var parseErrs []string = []string{
		`On condition type "Droid" does not implement interface "Character", at line: 13 column: 7`,
	}
	var execErrs []string
	var expectedResult string
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	if err := p.Resolver.Register("Query/droid", client.ResolverDroid); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)

		if len(errs) == 0 {
			if compare(result, expectedResult) {
				t.Errorf("Got:      [%s] \n", trimWS(result))
				t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
				t.Errorf(`Unexpected: JSON output wrong. `)
			}
			t.Log(result)
		}
	}
	//
	// teardown
	//
	inputSDL := `
		type Droid implements Character  {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}
	directive @include (arg1: Int = 34 arg2: Float ) on FIELD | INLINE_FRAGMENT 
	`
	teardown(inputSDL, t)
}

func TestInlineFragmentDirectivesFALSE(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode):  [Character] # TODO: create test for Character (no List)
					 droid(id: ID!): [Droid]
					}

		enum Episode { NEWHOPE EMPIRE JEDI }
		directive @include  (  if : Boolean  = false  ) on | INLINE_FRAGMENT| FIELD|FRAGMENT_SPREAD
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//
	var input = `query ($expandedInfo: Boolean = false) {
	leftComparison: hero(episode: NEWHOPE) {
	   ...comparisonFields								# fragment spread no directives
	  }
	  middleComparision: hero(episode: NEWHOPE ) {
	   ...@include (if: $expandedInfo) {				# inlinefragment no type-condition with directive
	    	...comparisonFields
	   }
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields @include(if: $expandedInfo) # fragment spread with directive (can be different to directives in fragment statement)
	  }
	}


fragment comparisonFields on Character {				# fragment stmt no directives
  name
  friends {
    name
  }
  appearsIn
}
`

	var expectedErr []string
	var expectedResult string = `{ data : [
	leftComparison : [
	{
	name : "Luke Skywalker"
	friends : [
	        {
	        name : "Leia Organa"
	        }
	        {
	        name : "C-3PO"
	        }
	        {
	        name : "R2-D2"
	        } ]
	appearsIn : [  NEWHOPE JEDI ]
	}
	{
	name : "Leia Organa"
	friends : [
	        {
	        name : "Luke Skywalker"
	        }
	        {
	        name : "C-3PO"
	        }
	        {
	        name : "R2-D2"
	        } ]
	appearsIn : [  NEWHOPE EMPIRE ]
	}  ]
	middleComparision : [
	{
	}
	{
	}  ]
	rightComparison : [
	{
	}  ]
	] }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
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

func TestInlineFragmentDirectivesDupField(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode):  [Character] # TODO: create test for Character (no List)
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//
	var input = `query ($expandedInfo: Boolean = false) {
	leftComparison: hero(episode: NEWHOPE) {
	   ...comparisonFields								# fragment spread no directives
	  }
	  middleComparision: hero(episode: NEWHOPE ) {
	   ...@include (if: $expandedInfo) {				# inlinefragment no type-condition with directive
	    	...comparisonFields
	   }
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields @include(if: $expandedInfo) # fragment spread with directive (can be different to directives in fragment statement)
	    MyName: name
	  }
	}


fragment comparisonFields on Character {				# fragment stmt no directives
  name
  friends {
    name
  }
  appearsIn
}
`

	var expectedErr []string = []string{
		`Field "Character.Query/hero(rightComparison)/name" has already been specified at line: 12 column: 14`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

}

func TestInlineFragmentDirectivesTRUE(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode):  [Character] # TODO: create test for Character (no List)
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	//
	// test
	//
	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: NEWHOPE) {
	   ...comparisonFields								# fragment spread no directives
	  }
	  middleComparision: hero(episode: NEWHOPE ) {
	   ...@include (if: $expandedInfo) {				# inlinefragment no type-condition with directive
	    	...comparisonFields
	   }
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields @include(if: $expandedInfo) # fragment spread with directive (can be different to directives in fragment statement)
	  }
	}


fragment comparisonFields on Character {				# fragment stmt no directives
  name
  friends {
    name
  }
  appearsIn
}
`

	var expectedErr []string
	var expectedResult string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE EMPIRE ] 
         }  ]
         middleComparision : [ 
         {
                 name : "Luke Skywalker"
                 friends : [ 
                         {
                         name : "Leia Organa"
                         }
                         {
                         name : "C-3PO"
                         }
                         {
                         name : "R2-D2"
                         } ] 
                 appearsIn : [  NEWHOPE JEDI ] 
         }
         {
                 name : "Leia Organa"
                 friends : [ 
                         {
                         name : "Luke Skywalker"
                         }
                         {
                         name : "C-3PO"
                         }
                         {
                         name : "R2-D2"
                         } ] 
                 appearsIn : [  NEWHOPE EMPIRE ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE EMPIRE ] 
         }  ] 
         ] }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, expectedErr, t)

	if len(errs) == 0 {
		if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
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

func TestFragmentChangeFieldOrder(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode):  [Character] # TODO: create test for Character (no List)
					 droid(id: ID!): [Droid] 
					}
		
		enum Episode { NEWHOPE EMPIRE JEDI }
	
		interface Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							}

		type Human implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							starships: [Starship]
							totalCredits: Int
							}

		type Droid implements Character {
							id: ID!
							name: String!
							friends: [Character]
							appearsIn: [Episode]!
							primaryFunction: String
							}`
		setup(inputSDL, t)
	}
	var input = `query {
	  leftComparison: hero(episode: NEWHOPE) {
	    ...comparisonFields
	            appearsIn
	  }
	  rightComparison: hero(episode: EMPIRE ) {
	    ...comparisonFields
	  }
	}

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}

`

	var parsedErrs []string
	var execErrs []string
	var expectedResult string = `{ data : [ 
         leftComparison : [ 
         {
         name : "Luke Skywalker"
         friends : [ 
                 {
                 name : "Leia Organa"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE JEDI ] 
         }
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         appearsIn : [  NEWHOPE EMPIRE ] 
         }  ]
         rightComparison : [ 
         {
         name : "Leia Organa"
         friends : [ 
                 {
                 name : "Luke Skywalker"
                 }
                 {
                 name : "C-3PO"
                 }
                 {
                 name : "R2-D2"
                 } ] 
         }  ] 
         ] }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestFragmentNotExists(t *testing.T) {

	var input = `query {
	  leftComparison: hero(episode: EMPIRE) {
	    ...comparisonFields
	  }
	  rightComparison: hero(episode: JEDI) {
	    ...comparisonFields2
	  }
	}

fragment comparisonFields on Character {
  name
  appearsIn
  friends {
    name
  }
}

`

	var expectedErr [1]string
	expectedErr[0] = `Fragment definition "comparisonFields2" not found at line: 6 column: 9`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()
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

func TestFragmentFieldErr(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		type Query { hero(episode: Episode):  [Character] # TODO: create test for Character (no List)
					 droid(id: ID!): [Character] 
					}`
		setup(inputSDL, t)
	}

	var input = `query {
  leftComparison: hero(episode: EMPIRE) {
    ...comparisonFields
  }
  rightComparison: hero(episode: JEDI) {
    ...comparisonFields
  }
	}

fragment comparisonFields on Character {
  name
  CTX
  appearsIn
  friends {
    name
    cars

  }
}

`

	var expectedErr []string = []string{
		`Field "CTX" is not a member of "Character" at line: 12 column: 3`,
		`Field "cars" is not a member of "friends" (SDL Interface "Character") at line: 16 column: 5`,
	}
	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()
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

func TestUnionTypeWithFieldErr(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		schema {
				query : SearchQuery 
				mutation : Mutation
				subscription : Subscription
				}
		
		type SearchQuery {firstSearchResult : SearchResult}
		
		union SearchResult =| Photo| Person 
		
		type Person {name : String! age  (  ScaleBy : Float   ) : Int! }`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query {
   firstSearchResult {
   name 
    ... on Person {    
			 name
			 age (ScaleBy:1.3)
    }
    ... on Photo {
          width
          height
	}
}
}
`

	var parsedErrs []string = []string{
		`As the enclosing type is a Union, expected a fragment to resolve the type, got a non-fragment instead, "name" at line: 3 column: 4`,
	}
	var execErrs []string
	var expectedResult string = `        {
        data: {
         firstSearchResult : {
                 name : "Ross Payne"
                 age : 61 }
        }
        }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("SearchQuery/firstSearchResult", client.ResolverHeroUnion); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)
	t.Log(errs)
	if len(errs) == 0 {

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestUnionType(t *testing.T) {
	//
	// Setup
	//
	{
		inputSDL := `
		schema {
				query : SearchQuery 
				mutation : Mutation
				subscription : Subscription
				}
		
		type SearchQuery {firstSearchResult : SearchResult}
		
		union SearchResult =| Photo| Person 
		
		type Person {name : String! age  (  ScaleBy : Float =3.4  ) : Int! }`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query {
   firstSearchResult {
    ... on Person {    
			 name
			 age
    }
    ... on Photo {
          width
          height
	}
}
}
`

	var parsedErrs []string
	var execErrs []string
	var expectedResult string = `        {
        data: {
         firstSearchResult : {
                 name : "Ross Payne"
                 age : 61 }
        }
        }`

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	if err := p.Resolver.Register("SearchQuery/firstSearchResult", client.ResolverHeroUnion); err != nil {
		p.addErr(err.Error())
	}
	p.SetDocument("DefaultDoc")
	_, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {

		result, errs := p.ExecuteDocument()

		checkErrors(errs, execErrs, t)
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
	}
}

func TestFragmentRootInterface(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}

		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query {
					 item {
							entity {
								 name
    								... on Person {
    											age
    											}
									},
							phoneNumber
					}
			}`

	var parsedErrs []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestFragmentRootInterfaceErr1(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}

		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `query {
					 item {
							entity {
								name
    							age
								},
							
							phoneNumber
					}
			}`

	var parsedErrs []string = []string{
		`Field "age" is not a member of "entity" (SDL Interface "NamedEntity") at line: 5 column: 12`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithFragmentNotImplemented(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		#type Business implements NamedEntity & ValuedEntity {
		type Business implements ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								... BizXYZ
								},
							phoneNumber
					}
			}
			
		fragment BizXYZ on Business {
			name
			value
		#	... on Business {
		#				value
		#			}
		}
		
		`

	var parsedErrs []string = []string{
		`Enclosing interface "NamedEntity" is not implemented in fragment "Business", at line: 5 column: 13`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithFragmentIsImplementedFieldErr(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								... BizXYZ
								},
							phoneNumber
					}
			}
			
		fragment BizXYZ on Business {
			name
			value
			XYZ
		#	... on Person {
		#				value
		#			}
		}
		
		`

	var parsedErrs []string = []string{
		`Field "XYZ" is not a member of "Business" at line: 14 column: 4`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithFragmentIsImplementedInlineErr(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								... BizXYZ
								},
							phoneNumber
					}
			}
			
		fragment BizXYZ on Business {
			name
			value
			... on Person {
						value
					}
		}
		
		`

	var parsedErrs []string = []string{
		`Enclosing type for an inline fragment field must be an Interface or Union if type on-condition specified or Object type if none. Got "Object" at line: 14 column: 6`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithFragmentIsImplementedDupFields(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								name 
								... BizXYZ
								},
							phoneNumber
					}
			}
			
		fragment BizXYZ on Business {
			name
			employeeCount
		}
		
		`

	var parsedErrs []string = []string{
		`Field "Business.Query/item/entity/name" has already been specified at line: 13 column: 4`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithOnlineFragmentDupFields(t *testing.T) {
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
		
		type Query {item : Contact}
		
		interface NamedEntity {
					name: String
		}

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: NamedEntity
				 phoneNumber: String
				 address: String
		}
		
`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								name 
								... on Business {
									name
									employeeCount
								}
								},
							phoneNumber
					}
			}
		
		`

	var parsedErrs []string = []string{
		`Field "Business.Query/item/entity/name" has already been specified at line: 7 column: 10`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithOnlineFragNotImplement(t *testing.T) {
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
		
		type Query {item : Contact}

		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type XBusinessY implements ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
`
		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								name 
								... on XBusinessY {
									name
								}
								},
							phoneNumber
					}
			}
		
		`

	var parsedErrs []string = []string{
		`On condition type "XBusinessY" does not implement interface "NamedEntity", at line: 6 column: 11`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestRootInterfaceWithOnlineXX(t *testing.T) {
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
		
		type Query {item : Contact}
		

		interface ValuedEntity {
					 value: Int
		}

		type Person implements NamedEntity {
				 name: String
				 age: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: Person
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query {
					 item {
							entity {
								... on Person {
									name
									age
								}
								},
							phoneNumber
					}
			}
		
		`

	var parsedErrs []string = []string{
		`Enclosing type for an inline fragment field must be an Interface or Union if type on-condition specified or Object type if none. Got "Object" at line: 5 column: 11`,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)

	if len(errs) == 0 {
		t.Log(d.String())
	}

}

func TestFragementDirectivesNotExist(t *testing.T) {
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
		
		type Query  {item  : Contact}
		

		interface ValuedEntity {
					name: String
					 value: Int
		}
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: ValuedEntity
				 phoneNumber: String
				 address: String
		}
		
`
		setup(inputSDL, t)

		err := db.DeleteType("IncludeX")
		if err != nil {
			t.Errorf(`Not expected Error =[%q]`, err.Error())
		}
		err = db.DeleteType("IncludeY")
		if err != nil {
			t.Errorf(`Not expected Error =[%q]`, err.Error())
		}
	}
	//
	// Test
	//
	// all these not-exist errors are caught in resolveDependents(), well before checkFields().
	//
	var input = `
		query xyz ($info: Boolean = true) @   includeR @ includeP {
					 item {
							entity @includeT {
								name @  includeZ (abc: 23 if: $info iff : 33)
								... @includeX (if: $info iff : 33)  @ includeY (Name: "Ross") {
									value
								}
								},
							phoneNumber @includeP
					}
			}
		
		`
	var parsedErrs []string = []string{
		`"@includeR" does not exist in document "DefaultDoc" at line: 2 column: 41`,
		`"@includeX" does not exist in document "DefaultDoc" at line: 6, column: 14`,
		`"@includeY" does not exist in document "DefaultDoc" at line: 6, column: 47`,
		`"@includeT" does not exist in document "DefaultDoc" at line: 4 column: 16`,
		`"@includeZ" does not exist in document "DefaultDoc" at line: 5 column: 17`,
		`"@includeP" does not exist in document "DefaultDoc" at line: 10 column: 21`,
		`"@includeP" does not exist in document "DefaultDoc" at line: 2 column: 52 `,
	}

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
		t.Log(errs)
	}
}

func TestFragementDirectiveExist(t *testing.T) {
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
		
		type Query  {item  : Contact}
		

		interface ValuedEntity {
					name: String
					 value: Int
		}

		directive @include (if : Boolean = false) on INLINE_FRAGMENT | FIELD
		
		type Business implements NamedEntity & ValuedEntity {
				name: String
				value: Int
				employeeCount: Int
		}
		
		type Contact {
				 entity: ValuedEntity
				 phoneNumber: String
				 address: String
		}
		
`

		setup(inputSDL, t)
	}
	//
	// Test
	//
	var input = `
		query xyz ($info: Boolean = true) {
					 item {
							entity {
								name
								... @include (if: $info) {
									value
								}
								},
							phoneNumber
					}
			}
		
		`

	var parsedErrs []string

	l := lexer.New(input)
	p := New(l)
	p.ClearCache()

	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()

	checkErrors(errs, parsedErrs, t)
	if len(errs) == 0 {
		t.Log(d.String())
	}
}
