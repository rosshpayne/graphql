package parser

import (
	"errors"
	"fmt"
	_ "os"
	"unicode"
	"unicode/utf8"

	"github.com/graphql/ast"
	"github.com/graphql/lexer"
	"github.com/graphql/token"
)

type (
	parseFn func(op string) ast.StatementDef
)

const (
	cErrLimit = 5 // how many parse errors are permitted before processing stops
)

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token

	nameOptional bool // token is nameOptional

	parseFns map[token.TokenType]parseFn
	root     ast.StatementDef
	rootVar  []*ast.VariableDef
	perror   []error // slice of IVs [concrete value,concrete type] - in this case *errors.errorString (assorted concrete types that have a Error() method)
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	p.parseFns = make(map[token.TokenType]parseFn)
	// regiser Parser methods for each statement type
	p.registerFn(token.QUERY, p.parseOperationStmt)
	p.registerFn(token.MUTATION, p.parseOperationStmt)
	//p.registerFn(token.SUBSCRIPTION, p.parseSubscriptionStmt)
	//p.registerFn(token.TYPE, p.parseFragmentStmt)
	//p.registerFn(token.TYPE, p.parseTypeSystemStmt)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	//fmt.Println("New 1 ", p.curToken.Literal)
	p.nextToken()
	//fmt.Println("New 1 ", p.curToken.Literal)

	return p

}

func (p *Parser) ErrLine() string {
	if p.l.Eloc.Line > 0 {
		return fmt.Sprintf(" at [%d : %d]", p.l.Eloc.Line, p.l.Eloc.Col-1)
	}
	return fmt.Sprintf(" at [%d : %d]", p.curToken.Position.Line, p.curToken.Position.Col)
}

func (p *Parser) printTok() {
	fmt.Println("** Current Token: ", p.curToken.Type, p.curToken.Literal, p.curToken.Cat, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)

}
func (p *Parser) hasError() bool {
	if len(p.perror) > 0 {
		return true
	}
	return false
}
func (p *Parser) addErr(s string) error {
	e := errors.New(s + p.ErrLine())
	p.perror = append(p.perror, e)
	return e
}

func (p *Parser) registerFn(tokenType token.TokenType, fn parseFn) {
	p.parseFns[tokenType] = fn
}

func (p *Parser) nextToken(h ...bool) {
	p.curToken = p.peekToken

	p.peekToken = p.l.NextToken() // get another token from lexer:    [,+,(,99,Identifier,keyword etc.
	if len(h) > 0 {
		fmt.Println("** Current Token: ", p.curToken.Type, p.curToken.Literal, p.curToken.Cat, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal, p.peekToken.Cat)
	}
	if p.curToken.Illegal {
		p.addErr(fmt.Sprintf("Illegal %s token, [%s]", p.curToken.Type, p.curToken.Literal))
	}
	// if $variable present then mark the identier as a VALUE
	if p.curToken.Literal == token.DOLLAR {
		p.peekToken.Cat = token.VALUE
	}
}

// ==================== Start =========================

func (p *Parser) ParseDocument() (*ast.Document, []error) {
	program := &ast.Document{}
	program.Statements = []ast.StatementDef{} // slice is initialised with no elements - each element represents an interface value of type ast.StatementDef

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if p.hasError() {
			break
		}
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program, p.perror
}

// ==================== End  =========================

var multiStatement bool

func (p *Parser) parseStatement() ast.StatementDef {
	stmtType := p.curToken.Literal
	if f, ok := p.parseFns[p.curToken.Type]; ok {
		return f(stmtType)
	}
	return nil
}

// parse Name nameOptional
// parse VariableDef nameOptional
// parse Directives  nameOptional
// SelectionSet  nameOptional
//
func (p *Parser) parseOperationStmt(op string) ast.StatementDef {
	// Types: query, mutation, subscription
	p.nextToken() // first token after query, mutation keywords
	stmt := &ast.OperationStmt{Type: op}
	p.root = stmt

	_ = p.parseName(stmt).parseVariables(stmt).parseDirectives(stmt).parseSelectionSet(stmt)

	return stmt

}

