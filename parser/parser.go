package parser

import (
	"errors"
	"fmt"
	_ "os"
	"strconv"
	"strings"

	sdl "github.com/graph-sdl/ast"
	lex "github.com/graph-sdl/lexer"
	pse "github.com/graph-sdl/parser"
	"github.com/graphql/ast"
	"github.com/graphql/lexer"
	"github.com/graphql/resolver"
	"github.com/graphql/token"
)

const (
	cErrLimit  = 8 // how many parse errors are permitted before processing stops
	Executable = 'E'
	TypeSystem = 'T'
)

type Argument struct {
	Name  string
	Value string
}

type (
	parseFn func(op string) ast.StatementDef

	Parser struct {
		l *lexer.Lexer

		extend bool

		abort bool
		// schema rootAST

		curToken  token.Token
		peekToken token.Token

		responseMap map[string]*sdl.InputValueProvider //struct{}
		respOrder   []string                           // slice of field paths in order executed.
		//response  []*ast.ResponseValue // conerts response from reolver  to internal sdl.ObjectVal

		root    ast.StatementDef
		rootVar []*ast.VariableDef

		resolver resolver.Resolvers

		parseFns map[token.TokenType]parseFn
		perror   []error
	}
)

var (
	//	enumRepo      ast.EnumRepo_
	typeNotExists map[sdl.NameValue_]bool
)

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	p.parseFns = make(map[token.TokenType]parseFn)
	// regiser Parser methods for each statement type
	p.registerFn(token.QUERY, p.parseOperationStmt)
	p.registerFn(token.MUTATION, p.parseOperationStmt)
	//p.registerFn(token.SUBSCRIPTION, p.parseSubscriptionStmt)
	p.registerFn(token.FRAGMENT, p.parseFragmentStmt)
	// Read two tokens, to initialise curToken and peekToken
	p.nextToken()
	p.nextToken()
	//
	// remove cacheClar before releasing..
	//
	//ast.CacheClear()
	return p
}

var FragmentStmts map[sdl.Name_]*ast.FragmentStmt

// repository of all types defined in the graph

func init() {
	//	enumRepo = make(ast.EnumRepo_)
	typeNotExists = make(map[sdl.NameValue_]bool)

}

func (p *Parser) Loc() *sdl.Loc_ {
	loc := p.curToken.Loc
	return &sdl.Loc_{loc.Line, loc.Col}
}

// func (p *Parser) ClearCache() {
// 	ast.CacheClear()
// }
func (p *Parser) printToken(s ...string) {
	if len(s) > 0 {
		fmt.Printf("** Current Token: [%s] %s %s %s %v %s %s [%s]\n", s[0], p.curToken.Type, p.curToken.Literal, p.curToken.Cat, p.curToken.IsScalarType, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
	} else {
		fmt.Println("** Current Token: ", p.curToken.Type, p.curToken.Literal, p.curToken.Cat, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
	}
}
func (p *Parser) hasError() bool {
	if len(p.perror) > 17 || p.abort {
		return true
	}
	return false
}

// addErr appends to error slice held in parser.
func (p *Parser) addErr(s string) error {
	if strings.Index(s, " at line: ") == -1 {
		s += fmt.Sprintf(" at line: %d, column: %d", p.curToken.Loc.Line, p.curToken.Loc.Col)
	}
	e := errors.New(s)
	p.perror = append(p.perror, e)
	return e
}

func (p *Parser) registerFn(tokenType token.TokenType, fn parseFn) {
	p.parseFns[tokenType] = fn
}

func (p *Parser) nextToken(s ...string) {
	p.curToken = p.peekToken

	p.peekToken = p.l.NextToken() // get another token from lexer:    [,+,(,99,Identifier,keyword etc.
	if len(s) > 0 {
		fmt.Printf("** Current Token: [%s] %s %s %s %s %s %s\n", s[0], p.curToken.Type, p.curToken.Literal, p.curToken.Cat, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
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
	program.Statements = []ast.StatementDef{} // contains operational stmts (query, mutation, subscriptions) and fragment stmts
	//
	// preparation - get Schema ast from db
	//
	var (
		schemaAST, rootAST sdl.GQLTypeProvider
		schema             *sdl.Schema_
		allErrors          []error
	)

	if schemaAST = p.fetchAST(sdl.Name_{Name: sdl.NameValue_("schema")}); schemaAST == nil {
		p.addErr("Abort. There is no schema defined")
		return nil, p.perror
	}
	schema = schemaAST.(*sdl.Schema_)
	//
	// Phase 1: parse all statements (query, fragment) in the document and add to cache if statement has no errors
	//
	for p.curToken.Type != token.EOF {
		stmtAST := p.parseStatement()
		if p.hasError() {
			break
		}
		switch qry := stmtAST.(type) {
		case *ast.OperationStmt:
			switch qry.Type {
			case "query":
				// get query rootAST
				if rootAST = p.fetchAST(schema.Query); rootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Query))
					return nil, p.perror
				}
				fmt.Printf("rootAST: %T\n", rootAST)
			case "mutation":
				// get mutation rootAST
				if rootAST = p.fetchAST(schema.Mutation); rootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Mutation))
					return nil, p.perror
				}
			case "subscription":
				// get subscription rootAST
				if rootAST = p.fetchAST(schema.Subscription); rootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Subscription))
					return nil, p.perror
				}
			}
		}
		if stmtAST != nil {
			program.Statements = append(program.Statements, stmtAST)
			if len(p.perror) == 0 {
				ast.Add2Cache(stmtAST.StmtName(), stmtAST)
			}
			allErrors = append(allErrors, p.perror...)
			p.perror = nil
		}
	}
	//
	// phase 2: validate  - resolve ALL types. Once complete all type's AST will reside in the cache
	//                    and  *Type.AST assigned where applicable
	//
	for _, v := range program.Statements {
		// generic checks
		p.resolveAllTypes(v)
		if p.hasError() {
			return nil, p.perror
		}
		// check all fields belong to their respective root type
		// check for duplicate fields
		p.checkFields(rootAST, v)

		// type specific checks

		switch stmt := v.(type) {
		case *ast.OperationStmt:
			//	p.expandFragmentSpread(v)
			// in arguments only
			//stmt.CheckIsOutputType(&p.perror) // check all selectionSet datatypes are output types
			stmt.CheckIsInputType(&p.perror)
			stmt.CheckInputValueType(&p.perror)

		case *ast.FragmentStmt:
			//p.resolveAllTypes(v) // in arguments only
			//v.CheckIsOutputType(&p.perror) // check all selectionSet datatypes are output types
			stmt.CheckIsInputType(&p.perror)
			//	x.CheckInputValueType(&p.perror)
		}
	}
	allErrors = append(allErrors, p.perror...)

	for _, v := range program.Statements {
		p.executeStmt(rootAST, v)
		allErrors = append(allErrors, p.perror...)
	}

	return program, allErrors
}

// ==================== End  =========================

var multiStatement bool //TODO remove if not used.
var opt bool = true     // is optional

func (p *Parser) parseStatement() ast.StatementDef {
	stmtType := p.curToken.Literal
	if f, ok := p.parseFns[p.curToken.Type]; ok {
		return f(stmtType)
	}
	return nil
}

