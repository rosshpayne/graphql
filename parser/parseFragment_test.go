package parser

import (
	"fmt"
	"testing"

	"github.com/graphql/client"
	"github.com/graphql/lexer"
)

func TestFragmentx(t *testing.T) {

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

func TestShorthandOp(t *testing.T) {

	var input = ` {
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
	if compare(d.String(), input) {
		t.Errorf("Got:      [%s] \n", trimWS(d.String()))
		t.Errorf("Expected: [%s] \n", trimWS(input))
		t.Errorf(`Unexpected: program.String() wrong. `)
	}
}

func TestMultiStmt1(t *testing.T) {

	var input = `
	query ABC {
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
	query XYZ {
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
	if compare(d.String(), input) {
		t.Errorf("Got:      [%s] \n", trimWS(d.String()))
		t.Errorf("Expected: [%s] \n", trimWS(input))
		t.Errorf(`Unexpected: program.String() wrong. `)
	}
}

func TestMultiStmtDuplicates(t *testing.T) {

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

	var expectedErr [3]string
	expectedErr[0] = `Duplicate statement name "xyz" at line: 26 column: 8`
	expectedErr[1] = `Duplicate statement name "xyz2" at line: 38 column: 8`
	expectedErr[2] = `Duplicate fragment name "comparisonFields" at line: 59 column: 10`

	l := lexer.New(input)
	p := New(l)

	if err := p.Resolver.Register("Query/hero", client.ResolverHero); err != nil {
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

}

func TestMultiStmtNoDups(t *testing.T) {

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

fragment comparisonFields2 on Character {
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
	p.SetExecStmt("xyz2")
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

}

func TestFragmentNested(t *testing.T) {

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

func TestFragmentTypeCond1(t *testing.T) {

	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: DRTYPE) {
	   ...comparisonHuman
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

	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)

	if err := p.Resolver.Register("Query/hero", client.ResolverHero2); err != nil {
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

func TestInlineFragmentTypeCond1(t *testing.T) {

	var input = `query ($expandedInfo: Boolean = true) {
	leftComparison: hero(episode: DRTYPE) {
	   ...on Human { 
					name
					 friends {
  							friendsName: name
					}
					appearsIn
					totalCredits
					}
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

	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)

	if err := p.Resolver.Register("Query/hero", client.ResolverHero2); err != nil {
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

func TestInlineFragmentDirectives(t *testing.T) {

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
	expectedErr[0] = `Associated Fragment definition "comparisonFields2" not found in document at line: 6 column: 8`

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