func (p *Parser) parseFragmentStmt(op string) ast.StatementDef {

	stmt := &ast.FragmentStmt{}
	//p.root = stmt
	_ = p.parseName(stmt).parseDirectives(stmt).parseSelectionSet(stmt)

	return stmt

}

// type Field struct {
// 	Alias     string
// 	Name      string
// 	Arguments []*Argument
// 	//	directives   []directive
// 	SelectionSet []SelectionSetI // field as object
// }

func (p *Parser) parseField() *ast.Field {
	// Field :
	// Alias Name Arguments Directives SelectionSet
	f := &ast.Field{}

	_ = p.parseAlias(f).parseName(f).parseArguments(f).parseDirectives(f).parseSelectionSet(f)

	return f

}

func (p *Parser) parseFragmentSpread() ast.IsSelectionSetI { return nil }
func (p *Parser) parseInlineFragment() ast.IsSelectionSetI { return nil }

// =========================================================================

func (p *Parser) parseAlias(f *ast.Field) *Parser {
	if p.hasError() {
		return p
	}
	if len(p.perror) > cErrLimit {
		return p
	}
	// check if alias defined
	if p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON {
		if err := p.validateName(p.curToken.Literal); err != nil {
			return p
		} else {
			f.Alias = p.curToken.Literal
		}
		p.nextToken() // COLON
		p.nextToken() // IDENT - prime for next op
	}
	return p
}

func (p *Parser) parseName(f interface{}) *Parser { // type f *ast.Executable,  f=passedInArg converts argument to f
	// check if appropriate thing to do
	if p.hasError() {
		return p
	}

	if p.curToken.Type == token.IDENT { //&& !p.nameOptional {
		if err := p.validateName(p.curToken.Literal); err != nil {
			return p
		} else {
			switch n := f.(type) {
			case *ast.OperationStmt:
				n.Name = p.curToken.Literal
			case *ast.Field:
				n.Name = p.curToken.Literal
			case *ast.FragmentStmt:
				n.Name = p.curToken.Literal
			case *ast.Argument:
				n.Name = p.curToken.Literal
			default:
				p.addErr("parseName: Concrete type not supported")
			}
		}
		p.nameOptional = false
	} else {
		p.addErr(fmt.Sprintf(`Error: Expected name identifer got %s`, p.curToken.Literal))
		return p
	}
	p.nextToken() // prime for next parse op
	return p
}