// ===================  resolveAllTypes  ==========================
// resolveAllTypes in the couple of cases where types are explicitly defined in operation statements (query,mutation,subscription)
// It is also in the selectionset that objects are sourced and resolved.
// Once resolved we have the AST of all types referenced to in the statement saved in the cache
// A later validation will resolve all scalar types.
func (p *Parser) resolveAllTypes(v ast.StatementDef) {
	//returns slice of unresolved types from the statement passed in
	unresolved := make(sdl.UnresolvedMap)
	v.CheckUnresolvedTypes(unresolved)

	//  unresolved should only contain non-scalar types known upto that point.
	for tyName, ty := range unresolved { // unresolvedMap: [name]*Type

		ast_ := p.fetchAST(tyName)
		// type ENUM values will have nil *Type
		if ast_ != nil {
			if ty != nil {
				ty.AST = ast_
				// if not scalar then check for unresolved types in nested type
				if !ty.IsScalar() {
					p.resolveNestedType(ast_)
				}
			}

		} else {
			// nil ast_ means not found in db
			if ty != nil {
				p.addErr(fmt.Sprintf(`Type "%s" does not exist %s`, ty.Name, ty.AtPosition()))
			} else {
				p.addErr(fmt.Sprintf(`Type "%s" does not exist %s`, tyName, tyName.AtPosition()))
			}
		}
	}

}

func (p *Parser) resolveNestedType(v sdl.GQLTypeProvider) {
	//returns slice of unresolved types from the statement passed in
	unresolved := make(sdl.UnresolvedMap)
	v.CheckUnresolvedTypes(unresolved)

	//  unresolved should only contain non-scalar types known upto that point.
	for tyName, ty := range unresolved { // unresolvedMap: [name]*Type
		ast_ := p.fetchAST(tyName)
		// type ENUM values will have nil *Type
		if ast_ != nil {
			if ty != nil {
				ty.AST = ast_
				// if not scalar then check for unresolved types in nested type
				if !ty.IsScalar() {
					p.resolveNestedType(ast_)
				}
			}

		} else {
			// nil ast_ means not found in db
			if ty != nil {
				p.addErr(fmt.Sprintf(`Type "%s" does not exist %s`, ty.Name, ty.AtPosition()))
			} else {
				p.addErr(fmt.Sprintf(`Type "%s" does not exist %s`, tyName, tyName.AtPosition()))
			}
			p.abort = true
		}
	}
}

// ====================  fetchAST  =============================
// fetchAST should only be used after all statements have been passed
//  As each statement is parsed its types are added to the cache
//  During validation phase each type is checked for existence using this func.
//  if not in cache then looks at DB for types that have been predefined.
func (p *Parser) fetchAST(name sdl.Name_) sdl.GQLTypeProvider {
	var (
		ast_ sdl.GQLTypeProvider
		ok   bool
	)
	name_ := name.Name
	if ast_, ok = sdl.CacheFetch(name_); !ok {

		if !typeNotExists[name_] {

			if typeDef, err := ast.DBFetch(name_); err != nil {

				p.addErr(err.Error())
				p.abort = true
				return nil
			} else {

				if len(typeDef) == 0 { // no type found in DB
					// mark type as being nonexistent
					typeNotExists[name_] = true
					return nil

				} else {

					// generate type AST (not statement AST)
					l := lex.New(typeDef)
					p2 := pse.New(l)
					ast_ = p2.ParseStatement()
					if len(p2.Getperror()) > 0 {
						// error in parsing stmt from db - this should not happen as only valid stmts are saved.
						p.perror = append(p.perror, p2.Getperror()...)
					}
					// add to sdl cache
					sdl.Add2Cache(name_, ast_)
					// resolve types in this ast
					err := p2.ResolveAllTypes(ast_)
					p.perror = append(p.perror, err...)

				}
			}
		} else {
			return nil
		}
	}
	return ast_
}

// =====================================================================

// ================== checkFields ======================================

func (p *Parser) checkFields(root sdl.GQLTypeProvider, stmt_ ast.StatementDef) {

	var (
		stmt *ast.OperationStmt
		ok   bool
	)
	if stmt, ok = stmt_.(*ast.OperationStmt); !ok {
		p.addErr("checkFields only handlest Operational Statements, given something else")
		return
	}
	// only for operational Query
	if stmt.Type != "query" {
		return
	}
	p.responseMap = make(map[string]*sdl.InputValueProvider) // map[sdl.NameValue_]map[sdl.NameValue_]sdl.GQLTypeProvider

	// fields in rootAST (defined in Query or Schema)
	// before := len(p.perror)
	p.checkFields_(root, stmt.SelectionSet, string(root.TypeName()))
	// if len(p.perror) == before {
	// 	p.ResponseMap = make(ResponseMapT)
	// 	for k := range p.responseMap {
	// 		p.ReponseMap[k] = ast.ResponseValue{}
	// 	}
	// }
	// p.responseMap = nil
}

func (p *Parser) checkFields_(root sdl.GQLTypeProvider, set []ast.SelectionSetI, pathRoot string) {
	// ty_ (object):  type Query { allPersons(last : Int ) : [Person!]! }	<== root
	//
	// 	stmt:	query XYZ {
	//      allPersons(last: 2) {											<== set
	//          name
	//          age
	//      }
	// }
	var (
		rootObj *sdl.Object_
	)
	if root == nil {
		p.addErr("In checkFields_, passed in a root of nil")
		return
	}

	rootObj = root.(*sdl.Object_) // 	type Query { allPersons(last: Int): [Person!]	}

	for _, qryFld := range set {

		switch qry := qryFld.(type) { // allPersons(last:3)

		case *ast.Field:
			var (
				newRoot sdl.GQLTypeProvider
				found   bool
				rootFld *sdl.Field_
			)
			//
			// Confirm argument value type against type definition
			//
			for _, rootFld = range rootObj.FieldSet { // root object nested fields

				if qry.Name.Equals(rootFld.Name_) { // allPersons

					found = true
					//
					// validate argument inputs
					//
					for _, argVal := range qry.Arguments {
						var argfound bool
						for _, argDef := range rootFld.ArgumentDefs {
							if argVal.Name_.Equals(argDef.Name_) {
								argfound = true

								argVal.Value.CheckInputValueType(argDef.Type, argVal.Name_, &p.perror)
								break
							}
						}
						if !argfound {
							p.addErr(fmt.Sprintf(`Field argument "%s" is not defined in type "%s", %s`, argVal.Name_, rootObj.Name_, argVal.Name_.AtPosition()))
							p.abort = true
						}
					}
					// check if assoicated type is an object
					if _, ok := rootFld.Type.AST.(*sdl.Object_); ok {
						newRoot = rootFld.Type.AST // Person: AST is populated thanks to ResolveType(), so no need to find in cache.
					}
				}
			}
			if !found {
				p.addErr(fmt.Sprintf(`Field "%s" is not in object, %s %s`, qry.Name, rootObj.TypeName(), qry.Name.AtPosition()))

			} else {

				if newRoot != nil && len(qry.SelectionSet) != 0 {
					// new root object
					fieldPath := pathRoot + "/" + string(newRoot.TypeName())
					//	qryFldMap[fieldPath] = rootFld
					p.respOrder = append(p.respOrder, fieldPath)
					p.checkFields_(newRoot, qry.SelectionSet, fieldPath)

				} else {
					//
					// scalar field - append to response map
					//
					fieldPath := pathRoot + "/" + qry.Name.String()
					//	qryFldMap[fieldPath] = rootFld
					if _, ok := p.responseMap[fieldPath]; ok {
						p.addErr(fmt.Sprintf(`Field "%s.%s" has already been specified %s`, rootObj.TypeName(), qry.Name.String(), qry.Name.AtPosition()))
					} else {
						p.responseMap[fieldPath] = nil
						p.respOrder = append(p.respOrder, fieldPath)
					}
				}
			}

		case *ast.InlineFragment:

			rootFrag := root
			rootPath := pathRoot

			if !qry.TypeCond.Exists() && len(qry.Directives) == 0 {

				p.checkFields_(rootFrag, qry.SelectionSet, rootPath)

			} else {

				if qry.TypeCond.Exists() {
					// reset root object
					var ok bool
					// type condition should exist in cache - populated during resolve types.
					if rootFrag, ok = sdl.CacheFetch(qry.TypeCond.Name); !ok {
						p.addErr(fmt.Sprintf(`Fragment type condition "%s" does not eqryist as fragment %s`, qry.TypeCond.Name, qry.TypeCond.AtPosition()))
						return
					}
					rootPath += "/" + string(rootFrag.TypeName())
				}

				if len(qry.Directives) == 0 {

					p.checkFields_(rootFrag, qry.SelectionSet, rootPath)

				} else {
					//
					// process directives
					//
					for _, v := range qry.Directives {
						//... @include(if: $expandedInfo) {
						if v.Name_.String() == "@include" {
							for _, arg := range v.Arguments {
								if arg.Name.String() != "if" {
									p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
									return
								}
								// parse wil have populated argument value with variable value.
								argv := arg.Value.InputValueProvider.(sdl.Bool_)
								if argv == true {
									p.checkFields_(rootFrag, qry.SelectionSet, rootPath)
								}
							}
						} else {
							fmt.Println("no @include directive")
							p.checkFields_(rootFrag, qry.SelectionSet, rootPath)
						}
					}
				}
			}
		}
	}
}

