package parser

import (
	"fmt"
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

func TestXQGood(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
    profilePic(size: $devicePicSize) {
    	aa
    	bb
    	cc {
    		add: ddd
    		aee: eee ( f : "abcdef-hij" )
    	}
    	ff
    	gghi
    	xyz
    	dog
    }
}
}
`

	l := lexer.New(input)
	p := New(l)
	d, errs := p.ParseDocument()
	fmt.Printf(`Doc: [%s]\n`, d.String())
	if len(errs) > 0 {
		t.Errorf("Unexpected, should be 0 errors, got %d", len(errs))
		for _, v := range errs {
			t.Errorf(`Unexpected error: %s`, v.Error())
		}
	}
	if compare(d.String(), input) {
		t.Errorf("Got:      [%s] \n", trimWS(d.String()))
		t.Errorf("Expected: [%s] \n", trimWS(input))
		t.Errorf(`Unexpected: program.String() wrong. `)
	}

}

func TestXQMissingRPAREN(t *testing.T) {
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

	expectedErr := `Expected an argument name followed by colon got an "} ff" at line: 10, column: 35`

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	//fmt.Println(d.String())
	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQMisplacedVariable(t *testing.T) {
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
	expectedErr := `Expected an argument name followed by colon got an "$ devicePicSize" at line: 7, column: 9`

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQArgumentNull(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: [Int!] = [1234 23 234 32 null] ) {
  xyzalias: user(id: 4) {
    sex
    Person {
    	Name
    	Age
    }
  }
}
`
	var expectedErr [1]string
	expectedErr[0] = `List cannot contain NULLs at line: 1 column: 63`

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

func TestXQNoName1(t *testing.T) {
	var input = `query  {
  xyzalias: user(id: 4) {
    Person {
    	Name
    	Age
    }
  }
}
`
	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
	d, errs := p.ParseDocument()
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
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(input))
		t.Errorf(`*************  program.String() wrong.`)
	}
}

func TestXQNoName2(t *testing.T) {
	var input = `query ($devicePicSize: [Int!] = [1234 23 234 32] ) {
  xyzalias: user(id: 4) {
    Person {
    	Name
    	Age
    }
  }
}
`
	var expectedErr [1]string
	expectedErr[0] = ``

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
	d, errs := p.ParseDocument()
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
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(input))
		t.Errorf(`*************  program.String() wrong.`)
	}
}

func TestXQNoName3(t *testing.T) {
	var input = `query ($devicePicSize: Int! = null ) {
  xyzalias: user(id: 4) {
    Person {
    	Name
    	Age
    }
  }
}
`
	var expectedErr [1]string
	expectedErr[0] = `Value cannot be NULL at line: 1 column: 31`

	l := lexer.New(input)
	p := New(l)
	//	p.ClearCache()
	d, errs := p.ParseDocument()
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
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(input))
		t.Errorf(`*************  program.String() wrong.`)
	}
}

