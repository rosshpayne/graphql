package parser

import (
	"testing"

	"github.com/graphql/client"
	"github.com/graphql/lexer"
)

func TestQueryFieldCheck(t *testing.T) {

	var input = `query XYZ {
	     allPersons(last: 2) {
	         name 
	         age
	         posts {
	         	title
	         	author {
	         		name
	         	}
	         		
	         }
	     }
	}
`

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()
	//fmt.Println(d.String())
	if len(errs) > 0 {
		t.Errorf("Unexpected, should be 0 errors, got %d", len(errs))
		for _, v := range errs {
			t.Errorf(`Unexpected error: %s`, v.Error())
		}
	} else {
		if compare(d.String(), input) {
			t.Errorf("Got:      [%s] \n", trimWS(d.String()))
			t.Errorf("Expected: [%s] \n", trimWS(input))
			t.Errorf(`Unexpected: program.String() wrong. `)
		}
	}
}

func TestQueryOutput(t *testing.T) {

	var input = `query XYZ {
	     allPersons(last: [ 1 23 43] first: [["abc" "asdf" null] ["asdf"]]) {
	         name 
	         age
	         posts {
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

	var expectedErr [1]string
	expectedErr[0] = `Field "namee" is not in object , Person at line: 8 column: 13`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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

func TestQueryOutput2(t *testing.T) {

	var input = `query XYZ {
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

	var expectedErr [1]string
	expectedErr[0] = `asdf`

	schema := "DefaultDoc"
	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
	//
	if err := p.Resolver.Register("Query/allPersons", client.ResolverAll); err != nil {
		p.addErr(err.Error())
	}
	_, errs := p.ParseDocument(schema)
	//
	// register resolvers - this would normally be populated by the client and resolverMap passed to server

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

func TestQueryFieldCheckWithWrongName(t *testing.T) {

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

	var expectedErr [1]string
	expectedErr[0] = `Field "namee" is not in object , Person at line: 8 column: 13`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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

func TestQueryFieldCheckWithFragmentSpreadNoDirective(t *testing.T) {

	var input = `query XYZ ($expandedInfo: Boolean = false) {
	     allPersons(last: 2) {
	         name 
	         age
	         ... {
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

	var expectedErr [1]string
	expectedErr[0] = `Field "address" is not in object, Post at line: 11 column: 14`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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

func TestQueryFieldCheckWithFragmentSpreadDirective(t *testing.T) {

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

	var expectedErr [1]string
	expectedErr[0] = `Field "address" is not in object, Post at line: 11 column: 14`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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
func TestQueryWithFragmentSpreadDirectiveFALSE(t *testing.T) {

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

	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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

func TestQueryWithFragmentSpreadDirectiveTRUE(t *testing.T) {

	var input = `query XYZ ($expandedInfo: Boolean = true) {
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

	var expectedErr [1]string
	expectedErr[0] = `Field "Person.age" has already been specified at line: 14 column: 11`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
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