// ================== executeStmt ======================================

type Person struct {
	id    int
	name  string
	age   int
	other []string
	posts []int
}

func (p *Person) String() string {
	var s strings.Builder
	s.WriteString("{\n")
	s.WriteString(` name : "`)
	s.WriteString(p.name)
	s.WriteString(`"`)
	s.WriteString("\n age: ")
	s.WriteString(strconv.Itoa(p.age))
	s.WriteString("\n")
	s.WriteString("other : [")
	for _, v := range p.other {
		s.WriteString(`"`)
		s.WriteString(v)
		s.WriteString(`" `)
	}
	s.WriteString(" ]\n")
	s.WriteString(" posts : [")
	for _, v := range p.posts {
		//s.WriteString(strconv.Itoa(v) + " ")
		s.WriteString(posts[v-1].String())
	}
	s.WriteString(" ]\n")
	s.WriteString("}\n")
	return s.String()
}

func (p *Person) ShortString() string {
	var s strings.Builder
	s.WriteString("{")
	s.WriteString(`name : "`)
	s.WriteString(p.name)
	s.WriteString(`"`)
	s.WriteString(" ")
	s.WriteString(`age : `)
	s.WriteString(strconv.Itoa(p.age))
	s.WriteString(` }`)
	return s.String()
}

type Post struct {
	id     int
	title  string
	author int
}

func (p *Post) String() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(`	{ title : "`)
	s.WriteString(p.title)
	s.WriteString(`"	 author : [`)
	s.WriteString(persons[p.author-100].ShortString())
	s.WriteString("]	}")
	return s.String()
}

var persons = []*Person{
	&Person{100, "Jack Smith", 53, []string{"abc", "def", "hij"}, []int{1, 2, 3}},
	&Person{101, "Jenny Hawk", 25, []string{"aaaaabc", "def", "hij"}, []int{3, 4}},
	&Person{102, "Sabastian Jackson", 44, []string{"123", "def", "hij"}, nil},
	&Person{103, "Phillip Coats", 54, []string{"xyz", "def", "hij"}, nil},
	&Person{104, "Kathlyn Host", 33, []string{"abasdc", "def", "hij"}, []int{5}},
}
var posts = []*Post{
	&Post{1, "GraphQL for Begineers", 100}, &Post{2, "Holidays in Tuscany", 101}, &Post{3, "Sweet", 102}, &Post{4, "Programming in GO", 102}, &Post{5, "Skate Boarding Blog", 101},
	&Post{6, "GraphQL for Architects", 100},
}

func (p *Parser) executeStmt(root sdl.GQLTypeProvider, stmt_ ast.StatementDef) {

	var (
		stmt *ast.OperationStmt
		ok   bool
		out  strings.Builder
	)
	var (
		testResolverAll = func(resp sdl.InputValueProvider, args sdl.ObjectVals) string {

			var s strings.Builder
			var last_ int = 2
			var err error
			fmt.Println(args.String())
			if len(args) > 0 {
				if args[0].Name.EqualString("last") {
					last := args[0].Value.InputValueProvider.(sdl.Int_)
					fmt.Println("Limited to: ", string(last))
					if last_, err = strconv.Atoi(string(last)); err != nil {
						fmt.Println(err)
					}
				}
			}
			s.WriteString(" [")
			for i, v := range persons {
				if i > last_-1 {
					break
				}
				s.WriteString(v.String())
			}
			s.WriteString("]")
			return s.String()
		}

		// testResolver2 = func(resp sdl.InputValueProvider, args sdl.ObjectVals) string {
		// 	var name string
		// 	fmt.Println(resp.String())
		// 	switch x := resp.(type) {
		// 	case sdl.ObjectVals:
		// 		for _, v := range x {
		// 			if v.Name_.EqualString("name") {
		// 				name = v.Value.String()
		// 			}
		// 		}
		// 		for _, v := range persons {
		// 			if v.name == name {
		// 				return "{ name : " + name + "age : " + strconv.Itoa(v.age) + " }"
		// 			}
		// 		}
		// 		return "abc"
		// 	}
		// 	return "abc"
		// }
	)
	if stmt, ok = stmt_.(*ast.OperationStmt); !ok {
		return
	}
	// only for operational Query
	if stmt.Type != "query" {
		return
	}
	//
	// register resolvers - this would normally be populated by the client and resolverMap passed to server
	//
	p.resolver = resolver.New()
	if err := p.resolver.Register("Query/allPersons", testResolverAll); err != nil {
		p.addErr(err.Error())
		return
	}
	// if err := p.resolver.Register("Query/allPersons/posts/author", testResolver2); err != nil {
	// 	p.addErr(err.Error())
	// 	return
	// }
	fmt.Println("Resolver paths: ")
	fmt.Println(p.resolver.String())
	fmt.Println("================================ executeStmt ================================")
	out.WriteString("response: {")

	p.executeStmt_(root, stmt.SelectionSet, string(root.TypeName()), nil, &out)

	fmt.Println("==== output ====== ")
	fmt.Println(out.String())
}

var noNewLine bool = true

func writeout(path string, s *strings.Builder, str string, noNewLine ...bool) {
	tabs := strings.Count(path, "/")
	if len(noNewLine) == 0 {
		s.WriteString("\n")
		for i := tabs - 1; i > 0; i-- {
			s.WriteString("\t")
		}
	}
	s.WriteString(" ")
	s.WriteString(str)
}

//type responseProvider sdl.InputValueProvider

