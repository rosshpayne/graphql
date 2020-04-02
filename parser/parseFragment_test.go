package parser

import (
	"fmt"
	"testing"

	"github.com/graphql/client"
	"github.com/graphql/lexer"
)

func TestFragmentx(t *testing.T) {
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
		if compare(result, expectedResult) {
			t.Errorf("Got:      [%s] \n", trimWS(result))
			t.Errorf("Expected: [%s] \n", trimWS(expectedResult))
			t.Errorf(`Unexpected: JSON output wrong. `)
		}
		t.Log(result)
		t.Log(d.String())
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

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")

	d, errs := p.ParseDocument()
	checkErrors(errs, parseErrs, t)
	t.Log(d.String())
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

func TestMultiStmt1(t *testing.T) {
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

fragment comparisonFields on Character {
  name
  friends {
    name
  }
}`

	var execErrs []string = []string{
		`Response type "Human" does not implement interface "Character" at line: 19, column: 1`,
	}
	var parseErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)

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
		`Associated Fragment definition "comparisonFieldx1" not found at line: 6 column: 9`,
		`Associated Fragment definition "comparisonFieldx2" not found at line: 9 column: 9`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)

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
		`
		setup(inputSDL, t)
	}

	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: DRTYPE) {
	   ...comparisonHuman
	   ...comparisonDroid
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
		`Field "Human.Query/hero(leftComparison)/name" has already been specified at line: 20 column: 3`,
		`Field "Character.Query/hero(leftComparison)/friends/name(friendsName)" has already been specified at line: 22 column: 17`,
		`Field "Human.Query/hero(leftComparison)/appearsIn" has already been specified at line: 24 column: 3`,
		`Field "primaryFunction" is not a member of "hero(leftComparison)" (SDL Object "Human") at line: 16 column: 3`,
		`Field "Human.Query/hero(leftComparison)/appearsIn" has already been specified at line: 5 column: 5`,
	}
	var execErrs []string
	var expectedResult string

	l := lexer.New(input)
	p := New(l)

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
		type Droid {
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
		`Field "totalCredits" is not a member of "hero(leftComparison)" (SDL Interface "Character") at line: 9 column: 6`,
		`On condition type for  "Droid" does not implement interface "Character", at line: 13 column: 11`,
	}
	var execErrs []string
	var expectedResult string
	l := lexer.New(input)
	p := New(l)

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
		`
	teardown(inputSDL, t)
}

func TestInlineFragmentDirectives(t *testing.T) {
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
	   ...@include (if: $expandedInfo) {				# inlinefragment (ie. no fragment name) with directive
	    	...comparisonFields
	   }
	   	appearsIn
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
         MyName : "Leia Organa"
         }  ] 
         ] } `

	l := lexer.New(input)
	p := New(l)

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

	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	fmt.Println(d.String())
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
	if compare(d.String(), input) {
		t.Errorf("Got:      [%s] \n", trimWS(d.String()))
		t.Errorf("Expected: [%s] \n", trimWS(input))
		t.Errorf(`Unexpected: program.String() wrong. `)
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
	expectedErr[0] = `Associated Fragment definition "comparisonFields2" not found in document at line: 6 column: 9`

	l := lexer.New(input)
	p := New(l)

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

	var expectedErr [2]string
	expectedErr[0] = `Field "CTX" is not in Interface "Character" at line: 12 column: 3`
	expectedErr[1] = `Field "cars" is not in Interface "Character" at line: 16 column: 5`

	l := lexer.New(input)
	p := New(l)

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

func TestUnionType(t *testing.T) {

	var input = `query {
  hero (episode: JEDII) {
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
}
`

	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)

	if err := p.Resolver.Register("Query/hero", client.ResolverHeroUnion); err != nil {
		p.addErr(err.Error())
	}
	//	p.ClearCache()
	p.SetDocument("DefaultDoc")
	d, errs := p.ParseDocument()
	if d != nil {
		fmt.Println(d.String())
	}
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
	if d != nil {
		if compare(d.String(), input) {
			t.Errorf("Got:      [%s] \n", trimWS(d.String()))
			t.Errorf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected: program.String() wrong. `)
		}
	} else {
		t.Errorf("Error in creating statement")
	}
}