func TestXQIllegal1(t *testing.T) {
	var input = `query getZuckProfile(#devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
}
}
`
	expectedErr := `Error: Expected "$" got "#" at [1 : 22]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	for _, v := range errs {
		if v.Error() != expectedErr {
			fmt.Println(v.Error())
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQIllegal2(t *testing.T) {
	var input = `query getZuckProfile(!devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
}
}
`
	expectedErr := `Error: Expected "$" got "!" at [1 : 22]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		if v.Error() != expectedErr {
			fmt.Println(v.Error())
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQIllegal3(t *testing.T) {
	var input = `query getZuckProfile(?devicePicSize: Int = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
}
}
`
	expectedErr := `Error: Expected "$" got "?" at [1 : 22]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	for _, v := range errs {
		if v.Error() != expectedErr {
			fmt.Println(v.Error())
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQIllegal4(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: us*er(id: 4) {
    Xid
    Zname
}
}
`
	expectedErr := `Expected an identifier for a fragment or inlinefragment got ILLEGAL. at [2 : 15]`

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	for _, v := range errs {
		if v.Error() != expectedErr {
			fmt.Println(v.Error())
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}
func TestXQMissingArgName(t *testing.T) {

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

	expectedErr := `Error: Expected an argument name followed by colon got an ": 65.4" at [7 : 10]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQMissingColon(t *testing.T) {

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

	expectedErr := `Error: Expected an argument name followed by colon got an "acd 65.4" at [7 : 10]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		//fmt.Println(v)
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQLeadingDoubleUnderscore(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: __use_r(id: 4) {
    Xid
    Zname
  }
}
`
	expectedErr := `identifer [__use_r] cannot start with two underscores at [2 : 13]`
	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()
	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`program.String() wrong. got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestXQLeadingSingleUnderscore(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: 4) {
    Xid
    Zname
  }
}
`
	expectedDoc := `
 query getZuckProfile( $devicePicSize:Int = 1234) { 
                xyzalias : _use_r ( id : 4) {
                        Xid
                        Zname
                        }
        }`
	l := lexer.New(input)
	p := New(l)
	d, _ := p.ParseDocument()
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQStmtVariableNoDefault(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int) {
  xyzalias: _use_r(id: false) {
    Xid
    Zname
  }
}
`
	expectedDoc := `
 query getZuckProfile( $devicePicSize:Int) { 
                xyzalias : _use_r ( id : false ) {
                        Xid
                        Zname
                        }
        }`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println(e.Error())
	}
	fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQBoolArgValue(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: false) {
    Xid
    Zname
  }
}
`
	expectedDoc := `
 query getZuckProfile( $devicePicSize:Int = 1234) { 
                xyzalias : _use_r ( id : false ) {
                        Xid
                        Zname
                        }
        }`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println(e.Error())
	}
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQMultiArgValue(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: false name: """Ross""" ) {
    Xid
    Zname
  }
}
`
	expectedDoc := `
query getZuckProfile( $devicePicSize:Int = 1234) {
                xyzalias : _use_r ( id : false name : """Ross""" ) {
                        Xid
                        Zname
                        }
        }`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println(e.Error())
	}
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}
func TestXQBooleanVarType(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Boolean = false) {
  xyzalias: _use_r(id: $devicePicSize) {
    Xid
    Zname
  }
}`
	expectedDoc := `query getZuckProfile($devicePicSize: Boolean = false) {
  xyzalias: _use_r(id: false) {
    Xid
    Zname
  }
}`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQVariableReference(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: $devicePicSize) {
    Xid
    Zname
  }
}`
	expectedDoc := `query getZuckProfile( $devicePicSize : Int = 1234) {
  xyzalias: _use_r ( id: 1234) {
    Xid
    Zname
  }
}`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQWrongVariableNameInArgument(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: $ePicSize) {
    Xid
    Zname
  }
}
`
	expectedErr := "Variable, ePicSize not defined  at [2 : 25]"
	l := lexer.New(input)
	p := New(l)
	_, err := p.ParseDocument()

	if p.hasError() && err[0].Error() != expectedErr {
		t.Errorf(`program.String() wrong. got=[%q] expected [%s]`, err[0].Error(), expectedErr)
	}
}

func TestXQNullValue(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: null) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: null) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	//fmt.Println(d.String())
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQList0(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2 34 56.78]) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id:  [1 2 34 56.78]) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQList1(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"]) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"]) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQList2(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2 34 56.78 [6 "xyz" [ "yut" 33 false ] null 78.076 true $devicePicSize] false "abc" ]) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 [6 "xyz" [ "yut" 33 false ] null  78.076 true 1234] false "abc" ]) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQUseOfCommas(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2,, 34, ,56.78 , [6 "xyz" , [ "yut" 33 false ]  null 78.076 true ,$devicePicSize] false,,, "abc" ]) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2,, 34, ,56.78 , [6 "xyz" , [ "yut" 33 false ] null 78.076 true ,1234] false,,, "abc" ]) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQUseOfCommas2(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2,, 34, ,56.78 , [6 "xyz" , [ "yut" 33 false ] null 78.076 true ,$devicePicSize] false,,, "abc" ]) {
    Xid,
    ,Zname,
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2 34 56.78  [6 "xyz"  [ "yut" 33 false ] null 78.076 true 1234] false "abc" ]) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObject1(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"] obj: { id:1 cat :234 food : [ 1 2 3] }) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"] obj: { id:1 cat :234 food : [ 1 2 3] }) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObjecWithVariable(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 1234 ] "abc" "def"] obj: { id:1 cat :1234 food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObjecWithVariableWrongInput(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: OddOne = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 1234 ] "abc" "def"] obj: { id:1 cat :1234 food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObjecWithVariableWrongInput2(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: OddOne = {x: 123 y:123}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize:{x: 123 y:123}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 1234 ] "abc" "def"] obj: { id:1 cat :1234 food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObjecWithVariableWrongInput3(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: OddOne = {x: 123 yy:123.2}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: OddOne ={x: 123 y:123.2}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 {x: 123 y:123.2}] "abc" "def"] obj: { id:1 cat : {x: 123 y:123.2} food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQObjecWithVariableWrongInputType(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Person = {x: 123 yy:123.2}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	var expectedErr [1]string
	expectedErr[0] = `Argument "devicePicSize" type "Person", is not an input type at line: 1 column: 38`

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

func TestXQObjecWithVariableCorrectInput(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: OddOne = {x: 123 y:123.2}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 $devicePicSize] "abc" "def"] obj: { id:1 cat :$devicePicSize food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: OddOne ={x: 123 y:123.2}) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 {x: 123 y:123.2}] "abc" "def"] obj: { id:1 cat : {x: 123 y:123.2} food : [ 1 2 3] } node: "flight" ) {
    Xid
    Zname
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQLotsOfFItems(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
   xyzalias: _use_r(id: [1 2 34 ] a : 1 b : 2 c : 3 d:4 e:5 abc:33 par:"abc" kit:123.12 aa:"a" d:123 p:12 c:"3" f:98 z:12 dd:23 d0:98 e:5 abc:33 par:"abc" kit:123.12 aa:"a" d:123 p:12 c:"3" f:98 z:12 dd:23 d0:98)  {
    Xid
    Zname
    aa
    nn
    dddw
    ew
    sd
    fs
    sf
    ef
    xv
    gd
    df
    fb
    readPoser
    fdg
    cb
    dr
    dd
    ha
    ss
    yj
    rb
    nvn
    fgh
    rt
    ghj
    ll
    kk
    nn
    r3
    r4
    r5
    r6
    r7
    r9
    r8
    r10
      Xid
    Zname
    aa
    nn
    dddw
    ew
    sd
    fs
    sf
    ef
    xv
    gd
    df
    fb
    readPoser
    fdg
    cb
    dr
    dd
    ha
    ss
    yj
    rb
    nvn
    fgh
    rt
    ghj
    ll
    kk
    nn
    r3
    r4
    r5
    r6
    r7
    r9
    r8
    r10
      Xid
    Zname
    aa
    nn
    dddw
    ew
    sd
    fs
    sf
    ef
    xv
    gd
    df
    fb
    readPoser
    fdg
    cb
    dr
    dd
    ha
    ss
    yj
    rb
    nvn
    fgh
    rt
    ghj
    ll
    kk
    nn
    r3
    r4
    r5
    r6
    r7
    r9
    r8
    r10
  }
}
`

	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	fmt.Println("doc ", d.String())
	if compare(d.String(), input) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(input))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQDirective11(t *testing.T) {

	var input = `query getZuckProfile($withFriends: Boolean = true) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"] obj: { id:1 cat :234 food : [ 1 2 3] }) {
    Xid @include(if: $withFriends) @ Size (aa:1 bb:2) @ Pack (filter: true) 
    Zname @include(if: $withFriends) @ Size (aa:1 bb:2) @ Pack (filter: true) {
      aa
      bb
      cc
    }
  }
}
`

	expectedDoc := `query getZuckProfile($withFriends: Boolean = true) {
   xyzalias: _use_r(id: [1 2 34 56.78 "xyz" false [ 1 2 3 4 ] "abc" "def"] obj: { id:1 cat :234 food : [ 1 2 3] }) {
    Xid @include(if: true) @ Size (aa:1 bb:2) @ Pack (filter: true) 
    Zname @include(if: true) @ Size (aa:1 bb:2) @ Pack (filter: true) {
      aa
      bb
      cc
    }
  }
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	fmt.Println(d.String())
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQFragment1(t *testing.T) {

	var input = `query withFragments {
  user(id: 4) {
    friends(first: 10) {
      ...friendFields
    }
    mutualFriends(first: 10) {
      ...friendFields
    }
} }
fragment friendFields on User {
  id
name
  profilePic(size: 50)
}`

	expectedDoc := `
query withFragments {
  user(id: 4) {
    friends(first: 10) {
      ...friendFields
    }
    mutualFriends(first: 10) {
      ...friendFields
    }
} }
fragment friendFields on User {
  id
name
  profilePic(size: 50)
}
`
	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println("[" + d.String() + "]")
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQFragment2(t *testing.T) {

	var input = `query withNestedFragments {
  user(id: 4) {
    friends(first: 10) {
      ...friendFields
}
mutualFriends(first: 10) {
      ...friendFields
} }
}
fragment friendFields on User {
  id
name
  ...standardProfilePic
}
fragment standardProfilePic on User {
  profilePic(size: 50)
}`

	expectedDoc := `query withNestedFragments {
  user(id: 4) {
    friends(first: 10) {
      ...friendFields
}
mutualFriends(first: 10) {
      ...friendFields
} }
}
fragment friendFields on User {
  id
name
  ...standardProfilePic
}
fragment standardProfilePic on User {
  profilePic(size: 50)
}`

	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	fmt.Println("[" + d.String() + "]")
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQInlineFragment2(t *testing.T) {

	var input = `query inlineFragmentTyping {
  profiles(handles: ["zuck", "cocacola"]) {
    handle
    ... on User {
      friends {
        count
} }
    ... on Page {
      likers {
count
} }
} }`
	expectedDoc := `query inlineFragmentTyping {
  profiles(handles: ["zuck", "cocacola"]) {
    handle
    ... on User {
      friends {
        count
} }
    ... on Page {
      likers {
count
} }
} }`

	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println("[" + d.String() + "]")
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}

func TestXQInlineFragWithDirective(t *testing.T) {

	var input = `query inlineFragmentNoType($expandedInfo: Boolean) {
  user(handle: "zuck") {
    id
    name
    ... @include(if: $expandedInfo) {
      firstName
      lastName
      birthday
} }
}`
	expectedDoc := `query inlineFragmentNoType($expandedInfo: Boolean) {
  user(handle: "zuck") {
    id
    name
    ... @include(if: ) {
      firstName
      lastName
      birthday
} }
}`

	l := lexer.New(input)
	p := New(l)
	d, err := p.ParseDocument()
	for _, e := range err {
		fmt.Println("*** ", e.Error())
	}
	//fmt.Println("[" + d.String() + "]")
	if compare(d.String(), expectedDoc) {
		fmt.Println(trimWS(d.String()))
		fmt.Println(trimWS(expectedDoc))
		t.Errorf(`*************  program.String() wrong.`)
	}

}