func (p *Parser) executeStmt_(root sdl.GQLTypeProvider, set []ast.SelectionSetI, pathRoot string, responseItems sdl.InputValueProvider, out *strings.Builder) { //type ObjectVals []*ArgumentT - serialized object
	// ty_ (object):  type Query { allPersons(last : Int ) : [Person!]! }	<== root (type information)
	//
	// 	stmt:	`query XYZ {												<== query statement (what to display)
	//      allPersons(last: 2 ) {											<== resolver here - generates data below
	//          name														<== all other resolvers run default e.g. dispaly associated data
	//          age
	//          posts {
	//          	title
	//          	author {
	//          		name
	//					age
	//          	}
	//          }
	//      }
	// }
	// {response : {														<== response data (source data) from resolver
	// 	[{
	//  name : "Jack Smith"
	//  age: 53
	//  posts : [
	//         { title : "GraphQL for Begineers"        author : {name : "Jack Smith" }        }
	//         { title : "Holidays in Tuscany"  author : {name : "Jenny Hawk" }        }
	//         { title : "Sweet"        author : {name : "Sabastian Jackson" } } ]
	// }
	// {
	//  name : "Jenny Hawk"
	//  age: 25
	//  posts : [
	//         { title : "Sweet"        author : {name : "Sabastian Jackson" } }
	//         { title : "Programming in GO"    author : {name : "Sabastian Jackson" } } ]
	// }
	// ]
	//}}
	//
	// response: {															<==== output ====== CheckInputValueType( [Person!]!,....
	//  allPersons :  [
	//  {
	//  name :  "Jack Smith"
	//  age :  53
	//  posts :  [
	//          {
	//          title :  "GraphQL for Begineers"
	//          author :  {
	//                  name :  "Jack Smith"
	//                  age :  53 }
	//          }
	//          {
	//          title :  "Holidays in Tuscany"
	//          author :  {
	//                  name :  "Jenny Hawk"
	//                  age :  25 }
	//          }
	//          {
	//          title :  "Sweet"
	//          author :  {
	//                  name :  "Sabastian Jackson"
	//                  age :  44 }
	//          } ]
	//  }
	//  {
	//  name :  "Jenny Hawk"
	//  age :  25
	//  posts :  [
	//          {
	//          title :  "Sweet"
	//          author :  {
	//                  name :  "Sabastian Jackson"
	//                  age :  44 }
	//          }
	//          {
	//          title :  "Programming in GO"
	//          author :  {
	//                  name :  "Sabastian Jackson"
	//                  age :  44 }
	//          } ]
	//  } ]

	var (
		rootObj *sdl.Object_
	)

	if p.hasError() {
		return
	}
	if root == nil {
		p.addErr("In checkFields_, passed in a root of nil")
		return
	}

	rootObj = root.(*sdl.Object_)

	// scan selection set from query statement's AST
	for _, qryFld := range set {

		switch qry := qryFld.(type) {

		case *ast.Field:
			var (
				newRoot   sdl.GQLTypeProvider
				fieldPath string
				response  string
				rootFld   *sdl.Field_
			)

			// rootFld = qryFldMap[pathRoot] // could access via map but thinking about memory requirements for maps, when simple scan-loop swaps CPU for scan instead of memory
			// match field name to root object's AST to determine field's type
			for _, rootFld = range rootObj.FieldSet { // root object nested fields

				fmt.Println("looking for: ", rootFld.Name_, qry.Name)

				if !qry.Name.Equals(rootFld.Name_) {
					continue
				}
				fmt.Println("found : ", rootFld.Name_, qry.Name)

				// if responseItems == nil {
				// 	qry.Resolver = testResolverAll
				// } else {
				// 	qry.Resolver = nil
				// }

				// check if assoicated type is an object
				switch rootFld.Type.AST.(type) {

				case *sdl.Object_:
					//
					// object field
					//
					newRoot = rootFld.Type.AST // qryFld's matching  AST
					fmt.Println("ALias: ", qry.Alias.String())
					// if qry.Alias.Exists() {
					// 	fieldPath = pathRoot + "/" + qry.Alias.String()
					// } else {

					// }
					fieldPath = pathRoot + "/" + rootFld.Name_.String()
					fmt.Println("*************************** pathRoot, fieldPath: ", rootFld.Name_.String(), fieldPath)

					qry.Resolver = p.resolver.GetFunc(fieldPath)

					if qry.Resolver == nil {

						if responseItems == nil {
							p.addErr(fmt.Sprintf(`No responseItem provided. Default Resolver must have a responseItem. Field "%s" has no resolver function, %s %s`, qry.Name, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}
						// now find name element in response, match up to current query field
						switch respItem := responseItems.(type) {
						// response will always be  name:value pairs not a list. TODO - is that correct?
						case sdl.ObjectVals:
							//  { name:value name:value ... } - type ObjectVals []*ArgumentT
							for _, v := range respItem {
								fmt.Println(qry.Name, v.Name_)
								if qry.Name.Equals(v.Name_) {

									switch r := v.Value.InputValueProvider.(type) {

									case sdl.List_:
										switch len(r) {
										case 0:
											writeout(fieldPath, out, "[ ]", noNewLine)

										default:
											if qry.Alias.Exists() {
												writeout(pathRoot, out, qry.Alias.String())
											} else {
												writeout(pathRoot, out, qry.Name.String())
											}
											writeout(pathRoot, out, ":", noNewLine)
											writeout(fieldPath, out, "[", noNewLine)
											for _, k := range r {
												fmt.Printf("== Response is an List of objects/fields . %T - %s .newroot Type : %s\n\n", k, k.String(), newRoot.TypeName())
												writeout(fieldPath, out, "{")

												p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, k.InputValueProvider, out)

												writeout(fieldPath, out, "}")
											}
											writeout(fieldPath, out, "]", noNewLine)
										}

									case sdl.ObjectVals:
										if qry.Alias.Exists() {
											writeout(pathRoot, out, qry.Alias.String())
										} else {
											writeout(pathRoot, out, qry.Name.String())
										}
										writeout(pathRoot, out, ":")
										writeout(fieldPath, out, "{", noNewLine)

										p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, r, out)

										writeout(fieldPath, out, "}", noNewLine)

									default:
										fmt.Println("unknown Input Value Type") //TODO - make an error
									}
									break
								}
							}
						default:
							p.addErr(fmt.Sprintf(`Expected ObjectVals for responseItems. Got %T for  %s %s`, responseItems, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}

					} else { //qry.Resolver != nil

						fmt.Println("About to call resolver....")

						response := qry.Resolver(responseItems, qry.Arguments)

						fmt.Printf(`>>>>>>>  response: "%s"\n`, response)

						// respone returns an object that matches the rootObj i.e. Person, Pet, Address, Business

						if len(response) == 0 {
							p.addErr(fmt.Sprintf(`Resolver for "%s" produced no content, %s %s\n`, qry.Name, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}
						errCnt := len(p.perror)
						//
						// generate type AST from response JSON { name: value name: value ... }
						//
						responseItems = nil
						l := lex.New(response)
						p2 := pse.New(l)
						responseItems = p2.ParseResponse() // similar to sdl.parseArguments. Populates responseItems with parsed values from response.
						fmt.Printf("finished ParseResponse: %T %s\n\n", responseItems, responseItems)
						if len(p2.Getperror()) > 0 {
							// error in parsing stmt from db - this should not happen as only valid stmts are saved.
							p.perror = append(p.perror, p2.Getperror()...)
						}
						fmt.Println("** RootFld Type ", rootFld.Type)
						fmt.Println("*** RootFld Type.IsType() ", rootFld.Type.IsType())
						fmt.Println("*** RootFld Type.Depth ", rootFld.Type.Depth)
						//
						// validate response against type defined in schema statement
						//
						// first set the response as the value in a InputValue_ type
						respname_ := sdl.Name_{Name: sdl.NameValue_("response"), Loc: nil}
						iv := sdl.InputValue_{InputValueProvider: responseItems, Loc: nil}
						errCnt = len(p.perror)
						// now validate the iv against the associated type from the QUERY statement e.g. [Person!]!
						iv.CheckInputValueType(rootFld.Type, respname_, &p.perror)
						// if errors generated during validation abort
						if errCnt != len(p.perror) {
							p.abort = true
							return
						}
						//
						// process each reqponse item and generate output based on query fields in operational statement
						//
						switch y := responseItems.(type) {
						case sdl.List_: //TODO - List_ will  be produced by ParseResponse

							switch len(y) {
							case 0:
								writeout(fieldPath, out, "[ ]", noNewLine)
							case 1:
								if qry.Alias.Exists() {
									writeout(pathRoot, out, qry.Alias.String())
								} else {
									writeout(pathRoot, out, qry.Name.String())
								}
								writeout(pathRoot, out, ":", noNewLine)
								writeout(fieldPath, out, "[ ", noNewLine)
								writeout(fieldPath, out, "{", noNewLine)

								p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, y[0].InputValueProvider, out)

								writeout(fieldPath, out, "}", noNewLine)
								writeout(fieldPath, out, "]")
							default:
								if qry.Alias.Exists() {
									writeout(pathRoot, out, qry.Alias.String())
								} else {
									writeout(pathRoot, out, qry.Name.String())
								}
								writeout(pathRoot, out, ":", noNewLine)
								writeout(fieldPath, out, "[ ", noNewLine)
								for _, k := range y {
									fmt.Printf("++ Response is an List of objects/fields . %T - %s\n\n", k, k.String())
									writeout(fieldPath, out, "{")

									p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, k.InputValueProvider, out)

									writeout(fieldPath, out, "}")
								}
								writeout(fieldPath, out, "]", noNewLine)
							}

						case sdl.ObjectVals: // type ArgumentS []*ArgumentT  -  represents object with fields

							fmt.Println("Reponse is a single object")
							writeout(pathRoot, out, qry.Name.String()+" : ")
							writeout(fieldPath, out, "{", noNewLine)
							p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseItems, out)
							writeout(fieldPath, out, "}", noNewLine)
						default:
							fmt.Printf(" responseItems NOT EITHER %T\n", responseItems)
						}
						// TODO: implement validation of response data
						//responseItems.ValidateResponseValues(rootFld.Type, &p.perror)

						if len(p.perror) > errCnt {
							p.abort = true
							return
						}
					}

				default:
					//
					// scalar
					//
					// if qry.Alias.Exists() {
					// 	fieldPath = pathRoot + "/" + qry.Alias.String()
					// } else {

					// }
					fieldPath = pathRoot + "/" + qry.Name.String()

					if qry.Resolver == nil {
						//
						// implicit resolver - assign response value by field name
						//
						if responseItems == nil {
							p.addErr(`responseItems is empty at scalar resolve execution`)
							p.abort = true
							return
						}
						fmt.Println("xx default response: ")
						switch y := responseItems.(type) {
						// TODO - remove sdl.List_ as responseItem will always be an ObjectVals for scalars
						// case sdl.List_:
						// 	fmt.Println("** Reponse is a list of fields")
						// 	for _, v := range y {

						// 		fmt.Println(v.InputValueProvider.String())

						// 		switch u := v.InputValueProvider.(type) {
						// 		case sdl.List_:
						// 			fmt.Println("Reponse is a list of object")
						// 		case sdl.ObjectVals:
						// 			fmt.Println(rootFld.Name_)
						// 			for _, v := range u {
						// 				fmt.Println(v.Name_)
						// 			}

						// 			fmt.Println("Reponse is a single object")

						// 		}
						// 	}
						case sdl.ObjectVals:
							fmt.Println("Reponse is a single object { name:value name2:value2 ...", fieldPath)
							for _, v2 := range y {
								if v2.Name.EqualString(rootFld.Name_.String()) { // name

									if qry.Alias.Exists() {
										writeout(pathRoot, out, qry.Alias.String())
									} else {
										writeout(pathRoot, out, qry.Name.String())
									}
									writeout(pathRoot, out, ":", noNewLine)
									switch s := v2.Value.InputValueProvider.(type) { // value
									case sdl.String_:
										s_ := string(`"` + s.String() + `"`)
										writeout(fieldPath, out, s_, noNewLine)
									case sdl.RawString_:
										s_ := string(`"""` + s.String() + `"""`)
										writeout(fieldPath, out, s_, noNewLine)
									default:
										s_ := v2.Value.InputValueProvider.String()
										writeout(fieldPath, out, s_, noNewLine)
									}
									break
								}
							}
						default:
							fmt.Printf("Reponse is aunknown %T\n", y)
						}

					} else { // queryResolver != nil

						// response contains AST of type e.g. PET, PERSON (response.TypeName())
						response = qry.Resolver(responseItems, qry.Arguments)

						fmt.Println("field response: ", response)
						errCnt := len(p.perror)
						// generate type AST from response JSON { name: value name: value ... }
						responseItems = nil
						l := lex.New(response)
						p2 := pse.New(l)
						responseItems = p2.ParseResponse() // similar to sdl.parseArguments
						if len(p2.Getperror()) > 0 {
							// error in parsing stmt from db - this should not happen as only valid stmts are saved.
							p.perror = append(p.perror, p2.Getperror()...)
						}
						switch responseItems.(type) {
						case sdl.List_:
							fmt.Println("Response is an List of objects")
						case sdl.ObjectVals:
							fmt.Println("Reponse is a single object")
						}
						if len(p2.Getperror()) > 0 {
							// error in parsing stmt from db - this should not happen as only valid stmts are saved.
							p.perror = append(p.perror, p2.Getperror()...)
						}

						switch y := responseItems.(type) {
						case sdl.List_: // type List_ []*InputValue_ - respresents many sdl.ObjectVals

							switch len(y) {
							case 0:
								writeout(fieldPath, out, "[ ]", noNewLine)
							case 1:
								if qry.Alias.Exists() {
									writeout(pathRoot, out, qry.Alias.String())
								} else {
									writeout(pathRoot, out, qry.Name.String())
								}
								writeout(pathRoot, out, ":", noNewLine)
								writeout(fieldPath, out, "[ ", noNewLine)

								writeout(fieldPath, out, "{", noNewLine)

								p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, y[0].InputValueProvider, out) //responseItems)

								writeout(fieldPath, out, "}", noNewLine)
								writeout(fieldPath, out, "]", noNewLine)
							default:
								if qry.Alias.Exists() {
									writeout(pathRoot, out, qry.Alias.String())
								} else {
									writeout(pathRoot, out, qry.Name.String())
								}
								writeout(pathRoot, out, ":", noNewLine)
								writeout(fieldPath, out, "[ ", noNewLine)
								for _, k := range y {
									fmt.Printf("++ Response is an List of objects/fields . %T - %s\n\n", k, k.String())
									writeout(fieldPath, out, "{")

									p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, k.InputValueProvider, out) //responseItems)

									writeout(fieldPath, out, "}")
								}
								writeout(fieldPath, out, "]", noNewLine)
							}

						case sdl.ObjectVals: // type ArgumentS []*ArgumentT  -  represents object with fields
							fmt.Println("Reponse is a single object")
							if qry.Alias.Exists() {
								writeout(pathRoot, out, qry.Alias.String())
							} else {
								writeout(pathRoot, out, qry.Name.String())
							}
							writeout(pathRoot, out, ":", noNewLine)
							writeout(fieldPath, out, "{", noNewLine)

							p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseItems, out)

							writeout(fieldPath, out, "}", noNewLine)
						default:
							fmt.Printf(" responseItems NOT EITHER %T\n", responseItems)
						}
						// TODO: implement validation of response data
						//responseItems.ValidateResponseValues(rootFld.Type, &p.perror)

						if len(p.perror) > errCnt {
							p.abort = true
							return
						}
					}
				}

				break
			}
			// for object fields recursively call its fields, otherwise return
			// if newRoot != nil && len(x.SelectionSet) != 0 {
			// 	// new root object
			// 	p.executeStmt_(newRoot, x.SelectionSet, pathRoot+"/"+string(newRoot.TypeName()), responseItems, out)

			// }

		case *ast.InlineFragment:

			rootFrag := root
			rootPath := pathRoot

			if !qry.TypeCond.Exists() && len(qry.Directives) == 0 {

				p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseItems, out)

			} else {

				if qry.TypeCond.Exists() {
					// check response fields match those of typeCondition type
					// if typeCondAST=sdl.
					// if ok := qry.ValidateTypeCond(rootObj, &p.perror); ok {
					// 	rootPath += "/" + string(rootFrag.TypeName())
					// } else {
					// 	// response does not match typeCondition - go to next item in set
					// 	continue
					// }
					fmt.Println("Validate type condition against responseItems")
				}

				if len(qry.Directives) == 0 {

					p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseItems, out)

				} else {
					//
					// process directives
					//
					for _, v := range qry.Directives {
						//... @include(if: $expandedInfo) {
						if v.Name_.String() == "@include" {
							for _, arg := range v.Arguments {
								if arg.Name.String() != "if" {
									p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
									return
								}
								// parse wil have populated argument value with variable value.
								argv := arg.Value.InputValueProvider.(sdl.Bool_)
								if argv == true {
									p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseItems, out)
								}
							}
						} else {
							fmt.Println("no @include directive")
							p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseItems, out)
						}
					}
				}
			}
		}
	}
}