//
// type Argument struct {
// 	//( name:value )
// 	Name  Name_
// 	Value []InputValue_ // could use string as this value is mapped directly to get function - at this stage we don't care about its type, maybe?
// }
//
// Arguments[Const] :
//		( Argument[?Const]list )
// Argument[Const] :
//		Name : Value [?Const]
func (p *Parser) parseArguments(f *ast.Field) *Parser {

	if p.hasError() || p.curToken.Type != token.LPAREN {
		return p
	}
	//
	parseArgument := func(v *ast.Argument) error {

		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			return p.addErr(fmt.Sprintf(`Error: Expected an argument name followed by colon got an "%s %s"`, p.curToken.Literal, p.peekToken.Literal))
		}
		if err := p.validateName(p.curToken.Literal); err != nil {
			return err
		} else {
			v.Name = p.curToken.Literal
		}
		p.nextToken() // :
		p.nextToken() // argument value
		if !((p.curToken.Cat == token.VALUE && (p.curToken.Type == token.DOLLAR && p.peekToken.Cat == token.VALUE)) ||
			(p.curToken.Cat == token.VALUE && (p.peekToken.Cat == token.NONVALUE || p.peekToken.Type == token.RPAREN)) ||
			(p.curToken.Type == token.LBRACKET || p.curToken.Type == token.LBRACE)) { // [  or {
			return p.addErr(fmt.Sprintf(`Error: Expected an argument Value followed by IDENT or RPAREN got an %s:%s:%s %s:%s:%s`, p.curToken.Cat, p.curToken.Type, p.curToken.Literal, p.peekToken.Cat, p.peekToken.Type, p.peekToken.Literal))
		}

		var err error
		if v.Value, err = p.parseInputValue_(); err != nil {
			return err
		}

		return nil
	}

	p.nextToken() // (
	for ; p.curToken.Type != token.RPAREN; p.nextToken() {
		v := new(ast.Argument)
		if err := parseArgument(v); err != nil {
			break
		}
		if p.curToken.Type == token.EOF {
			p.addErr(fmt.Sprintf(`Error: Expected ) got a "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			break
		}
		f.Arguments = append(f.Arguments, v)
	}
	p.nextToken() // prime for next parse op
	return p
}

func (p *Parser) parseDirectives(f interface{}) *Parser { // f is a iv initialised from concrete types *ast.Field,*OperationStmt,*FragementStmt. It will panic if they don't satisfy DirectiveI
	// so far all types have directives, *ast.OperationalStmt, *ast.FragmentStmt, *ast.Field

	return p
}

// parseSelectionSet - starts with {
// { Selection ... }
// Selection :	Field
//				FragmentSpread
//				InlineFragment
func (p *Parser) parseSelectionSet(f ast.HasSelectionSetI) *Parser {

	if p.hasError() || p.curToken.Type != token.LBRACE {
		return p
	}

	parseSSet := func() ast.IsSelectionSetI {
		var node ast.IsSelectionSetI

		switch p.curToken.Type {
		case token.IDENT:
			node = p.parseField()
		case token.FRAGMENT:
			node = p.parseFragmentSpread()
		case token.EXPAND:
			node = p.parseInlineFragment()
		default:
			switch p.curToken.Type {
			case "Int":
				p.addErr(fmt.Sprintf("Expected an identifier got %s. Probable cause is identifers cannot start with a number", p.curToken.Type))
			default:
				p.addErr(fmt.Sprintf("Expected an identifier, fragment or inlinefragment got %s.", p.curToken.Type))
			}
		}
		return node
	}

	for p.nextToken(); p.curToken.Type != token.RBRACE; {

		node := parseSSet()
		if p.hasError() {
			break
		}
		if s := f.AppendSelectionSet(node); s != nil {
			p.addErr(*s)
			break
		}
	}

	return p
}

// type VariableDef struct {
// 	Name       Name_
// 	Type       InputValueType_ //  string scalar (primitive) type, int, float, string, ID, Boolean, EnumName, ObjectName
// 	DefaultVal InputValue_
// 	Value      InputValue_
// }
// Variable :
//		 $ Name
// VariableDefinitions :
//		( VariableDefinition ... )
// VariableDefinition
//		Variable : Type DefaultValue
// DefaultValue :
//		= Value[Const]
func (p *Parser) parseVariables(st ast.OperationDef) *Parser { // st is an iv initialised from passed in argument which is a *OperationStmt

	if p.hasError() {
		return p
	}

	parseVariable := func(v *ast.VariableDef) bool {

		p.nextToken()
		if p.curToken.Type != token.DOLLAR {
			p.addErr(fmt.Sprintf(`Error: Expected "$" got "%s"`, p.curToken.Literal))
			return true
		}
		p.nextToken() // name identifer

		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			p.addErr(fmt.Sprintf(`Error: Expected an identifer got an "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return true
		}
		if err_ := p.validateName(p.curToken.Literal); err_ != nil {
			return true
		} else {
			v.Name = p.curToken.Literal
		}
		p.nextToken()
		// :
		if !(p.curToken.Type == token.COLON && p.peekToken.Type == token.IDENT) {
			p.addErr(fmt.Sprintf(`Error: Expected : got a "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return true
		}
		p.nextToken() // variable type

		if !(p.curToken.Type == token.IDENT && (p.peekToken.Type == token.IDENT || p.peekToken.Type == token.RPAREN || p.peekToken.Type == token.ASSIGN)) {
			p.addErr(fmt.Sprintf(`Error: Expected a type identifer followed  =, ) or another identifier. Got an "%s" value "%s"`, p.peekToken.Type, p.peekToken.Literal))
			return true
		}
		// use method to assign type value - as it will aggregate some types to a higher type e.g true -> bool. Will also validate the token literal as an appropiate type.
		if err_ := v.AssignType(p.curToken.Literal); err_ != nil {
			p.addErr(err_.Error())
			return true
		}
		p.nextToken() // = , IDENT )

		if p.curToken.Type == token.ASSIGN {
			// an optional default value - stored as an ast.inputvalue. The value  (token.Literal) is a type which is provided by the token.Type.
			// value will go to iv.Value and the type will be "assign" to iv.Type where true -> bool
			// e.g. true will be stored as ast.InputValue_{Value: "true", Type: ast.InputValueType{"bool"} ]
			// e.g. 123 will be stored as ast.InputValue_{Value: "123", Type: ast.InputValueType{"Int"}  ]
			// e.g. 23.4 will be stored as ast.InputValue_{Value: "23.4", Type: ast.InputValueType{"Float"}  ]
			// e.g. {a:1} will be stored as ast.InputValue_{Value: "ast.Argument{Name: "a", ast.InputValue_{Value: 1,ast.InputValueType{"Int"} }}", Type: ast.InputValueType{"Object"}  ]
			p.nextToken()
			if vv, err_ := p.parseInputValue_(); err_ != nil {
				return true
			} else {
				v.DefaultVal = vv
			}
			//	p.nextToken()
		}
		return false
	}

	if len(p.perror) > cErrLimit {
		return p
	}

	switch stmt := st.(type) {
	case *ast.OperationStmt:
		if p.curToken.Type == token.LPAREN {
			for ; p.curToken.Type != token.RPAREN; p.nextToken() {

				v := ast.VariableDef{}
				if err := parseVariable(&v); err {
					return p
				} else {
					stmt.Variable = append(stmt.Variable, &v)
				}
			}
			p.rootVar = stmt.Variable
			p.nextToken() // prime for next op
		}
	}

	return p
}

// type InputValue_ struct {
// 	Value string //  Token.Literal, now need to consider non-literal types like []value, hence need to use an interface
// 	Type  string //  Token.Type - scalar types ony supported for the moment.
// }

//  parseInputValue_ expects an InputValue_ literal (true,false, 234, 23.22, "abc" or $variable in the next token.  The value is a type bool,int,flaot,string..
//  if it is a variable then the variable value (which is an InputValue_ type) will be sourced
//  TODO: currently called from parseArgument only. If this continues to be the case then add this func as anonymous func to it.
func (p *Parser) parseInputValue_() (ast.InputValue_, error) {
	if p.curToken.Cat != token.VALUE {
		return ast.InputValue_{}, p.addErr(fmt.Sprintf("Value expected got %s of %s", p.curToken.Type, p.curToken.Literal))
	}
	//p.printTok()
	switch p.curToken.Type {

	case token.DOLLAR:
		// variable supplied - need to fetch value
		p.nextToken() // IDENT variable name
		// change category of token to VALUE as previous token was $ - otherwise this step would not be executed.
		p.curToken.Cat = token.VALUE
		if p.curToken.Type == token.IDENT {
			// get variable value....
			if val, ok := p.getVarValue(p.curToken.Literal); !ok {
				return ast.InputValue_{}, p.addErr(fmt.Sprintf("Variable, %s not defined ", p.curToken.Literal))
			} else {
				return val, nil
			}
		} else {
			return ast.InputValue_{}, p.addErr(fmt.Sprintf("Expected Variable Name Identifer got %s", p.curToken.Type))
		}

	case token.LBRACKET:
		// [ value value value .. ]
		p.nextToken() // first value , ]
		if !(p.curToken.Cat == token.VALUE && (p.peekToken.Cat == token.VALUE || p.curToken.Type == token.RBRACKET)) {
			return ast.InputValue_{}, p.addErr(fmt.Sprintf("Expect an Input Value followed by another Input Value or a ], got %s %s ", p.curToken.Literal, p.peekToken.Literal))
		}
		// edge case []
		if p.peekToken.Type == token.RBRACKET {
			p.nextToken() // ]
			var null ast.Null_ = true
			iv := ast.InputValue_{Value: null}
			err := iv.AssignType(token.NULL)
			if err != nil {
				return ast.InputValue_{}, p.addErr(err.Error())
			}
			return iv, err
		}
		// process list of values
		var vallist ast.List_
		for ; p.curToken.Type != token.RBRACKET; p.nextToken() {
			if v, err := p.parseInputValue_(); err != nil {
				return ast.InputValue_{}, err
			} else {
				vallist = append(vallist, v)
			}
		}
		// completed processing values, return List type
		iv := ast.InputValue_{Value: vallist}
		err := iv.AssignType("List")
		if err != nil {
			return ast.InputValue_{}, p.addErr(err.Error())
		}

		return iv, err

	/*	case token.LBRACE: // Object value
		case token.IDENT: // Name_
	*/
	case token.NULL:
		var null ast.Null_ = true
		iv := ast.InputValue_{Value: null}
		err := iv.AssignType(token.NULL)
		if err != nil {
			return ast.InputValue_{}, p.addErr(err.Error())
		}
		return iv, nil

	default: // name: value , scalar value specified - token.Type == IDENT, token.Literal == variable-value-to-save

		sc := ast.Scalar_(p.curToken.Literal)
		iv := ast.InputValue_{Value: &sc}
		err := iv.AssignType(string(p.curToken.Type))
		if err != nil {
			return ast.InputValue_{}, p.addErr(err.Error())
		}
		return iv, nil
	}

}

// rootvar: &ast.VariableDef{
// Name:"devicePicSize",
// inputValueType_:"Int",
// DefaultVal:ast.InputValue_{Value:(*ast.Scalar_)(0xc420050440), inputValueType_:"Int"},
// Value:ast.InputValue_{Value:ast.ValueI(nil), inputValueType_:""}
// }
func (p *Parser) getVarValue(name string) (ast.InputValue_, bool) {
	for _, v := range p.rootVar {
		//fmt.Printf(" rootvar: %#v . %s \n", v, v.DefaultVal.String())
		if v.Name == name {
			if v.Value.Value != nil {
				return v.Value, true
			} else {
				return v.DefaultVal, true
			}
		}
	}
	return ast.InputValue_{}, false
}

var errNameChar string = "Invalid character in identifer "
var errNameBegin string = "identifer [%s] cannot start with two underscores"

func (p *Parser) validateName(name string) error {
	// /[_A-Za-z][_0-9A-Za-z]*/
	if len(name) == 0 {
		return p.addErr("error in p.validateName_() - no argument supplied")
	}

	var err error = nil

	ch, _ := utf8.DecodeRuneInString(name[:1])
	if unicode.IsDigit(ch) {
		err = p.addErr("identifier cannot start with a number")
	}

	for i, v := range name {
		switch i {
		case 0:
			if !(v == '_' || (v >= 'A' || v <= 'Z') || (v >= 'a' && v <= 'z')) {
				err = p.addErr(errNameChar)
			}
		default:
			if !((v >= '0' && v <= '9') || (v >= 'A' || v <= 'Z') || (v >= 'a' && v <= 'z') || v == '_') {
				err = p.addErr(errNameChar)
			}
		}
		if err != nil {
			break
		}
	}
	if len(name) > 1 && name[:2] == "__" {
		return p.addErr(fmt.Sprintf(errNameBegin, name))
	} else if err != nil {
		return err
	}
	return nil
}

// Scalar types: Int, Float, String, Boolean, ID,
// Enum: <EnumName>
// List: [ <type> ]
// InputObjectValues
//
//	cpos:=p.l.Cpos()
// func (p *Parser) parseInputValueType_() (ast.InputValueType_, error) {

// 	p.nextToken()
// 	if p.curToken.Type != token.IDENT {
// 		return fmt.Sprintf("Invalid token, expect identifer got %s", p.curToken.Type))
// 	}
// 	switch t := s.curToken.Literal; t {
// 	case "Int", "Float", "Boolean", "ID":
// 		return ast.InputValueType_{Name: t, Type: "Scalar"}, nil
// 	default:
// 		// check literal value in type system
// 		return ast.InputValueType_{Name: s[cpos:p.l.Cpos], Type: "other.."}, nil
// 	}
// 	return ast.InputValueType_{}, fmt.Sprintf("Variable type, %s not found at", s.curToken.Literal, s.p.ErrLine()))
// }
