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

func TestMissingRPAREN(t *testing.T) {
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

	expectedErr := `Error: Expected an argument name followed by colon got an "} ff" at [10 : 35]`

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

func TestMisplacedVariable(t *testing.T) {
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
	expectedErr := `Error: Expected an argument name followed by colon got an "$ devicePicSize" at [7 : 9]`

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestBadVariableType(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: It = 1234) {
  xyzalias: user(id: 4) {
    Xid
    Zname
  }
}
`
	expectedErr := `Input value type not supported [It] at [1 : 38]`

	l := lexer.New(input)
	p := New(l)
	_, errs := p.ParseDocument()

	for _, v := range errs {
		if v.Error() != expectedErr {
			t.Errorf(`Wrong Error got=[%q] expected [%s]`, v.Error(), expectedErr)
		}
	}
}

func TestIllegal1(t *testing.T) {
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

func TestIllegal2(t *testing.T) {
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

func TestIllegal3(t *testing.T) {
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

func TestIllegal4(t *testing.T) {
	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: us*er(id: 4) {
    Xid
    Zname
}
}
`
	expectedErr := `Expected an identifier, fragment or inlinefragment got ILLEGAL. at [2 : 15]`
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
func TestMissingArgName(t *testing.T) {

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

func TestMissingColon(t *testing.T) {

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

func TestLeadingDoubleUnderscore(t *testing.T) {

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

func TestLeadingSingleUnderscore(t *testing.T) {

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

func TestBoolArgValue(t *testing.T) {

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

func TestMultiArgValue(t *testing.T) {

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
func TestBooleanVarType(t *testing.T) {

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

func TestVariableReference(t *testing.T) {

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

func TestWrongVariableNameInArgument(t *testing.T) {

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

func TestNullValue(t *testing.T) {

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

func TestList(t *testing.T) {

	var input = `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id: [1 2 34 56.78 [6 "xyz" [ "yut" 33 false ] null 78.076 true $devicePicSize] false "abc" ]) {
    Xid
    Zname
  }
}
`

	expectedDoc := `query getZuckProfile($devicePicSize: Int = 1234) {
  xyzalias: _use_r(id:  [1 2 34 56.78 [6 "xyz" [ "yut" 33 false ] null 78.076 true 1234 ] false "abc" ]) {
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