// ====================================================================================

func (p *Parser) parseOperationStmt(op string) ast.StatementDef {
	// Types: query, mutation, subscription
	p.nextToken() // read over query, mutation keywords
	stmt := &ast.OperationStmt{Type: op}
	p.root = stmt

	p.parseName(stmt, opt).parseVariables(stmt, opt).parseDirectives(stmt, opt).parseSelectionSet(stmt)

	return stmt

}

func (p *Parser) parseFragmentStmt(op string) ast.StatementDef {
	p.nextToken()               // read over Fragment keyword
	frag := &ast.FragmentStmt{} // TODO: alternative to Stmt field could simply use check len(Name) to determine if Stmt or inline

	_ = p.parseName(frag).parseTypeCondition(frag).parseDirectives(frag, opt).parseSelectionSet(frag)

	FragmentStmts[frag.Name] = frag

	return frag
}

func (p *Parser) parseFragmentSpread() ast.SelectionSetI {
	p.nextToken() // read over ...
	if p.curToken.Type != token.IDENT {
		p.addErr("Identifer expected for fragment spread after ...")
	}
	expnd := &ast.FragementSpread{}
	expnd.AssignName(p.curToken.Literal, p.Loc(), &p.perror) // Evaluation will deref fragment reference with actual fields
	p.nextToken()                                            // read over fragment name
	return expnd
}

