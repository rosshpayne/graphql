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
  rightComparison: hero(episode: EMPIRE ) {
    ...comparisonFields
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