func (p *Parser) parseInlineFragment(f ast.HasSelectionSetI) ast.SelectionSetI {

	frag := &ast.InlineFragment{Parent: f}
	p.nextToken() // read over ...

	p.parseTypeCondition(frag, opt).parseDirectives(frag, opt).parseSelectionSet(frag)

	return frag
}

// type Field struct {
// 	Alias     string
// 	Name      string
// 	Arguments []*Argument
// 	//	directives   []directive
// 	SelectionSet []SelectionSetI // field as object
// }

// Field
//  Alias-opt Name Arguments-opt  Directives-opt  SelectionSet-opt

func (p *Parser) parseField() *ast.Field {
	// Field :
	// Alias Name Arguments Directives SelectionSet
	f := &ast.Field{}

	p.parseAlias(f, opt).parseName(f).parseArguments(f, opt).parseDirectives(f, opt).parseSelectionSet(f, opt)

	return f

}

//func (p *Parser) extractFragment() ast.HasSelectionSetI     { return nil }
//func (p *Parser) parseInlineFragment() ast.HasSelectionSetI { return nil }

// =========================================================================

func (p *Parser) parseAlias(f *ast.Field, optional ...bool) *Parser {
	if p.hasError() {
		return p
	}
	// check if alias defined
	if p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON {
		f.AssignAlias(p.curToken.Literal, p.Loc(), &p.perror)
		p.nextToken() // COLON
		p.nextToken() // IDENT - prime for next op
	} else {
		if len(optional) == 0 {
			p.addErr("Expect an alias")
		}
	}
	return p
}

// parseName will validate input data against GraphQL name requirement and assign to a field called Name
func (p *Parser) parseName(f ast.NameI, optional ...bool) *Parser { // type f *ast.Executable,  f=passedInArg converts argument to f
	// check if appropriate thing to do
	if p.hasError() {
		return p
	}
	// alternative tokens, LPAREN+variableDef, ATSIGN+directive, LBRACE-selectionSet
	if p.curToken.Type == token.IDENT {
		f.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
	} else if len(optional) == 0 {
		p.addErr(fmt.Sprintf(`Expected name identifer got %s of %s`, p.curToken.Type, p.curToken.Literal))
		return p
	} else {
		return p
	}
	p.nextToken() // read over name
	return p
}

// ========================================================================

func (p *Parser) parseTypeCondition(f ast.FragmentDef, optional ...bool) *Parser {
	if p.hasError() {
		return p
	}
	if p.curToken.Type != token.ON {
		if len(optional) == 0 {
			p.addErr(fmt.Sprintf("Expecting ON keyword got %s %s", p.curToken.Type, p.curToken.Literal))
		}
		return p
	}
	if p.curToken.Type == token.ON {
		p.nextToken() // read over on
		if p.curToken.Type == token.IDENT {
			f.AssignTypeCond(p.curToken.Literal, p.Loc(), &p.perror)
			p.nextToken() // read over IDENT
		} else {
			p.addErr(fmt.Sprintf("Expecting IDENT for type condition got %s %s", p.curToken.Type, p.curToken.Literal))
		}
	}
	//
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
// only fields have arguments so not interface argument is necessary to support multiple types
func (p *Parser) parseArguments(f sdl.ArgumentAppender, optional ...bool) *Parser {

	if p.hasError() {
		return p
	}
	if p.curToken.Type != token.LPAREN {
		if len(optional) == 0 {
			p.addErr("Expect an argument")
		}
		return p
	}
	//
	parseArgument := func(v *sdl.ArgumentT) error {
		p.printToken("here2 ")
		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			p.abort = true
			return p.addErr(fmt.Sprintf(`Expected an argument name followed by colon got an "%s %s"`, p.curToken.Literal, p.peekToken.Literal))
		}
		v.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
		p.nextToken() // read over :
		p.nextToken() // argument value
		if !((p.curToken.Cat == token.VALUE && (p.curToken.Type == token.DOLLAR && p.peekToken.Cat == token.VALUE)) ||
			(p.curToken.Cat == token.VALUE && (p.peekToken.Cat == token.NONVALUE || p.peekToken.Type == token.RPAREN)) ||
			(p.curToken.Type == token.LBRACKET || p.curToken.Type == token.LBRACE)) { // [  or {
			return p.addErr(fmt.Sprintf(`Expected an argument Value followed by IDENT or RPAREN got an %s:%s:%s %s:%s:%s`, p.curToken.Cat, p.curToken.Type, p.curToken.Literal, p.peekToken.Cat, p.peekToken.Type, p.peekToken.Literal))
		}
		v.Value = p.parseInputValue_()
		p.printToken("After parseINputValue..")
		return nil
	}

	p.nextToken("here ")                  // (
	for p.curToken.Type != token.RPAREN { //p.nextToken() {
		v := new(sdl.ArgumentT)
		if err := parseArgument(v); err != nil {
			break
		}
		if p.curToken.Type == token.EOF {
			p.addErr(fmt.Sprintf(`Expected ) got a "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			break
		}
		//f.Arguments = append(f.Arguments, v)
		f.AppendArgument(v)
	}
	p.nextToken() // prime for next parse op
	return p
}

// Directives[Const]
// 		Directive[?Const]list
// Directive[Const] :
// 		@ Name Arguments[?Const]opt ...
// hero(episode: $episode) {
//     name
//     friends @include(if: $withFriends) @ Size (aa:1 bb:2) @ Pack (filter: true) {
//       name
//     }
func (p *Parser) parseDirectives(f sdl.DirectiveAppender, optional ...bool) *Parser { // f is a iv initialised from concrete types *ast.Field,*OperationStmt,*FragementStmt. It will panic if they don't satisfy DirectiveAppender

	if p.hasError() {
		return p
	}
	if p.curToken.Type != token.ATSIGN {
		if len(optional) == 0 {
			p.addErr("Variable is mandatory")
		}
		return p
	}
	parseArgument := func(d *sdl.DirectiveT) error {
		if !(p.curToken.Type == token.IDENT) {
			return p.addErr(fmt.Sprintf("Expecting a named type identifier go %s %s", p.curToken.Type, p.curToken.Literal))
		}
		// assign to argument Name
		d.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
		// assign arguments
		for p.nextToken(); p.curToken.Type == token.LPAREN; {
			p.parseArguments(d)
		}
		return nil
	}
	for p.curToken.Type == token.ATSIGN {
		p.nextToken() // read over @
		a := []*sdl.ArgumentT{}
		d := &sdl.DirectiveT{Arguments_: sdl.Arguments_{Arguments: a}} // popluate with receiver value for p.parseArguments(d) in parseDirective
		if err := parseArgument(d); err != nil {
			return p
		}
		if err := f.AppendDirective(d); err != nil {
			p.addErr(err.Error())
		}
		if p.curToken.Type != token.ATSIGN {
			break
		}
	}
	return p
}

var Statement bool

// parseSelectionSet - starts with {
// { Selection ... }
// Selection :	Field
//				FragmentSpread
//				InlineFragment
func (p *Parser) parseSelectionSet(f ast.HasSelectionSetI, optional ...bool) *Parser {
	// TODO - sometimes SS is optional other times its mandatory.  How to handle. Idea: method SelectionSetOptional() - which souces data from optional field, array.
	if p.hasError() {
		return p
	}

	if p.curToken.Type != token.LBRACE {
		if len(optional) == 0 {
			p.addErr("Expect a selection set")
		}
		return p
	}
	parseSSet := func() ast.SelectionSetI {
		var node ast.SelectionSetI

		switch p.curToken.Type {

		case token.IDENT:
			node = p.parseField() // returns field struct which itself may contain another selectionSet

		case token.EXPAND:
			if p.peekToken.Type == token.ON || p.peekToken.Type == token.ATSIGN || p.peekToken.Type == token.LBRACE {
				node = p.parseInlineFragment(f)
			} else if p.peekToken.Type == token.IDENT {
				node = p.parseFragmentSpread()
			} else {
				p.addErr("expected IDENT or ON or @ or LBRACE after spread ...")
			}

		default:
			switch p.curToken.Type {
			case "Int":
				p.addErr(fmt.Sprintf("Expected an identifier got %s. Probable cause is identifers cannot start with a number", p.curToken.Type))
			default:
				p.addErr(fmt.Sprintf("Expected an identifier for a fragment or inlinefragment got %s.", p.curToken.Type))
			}
		}
		return node
	}
	// read an LBRACE therefore have a selectionset to process. Each node/item in the SS must be either a Field, FragmentSpread, InlineFragment
	for p.nextToken(); p.curToken.Type != token.RBRACE; {

		node := parseSSet()

		if p.hasError() {
			break
		}
		f.AppendSelectionSet(node) // append each selection set current receiver.

	}
	p.nextToken() //
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
func (p *Parser) parseVariables(st ast.OperationDef, optional ...bool) *Parser { // st is an iv initialised from passed in argument which is a *OperationStmt

	if p.hasError() {
		return p
	}

	parseVariable := func(v *ast.VariableDef) bool {
		p.nextToken()
		if p.curToken.Type != token.DOLLAR {
			p.addErr(fmt.Sprintf(`Expected "$" got "%s"`, p.curToken.Literal))
			return true
		}
		p.nextToken() // read over name identifer

		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			p.addErr(fmt.Sprintf(`Expected an identifer got an "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return true
		}
		v.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
		p.nextToken()
		// :
		if p.curToken.Type != token.COLON {
			p.addErr(fmt.Sprintf(`Expected : got a "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return true
		}
		p.nextToken() // read over :
		p.parseType(v)
		if p.curToken.Type == token.ASSIGN {
			//	p.nextToken() // read over Datatype
			p.nextToken() // read over ASSIGN
			v.DefaultVal = p.parseInputValue_()
			return true
		}
		return false
	}

	if p.hasError() {
		return p
	}

	switch stmt := st.(type) {
	case *ast.OperationStmt:
		if p.curToken.Type == token.LPAREN {
			for ; p.curToken.Type != token.RPAREN; p.nextToken() {

				v := ast.VariableDef{}

				if parseVariable(&v) {
					stmt.Variable = append(stmt.Variable, &v)
				} else {
					return p
				}
			}
			p.rootVar = stmt.Variable
			//xyz	p.nextToken() //read over )
		} else if len(optional) == 0 { // if argument exists its optional
			p.addErr("Variables are madatory")
		}
	default:
		p.addErr("Variables are only permitted in Operational statements")
	}

	return p
}

// parseObjectArguments - used for input object values
func (p *Parser) parseObjectArguments(argS []*sdl.ArgumentT) []*sdl.ArgumentT {
	//p.nextToken("begin parseObjectArguments");
	for p.curToken.Type == token.IDENT {
		//for p.nextToken(); p.curToken.Type != token.RBRACE ; p.nextToken() { // TODO: use this
		v := new(sdl.ArgumentT)

		p.parseName(v).parseColon().parseInputValue(v)

		argS = append(argS, v)

	}
	return argS
}

func (p *Parser) parseInputValue(v *sdl.ArgumentT) *Parser {
	if p.hasError() {
		return p
	}

	v.Value = p.parseInputValue_()

	return p
}

// parseInputValue_ used to interpret "default value" in argument and field values.
//  parseInputValue_ expects an InputValue_ literal (true,false, 234, 23.22, "abc" or $variable in the next token.  The value is a type bool,int,flaot,string..
//  if it is a variable then the variable value (which is an InputValue_ type) will be sourced
//  TODO: currently called from parseArgument only. If this continues to be the case then add this func as anonymous func to it.
//func (p *Parser) parseInputValue_(iv ...*ast.InputValueDef) *sdl.InputValue_ { //TODO remove iv argeument now redundant
func (p *Parser) parseInputValue_() *sdl.InputValue_ {
	defer p.nextToken() // this func will finish paused on next token - always

	if p.curToken.Type == "ILLEGAL" {
		p.addErr(fmt.Sprintf("Value expected got %s of %s", p.curToken.Type, p.curToken.Literal))
		p.abort = true
		return nil
	}
	switch p.curToken.Type {

	case token.DOLLAR:
		// variable supplied - need to fetch value
		p.nextToken() // IDENT variable name
		// change category of token to VALUE as previous token was $ - otherwise this step would not be executed.
		p.curToken.Cat = token.VALUE
		if p.curToken.Type == token.IDENT {
			// get variable value....
			if val, ok := p.getVarValue(p.curToken.Literal); !ok {
				p.addErr(fmt.Sprintf("Variable, %s not defined ", p.curToken.Literal))
				return nil
			} else {
				return val
			}
		} else {
			p.addErr(fmt.Sprintf("Expected Variable Name Identifer got %s", p.curToken.Type))
			return nil
		}
	//
	// List type
	//
	case token.LBRACKET:
		// [ value value value .. ]
		p.nextToken() // read over [
		if p.curToken.Cat != token.VALUE {
			p.addErr(fmt.Sprintf("Expect an Input Value followed by another Input Value or a ], got %s %s ", p.curToken.Literal, p.peekToken.Literal))
			return &sdl.InputValue_{}
		}
		// edge case: empty, []
		if p.curToken.Type == token.RBRACKET {
			p.nextToken() // ]
			var null sdl.Null_ = true
			iv := sdl.InputValue_{InputValueProvider: null, Loc: p.Loc()}
			return &iv
		}
		// process list of values - all value types should be the same
		var vallist sdl.List_
		for p.curToken.Type != token.RBRACKET {
			v := p.parseInputValue_()
			vallist = append(vallist, v)
		}
		// completed processing values, return List type
		iv := sdl.InputValue_{InputValueProvider: vallist, Loc: p.Loc()}
		return &iv
	//
	//  Object type
	//
	case token.LBRACE:
		//  { name:value name:value ... }
		p.nextToken()              // read over {
		var ObjList sdl.ObjectVals // []*ArgumentT {Name_,Value *InputValue_}
		for p.curToken.Type != token.RBRACE {

			ObjList = p.parseObjectArguments(ObjList)
			if p.hasError() {
				return &sdl.InputValue_{}
			}
		}
		iv := sdl.InputValue_{InputValueProvider: ObjList, Loc: p.Loc()}
		return &iv
	//
	//  Standard Scalar types
	//
	case token.NULL:
		var null sdl.Null_ = true
		iv := sdl.InputValue_{InputValueProvider: null, Loc: p.Loc()}
		return &iv
	case token.INT:
		fmt.Println("Int : ", p.curToken.Literal)
		i := sdl.Int_(p.curToken.Literal)
		iv := sdl.InputValue_{InputValueProvider: i, Loc: p.Loc()}
		return &iv
	case token.FLOAT:
		f := sdl.Float_(p.curToken.Literal)
		iv := sdl.InputValue_{InputValueProvider: f, Loc: p.Loc()}
		return &iv
	case token.STRING:
		fmt.Println("String: ", p.curToken.Literal)
		f := sdl.String_(p.curToken.Literal)
		iv := sdl.InputValue_{InputValueProvider: f, Loc: p.Loc()}
		return &iv
	case token.RAWSTRING:
		f := sdl.RawString_(p.curToken.Literal)
		iv := sdl.InputValue_{InputValueProvider: f, Loc: p.Loc()}
		return &iv
	case token.TRUE, token.FALSE: //token.BOOLEAN:
		var b sdl.Bool_
		if p.curToken.Literal == "true" {
			b = sdl.Bool_(true)
		} else {
			b = sdl.Bool_(false)
		}
		iv := sdl.InputValue_{InputValueProvider: b, Loc: p.Loc()}
		return &iv
	// case token.Time:
	// 	b := sdl.Time_(p.curToken.Literal)
	// 	iv := sdl.InputValue_{Value: b, Loc: p.Loc()}
	// 	return &iv
	default:
		// possible ENUM value
		b := &sdl.EnumValue_{}
		b.AssignName(string(p.curToken.Literal), p.Loc(), &p.perror)
		iv := sdl.InputValue_{InputValueProvider: b, Loc: p.Loc()}
		return &iv
	}
	return nil

}

// rootvar: &ast.VariableDef{
// Name:"devicePicSize",
// inputValueType_:"Int",
// DefaultVal:sdl.InputValue_{Value:(*ast.Scalar_)(0xc420050440), inputValueType_:"Int"},
// Value:sdl.InputValue_{Value:ast.ValueI(nil), inputValueType_:""}
// }
func (p *Parser) getVarValue(name string) (*sdl.InputValue_, bool) {
	for _, v := range p.rootVar {
		//fmt.Printf(" rootvar: %#v . %s \n", v, v.DefaultVal.String())
		if v.Name == sdl.NameValue_(name) {
			if v.Value != nil {
				return v.Value, true
			} else {
				return v.DefaultVal, true
			}
		}
	}
	return &sdl.InputValue_{}, false
}

func (p *Parser) parseColon() *Parser {

	if !(p.curToken.Type == token.COLON) {
		p.addErr(fmt.Sprintf(`Expected a colon got an "%s"`, p.curToken.Literal))
	}
	p.nextToken() // read over :
	return p
}

// TODO - investigate using sdl parseType.
// option 1 - use sdl parser struct not local one would make it easier.
// func (p *Parser) parseType(f sdl.AssignTyper) *Parser {
// 	sdl.ParseType(p, f)
// }

func (p *Parser) parseType(f sdl.AssignTyper) *Parser {

	if p.hasError() {
		return p
	}
	// if p.curToken.Type == token.COLON {
	// 	p.nextToken() // read over :
	// } else {
	// 	p.addErr(fmt.Sprintf("Colon expected got %s of %s", p.curToken.Type, p.curToken.Literal))
	// }
	if !p.curToken.IsScalarType { // ie not a Int, Float, String, Boolean, ID, <namedType>
		if !(p.curToken.Type == token.IDENT || p.curToken.Type == token.LBRACKET) {
			p.addErr(fmt.Sprintf("Expected a Type, got %s, %s", p.curToken.Type, p.curToken.Literal))
		} else if p.curToken.Type == "ILLEGAL" {
			p.abort = true
			return p
		}
	}
	var (
		bit  byte
		name string
		//	ast_ ast.GQLTypeProvider
		//typedef ast.TypeFlag_ // token defines SCALAR types only. All other types will be populated in repoType map.
		depth   int
		nameLoc *sdl.Loc_
	)
	nameLoc = p.Loc()
	switch p.curToken.Type {

	case token.LBRACKET:
		// [ typeName ]
		var (
			depthClose uint
		)
		p.nextToken() // read over [
		for depth = 1; p.curToken.Type == token.LBRACKET; p.nextToken() {
			depth++
		}
		if depth > 7 {
			p.addErr("Nested list type cannot be greater than 8 deep ")
			break
		}
		if !(p.curToken.Type == token.IDENT || p.curToken.IsScalarType) {
			p.addErr(fmt.Sprintf("Expected type identifer got %s, %s", p.curToken.Type, p.curToken.Literal))
			break
		}
		nameLoc = p.Loc()
		name = p.curToken.Literal // actual type name, Int, Float, Pet ...
		// name_ := sdl.Name_{Name: sdl.NameValue_(name), Loc: nameLoc}
		// //System ScalarTypes are defined by the Type_.Name_, Non-system Scalar and non-scalar are defined by the AST.
		// if !p.curToken.IsScalarType {
		// 	ast_ = p.fetchAST(name_)
		// }
		p.nextToken() // read over IDENT
		for bangs := 0; p.curToken.Type == token.RBRACKET || p.curToken.Type == token.BANG; {
			if p.curToken.Type == token.BANG {
				bangs++
				if bangs > depth+1 {
					p.addErr("redundant !")
					p.nextToken() // read over !
					//return p
				} else {
					bit |= (1 << depthClose)
					p.nextToken() // read over !
				}
			} else {
				depthClose++
				p.nextToken() // read over ]
			}
		}
		if depth != int(depthClose) {
			p.addErr("close ] does not match opening [ in type specification")
			return p
		}

	default:
		if p.curToken.Type == token.IDENT || p.curToken.IsScalarType {
			name = p.curToken.Literal
			if p.peekToken.Type == token.BANG {
				bit = 1 << 0
				p.nextToken() // read over IDENT
			}
			p.nextToken() // read over ! or IDENT
		} else {
			p.addErr(fmt.Sprintf("Expected type identifer got %s, %s %v", p.curToken.Type, p.curToken.Literal, p.curToken.IsScalarType))
		}
	}

	if p.hasError() {
		return p
	}
	// name is the type name Int, Person, [name], ...
	t := &sdl.Type_{Constraint: bit, Depth: depth} //, AST: ast_}
	t.AssignName(name, nameLoc, &p.perror)
	f.AssignType(t) // assign the name of the named type. Later type validation pass of AST will confirm if the named type exists.
	return p

}
