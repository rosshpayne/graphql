package parser

import (
	"errors"
	"fmt"
	_ "os"
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
	defaultDoc = "DefaultDoc"
)

type Argument struct {
	Name  string
	Value string
}

type (
	parseFn func(op string) ast.StatementDef

	Parser struct {
		l        *lexer.Lexer
		document string
		extend   bool

		abort bool
		// schema rootAST

		curToken  token.Token
		peekToken token.Token

		responseMap map[string]*sdl.InputValueProvider //struct{}
		respOrder   []string                           // slice of field paths in order executed.
		//response  []*ast.ResponseValue // conerts response from reolver  to internal sdl.ObjectVal

		root    ast.StatementDef
		rootVar []*ast.VariableDef

		Resolver *resolver.Resolvers

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

	p.Resolver = resolver.New()

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

// astsitory of all types defined in the graph

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

func (p *Parser) ParseDocument(doc ...string) (*ast.Document, []error) {
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
	//
	// set document
	//
	ast.SetDefaultDoc(defaultDoc)
	sdl.SetDefaultDoc(defaultDoc)
	if len(doc) == 0 {
		ast.SetDocument(defaultDoc)
		sdl.SetDocument(defaultDoc)
	} else {
		ast.SetDocument(doc[0])
		sdl.SetDocument(doc[0])
	}
	//
	// fetch schema within the document
	//
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

			if typeDef, err := sdl.DBFetch(name_); err != nil {

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

var noNewLine bool = true

// ================== writeout ======================================
// writeout prints out the Grapql JSON passed to it.
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

// ================== executeStmt ======================================

func (p *Parser) executeStmt(root sdl.GQLTypeProvider, stmt_ ast.StatementDef) {

	var (
		stmt *ast.OperationStmt
		ok   bool
		out  strings.Builder
	)
	var (

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

	fmt.Println("Resolver paths: ")
	fmt.Println(p.Resolver.String())
	fmt.Println("================================ executeStmt ================================")
	out.WriteString("{ data: {")

	p.executeStmt_(root, stmt.SelectionSet, string(root.TypeName()), nil, &out)

	fmt.Println("==== output ====== ")
	fmt.Println(out.String())
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
	// {response : {														<== resolver response data (source data)
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
	//
	// ** High level description **
	//
	// loop on query fields (selection set passed in - initially query selection set)
	//
	//  on field type
	//
	//	 loop on root-type fields
	//    on match by name
	//
	//		field type is Object -- AAA ----
	//
	//        field has no resolver
	//			loop on response fields (from previous resolver call)
	//            on match by name (so query/root-type/data is now matched)
	//             get resp field value (data)
	//				 validate response for current field and recusively executeStmt with data
	//
	//
	//  	  field has resolver --- BBB ---
	//			match field to response data if response from previous execute exists (will not on first initial execution)
	//            assign response data to "resp" argument (as field is an object.) TODO - make input object
	//          ** execute resolver **
	//          parse resolver output (JSON) and generate AST representation, using SDL parser
	//          partially validate AST components against expected root-type and recursively call executeStmt using AST as input
	//
	//		field type is scalar  --- CCC ---
	//
	//        field has no resolver
	//          loop response object
	//			  on match with root-type field (by name)
	//				validate resp data against type
	//				output query JSON for field (as either List or single field)
	//
	//        field has resolver  --- DDD ---
	//			match field to response data if response from previous execute exists (will not on first initial execution)
	//            assign response data to "resp" argument (as field is an object.) TODO - make input object
	//          ** execute resolver **
	//          parse resolver output (JSON) and generate AST representation, using SDL parser
	//          partially validate AST components against expected root-type
	//          output query JSON for field (as either List or single field)
	//
	//  on inline-Fragment type
	//
	//
	var (
		rootObj *sdl.Object_
	)

	if p.hasError() {
		return
	}
	if root == nil {
		p.addErr("In executeStmt_, passed in a root of nil")
		return
	}

	rootObj = root.(*sdl.Object_)

	// scan selection set passed in.  Order of search, Query Field match with Root field match with Response Field (all matching based on Name attribute)
	for _, qryFld := range set {

		switch qry := qryFld.(type) {

		case *ast.Field:
			fmt.Println("\n\n*** Query field: ", qry.Name)
			var (
				newRoot   sdl.GQLTypeProvider
				fieldPath string
				fieldName string
				response  string
				rootFld   *sdl.Field_
			)
			fmt.Println("******* qry.Name ", qry.Name)
			// rootFld = qryFldMap[pathRoot] // could access via map but thinking about memory requirements for maps, when simple scan-loop swaps CPU for scan instead of memory
			// match field name to root object's AST to determine field's type
			// employee : [Person],  employee is the rootFld, and its type rootFld.Type (slice inof) rootFld.Type.AST (Person)
			// height: Float,        height is the roofFld, and type name = 'Float' (no AST as its a scalar)
			for _, rootFld = range rootObj.FieldSet {

				if !qry.Name.Equals(rootFld.Name_) {
					continue
				}
				//
				// root field matches query field - got field type now
				//
				fmt.Println("found : ", rootFld.Name_, qry.Name)
				if qry.Alias.Exists() {
					fieldName = qry.Alias.String()
				} else {
					fieldName = qry.Name.String()
				}
				//
				// associated GraphQL type (type system)
				//
				switch rootFld.Type.AST.(type) {
				//  -- AAA ----
				case *sdl.Object_:
					//
					// object field, details in AST (as it is not a scalar)
					//
					newRoot = rootFld.Type.AST // qryFld's matching type either scalar or object based (in AST)
					fieldPath = pathRoot + "/" + rootFld.Name_.String()

					fmt.Println("********** pathRoot, fieldPath: ", rootFld.Name_.String(), fieldPath, qry.Name)

					qry.Resolver = p.Resolver.GetFunc(fieldPath)

					if qry.Resolver == nil {
						//
						// use data from last resolver execution, passed in via argument "responseItems"
						//
						if responseItems == nil {
							p.addErr(fmt.Sprintf(`No responseItem provided. Default Resolver must have a responseItem. Field "%s" has no resolver function, %s %s`, qry.Name, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}
						//
						// find element in response that matches current query field. RespItem uses InputValue_
						//
						switch respItem := responseItems.(type) {
						// response will always be "FieldName:value" pairs e.g. { data: [ { } { } ], where value may be a List_ or another ObjectVal or a scalar
						// as a result the first (top entry) will always be an ObjectVals type
						case sdl.ObjectVals:
							//  { name:value name:value ... } -  type ObjectVals []*ArgumentT   type ArgumentT struct { Name_, Value *InputValue_}   type InputValue {InputValueProvider, Loc}
							for _, respfld := range respItem {

								if !qry.Name.Equals(respfld.Name_) {
									continue
								}
								//
								//  found response field now compare its value type against expected (root) type
								//
								writeout(pathRoot, out, fieldName+"-O")
								writeout(pathRoot, out, ":", noNewLine)
								if _, ok := respfld.Value.InputValueProvider.(sdl.List_); ok {
									if rootFld.Type.Depth == 0 {
										p.addErr(fmt.Sprintf(`Expected single value got List for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
										p.abort = true
										return
									}
								} else {
									if rootFld.Type.Depth > 0 {
										p.addErr(fmt.Sprintf(`Expected List of values got single value instead for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
										p.abort = true
										return
									}
								}
								switch riv := respfld.Value.InputValueProvider.(type) {

								case sdl.List_:
									//TODO include nullable check
									fmt.Println("+++++ rootFld.Type.IsType(), riv.IsType() = ", rootFld.Type.IsType(), riv.IsType())
									if rootFld.Type.Depth == 0 {
										p.addErr(fmt.Sprintf(`Expected a single value for "%s" , response returned a List  %s`, rootFld.Name, qry.Name.AtPosition()))
									}

									var f func(y sdl.List_, d int)
									// f will output sdl.List_ for any level of nesting
									// d is the nesting depth of List_
									f = func(y sdl.List_, d int) {

										for i := 0; i < len(y); i++ {
											if x, ok := y[i].InputValueProvider.(sdl.List_); ok {
												writeout(fieldPath, out, "[ ", noNewLine)
												d++ // nesting depth of List_
												if d > rootFld.Type.Depth {
													p.addErr(fmt.Sprintf(`Exceeds nesting of List type for "%s" %s`, qry.Name, qry.Name.AtPosition()))
												}
												f(x, d)
												writeout(fieldPath, out, "] ", noNewLine)
												d--
											} else {
												if d < rootFld.Type.Depth {
													p.addErr(fmt.Sprintf(`Expect a nesting level of %d, got %d, for scalar values in List for "%s" %s`, rootFld.Type.Depth, d, qry.Name, qry.Name.AtPosition()))
												}
												// optimise by performing loop here rather than use outer for loop
												for i := 0; i < len(y); i++ {

													writeout(fieldPath, out, "o{")

													p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, y[i].InputValueProvider, out)

													writeout(fieldPath, out, "}o")
												}
												break
											}
										}
									}
									fmt.Println("List - Object...")
									writeout(fieldPath, out, "[ ", noNewLine)
									f(riv, 1)
									writeout(fieldPath, out, "] ", noNewLine)

								case sdl.ObjectVals:
									//  compare with root type
									if rootFld.Type.Depth != 0 {
										p.addErr(fmt.Sprintf(`Expected List of values for "%s", resolver response returned single value %s`, rootFld.Name, qry.Name.AtPosition()))
									}
									//TODO include nullable check
									if rootFld.Type.IsType() != riv.IsType() {
										p.addErr(fmt.Sprintf(`2 Expected type of "%s" got %s instead for field "%s" %s`, rootFld.Type.IsType(), riv.IsType(), rootFld.Name, qry.Name.AtPosition()))
									}
									fmt.Printf("== Response is OBJECTVALS of objects/fields .")
									writeout(fieldPath, out, "{", noNewLine)

									p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, riv, out)

									writeout(fieldPath, out, "}", noNewLine)

								default:
									// as root type is an object we shoul not get a scalar type - so switch default represenets an error
									// compare with root type
									if rootFld.Type.Depth != 0 {
										p.addErr(fmt.Sprintf(`Expected List of values for "%s" , resolver response returned single value instead %s`, rootFld.Name, qry.Name.AtPosition()))
									}
									//TODO include nullable check
									if rootFld.Type.IsType() != riv.IsType() {
										p.addErr(fmt.Sprintf(`3 Expected type of "%s" got %s instead for field "%s" %s`, rootFld.Type.IsType(), riv.IsType(), rootFld.Name, qry.Name.AtPosition()))
									}
									p.addErr(fmt.Sprintf(`Expected Object type got scalar  %s`, qry.Name.AtPosition()))
									p.abort = true
									return
								}
								break
							}

						default:
							p.addErr(fmt.Sprintf(`Resolver response returned something other than name:value pairs. Got %T for  %s %s`, responseItems, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}

					} else {
						//  --- BBB ----
						//
						// Resolver exists for field object
						//
						// if we have response data find the associated response field. First time through response will be nil as no resolver has been called.
						//
						var (
							resp sdl.InputValueProvider
							//	mismatchTypes bool
							respType sdl.TypeFlag_
							argFound bool
						)
						//
						// First time through responseItems will be nil as no resolver has yet to be called.
						//  On subsequent recursive calls it will contain response data from the last resolve call (the one  about to be executed below).
						//  The objective will be to match the current query/root field with the associated field in the response data. If the response field's type does not
						//  the root field then try matching the reponse data against any arguments associated with the query field. If it matches then use the response data
						//  as input when executing the resolver.
						//
						// find response using Name. List_ can only ever be field data.
						//
						if responseItems != nil {
							switch respItem := responseItems.(type) {
							case sdl.ObjectVals:
								// { field: value, field: value ... } type ObjectVals []*ArgumentT   type ArgumentT struct { Name_, Value *InputValue_}   type InputValue { InputValueProvider, Loc *Loc_
								//
								// find response field matching current root/query field name
								//
								for _, response := range respItem {
									// match response field against root field and  grab the associated response field data.
									fmt.Println("Searching.. ", response.Name, rootFld.Name)
									if response.Name.EqualString(rootFld.Name_.String()) { // name
										resp = response.Value.InputValueProvider
										break
									}
								}
							}
							if resp == nil {
								p.addErr(fmt.Sprintf("XX No corresponding root field found from response "))
								p.abort = true
								return
							}
							//
							//	*** found response field
							//  so we now have circumstance where the query field has a resolver but we also have response data for this field.
							//  under this circumstance the reponse data must feed into the resolver via the "resp" argument. //TODO use input type rather than resp - maybe
							//
							switch y := resp.(type) {

							case sdl.List_:
								respType = y[0].InputValueProvider.IsType()

							case sdl.ObjectVals:
								// {field: value, field: value ... }, essentially an object to match againts an ast.Object_ (the fieldSet) root type
								//TODO - complete implementation for ObjectVals
								for _, v := range y {
									fmt.Println("embedded type for ObjectVals: ", v.Name_, v.Value.InputValueProvider.IsType())
								}
								//respType = ??

							default:
								// scalar types, Int, Float, String, EnumValues - as rootFld.Type is an Object (see above), scalars should not appear here.
								p.addErr(fmt.Sprintf(`Expect object type for response field "%s", got scalar type field %s`, qry.Name, qry.Name.AtPosition()))
								p.abort = true
								return

							}
							//
							// assign response field data to "resp" argument
							//
							fmt.Println("find resp argument and substitute response data..")
							for _, arg := range rootFld.ArgumentDefs {
								fmt.Println(" match arguments: ", arg.Name, respType, arg.Type.IsType(), arg.Type.IsType2(), resp.IsType())
								if arg.Type.IsType() == respType && arg.Type.IsType2() == resp.IsType() && arg.Name.EqualString("resp") {
									fmt.Println("matched.....")
									// append a "resp" argument to the query Arguments
									// does resp exist in query arguments for current field
									var respArg *sdl.ArgumentT
									for _, qarg := range qry.Arguments { // TODO check how this gets populated with resp argument from root definiton
										if qarg.Name_.EqualString("resp") {
											respArg = qarg
										}
									}
									iv := sdl.InputValue_{InputValueProvider: resp}
									if respArg != nil {
										respArg.Value = &iv
									} else {
										argT := sdl.ArgumentT{Value: &iv}
										argT.AssignName("resp", nil, nil)
										qry.Arguments = append(qry.Arguments, &argT)
									}
									argFound = true
									break
								}
							}
							if !argFound {
								p.addErr(fmt.Sprintf(`Response data does not match required type "%s" or any resp argument in query field "%s"`, rootFld.Type.TypeName(), qry.Name))
								p.abort = true
								return
							}
						}
						//}
						//
						// response data maybe nil (first time through) or supplied from recursive call via func argument
						//
						if resp == nil {
							resp = responseItems
						}
						//
						//  execute Resolver - using current response data (nil for the first time) and any arguments associated with field
						//
						response := qry.Resolver(resp, qry.Arguments)

						fmt.Printf(`>>>>>>>  response: "%s"`, response)
						fmt.Println()
						//
						// respone returns an object that matches the rootObj i.e. Person, Pet, Address, Business
						//
						if len(response) == 0 {
							p.addErr(fmt.Sprintf(`Resolver for "%s" produced no content, %s %s\n`, qry.Name, rootObj.TypeName(), qry.Name.AtPosition()))
							p.abort = true
							return
						}
						errCnt := len(p.perror)
						//
						// generate AST from response JSON { name: value name: value ... }
						//
						responseItems = nil
						l := lex.New(response)
						p2 := pse.New(l)
						responseItems = p2.ParseResponse() // similar to sdl.parseArguments. Populates responseItems with parsed values from response.
						fmt.Printf("finished ParseResponse: %T %s\n\n", responseItems, responseItems)
						if responseItems == nil {
							p.addErr(fmt.Sprintf(`Empty response from resolver for "%s" %s`, rootFld.Name, qry.Name.AtPosition()))
						}
						if len(p2.Getperror()) > 0 {
							// error in parsing stmt from db - this should not happen as only valid stmts are saved.
							p.perror = append(p.perror, p2.Getperror()...)
						}
						fmt.Println("** RootFld Type ", rootFld.Type, rootFld.Type.IsType2().String())                                       // [Post!] List
						fmt.Println("*** RootFld Type.IsType().String() ", rootFld.Type.IsType().String(), rootFld.Name, newRoot.TypeName()) // Object posts Post
						fmt.Printf("*** RootFld Type.Depth %#v, %d \n", rootFld, rootFld.Type.Depth)
						//
						// *** Commented out as CheckInputValueType is nolonger suitable as response maynot match root type completely. ***
						//
						// validate response against type defined in schema statemen
						// respname_ := sdl.Name_{Name: sdl.NameValue_("response"), Loc: nil}
						// iv := sdl.InputValue_{InputValueProvider: responseItems, Loc: nil}
						// iv.CheckInputValueType(rootFld.Type, respname_, &p.perror)
						// if errCnt != len(p.perror) {...
						//
						// process each reqponse item and generate output based on query fields in operational statement
						//
						// responseItems - InputValueProvider						responseItems = nil
						writeout(pathRoot, out, fieldName)
						writeout(pathRoot, out, ":", noNewLine)
						// find field "data" and access its data
						//
						// only interested in value component of response.      {name: value} e.g {data:["abc" "def"] interested only in ["abc" "def"]
						//
						fmt.Printf("x %T\n", responseItems)
						if x, ok := responseItems.(sdl.ObjectVals); ok {
							responseItems = x[0].Value.InputValueProvider
						}
						//
						if _, ok := responseItems.(sdl.List_); ok {
							if rootFld.Type.Depth == 0 {
								p.addErr(fmt.Sprintf(`Expected single value got List for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
								p.abort = true
								return
							}
						} else {
							if rootFld.Type.Depth > 0 {
								p.addErr(fmt.Sprintf(`Expected List of values got single value instead for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
								p.abort = true
								return
							}
						}

						switch resp := responseItems.(type) {
						case sdl.List_:
							//TODO include nullable check
							// Type check of list members will be performed in the following executeStmt checks.
							fmt.Println("newRoot: ", newRoot.TypeName())
							fmt.Println("qry.SS . ", len(qry.SelectionSet))
							fmt.Println("fieldPath: ", fieldPath)
							//TODO include nullable check
							fmt.Println("after resolver call - List data = ")
							if rootFld.Type.Depth == 0 {
								p.addErr(fmt.Sprintf(`Expected a single value for "%s" , response returned a List  %s`, rootFld.Name, qry.Name.AtPosition()))
							}

							var f func(y sdl.List_, d int)
							// f will output sdl.List_ for any level of nesting
							// d is the nesting depth of List_
							f = func(y sdl.List_, d int) {

								for i := 0; i < len(y); i++ {
									if x, ok := y[i].InputValueProvider.(sdl.List_); ok {
										writeout(fieldPath, out, "[ ", noNewLine)
										d++ // nesting depth of List_
										if d > rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Exceeds nesting of List type for "%s" %s`, qry.Name, qry.Name.AtPosition()))
										}
										f(x, d)
										writeout(fieldPath, out, "] ", noNewLine)
										d--
									} else {
										if d < rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Expect a nesting level of %d, got %d, for scalar values in List for "%s" %s`, rootFld.Type.Depth, d, qry.Name, qry.Name.AtPosition()))
										}
										// optimise by performing loop here rather than use outer for loop
										for i := 0; i < len(y); i++ {

											writeout(fieldPath, out, "or{")

											p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, y[i].InputValueProvider, out)

											writeout(fieldPath, out, "}or")
										}
										break
									}
								}
							}
							fmt.Println("List - Object...")
							writeout(fieldPath, out, "or[ ", noNewLine)
							f(resp, 1)
							writeout(fieldPath, out, " ]or", noNewLine)

							// switch len(y) {
							// case 0:
							// 	writeout(fieldPath, out, "[ ]", noNewLine) // responseItems is nil
							// default:
							// 	writeout(fieldPath, out, "[ ", noNewLine)
							// 	for _, k := range y {
							// 		fmt.Printf("++ Response is an List of objects/fields . %T - %s\n\n", k, k.String())
							// 		writeout(fieldPath, out, "{")

							// 		p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, k.InputValueProvider, out)

							// 		writeout(fieldPath, out, "}")
							// 	}
							// 	writeout(fieldPath, out, "]", noNewLine)
							// }

						case sdl.ObjectVals: // type ArgumentS []*ArgumentT  -  represents object with fields
							if rootFld.Type.Depth > 0 {
								p.addErr(fmt.Sprintf("Expected %s, got a response of name value pairs (ObjectVals) \n", rootFld.Type.IsType2().String()))
								p.abort = true
								return
							}
							fmt.Println("Reponse is a single object")
							writeout(pathRoot, out, qry.Name.String()+" : ")
							writeout(fieldPath, out, "{", noNewLine)

							p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseItems, out)

							writeout(fieldPath, out, "}", noNewLine)
						default:
							//TODO implement scalar code
							fmt.Printf(" responseItems NOT EITHER %T\n", responseItems)
						}

						if len(p.perror) > errCnt {
							p.abort = true
							return
						}
					}

				default:
					//  --- CCC ----
					//
					// scalar or List of scalar
					//
					fieldPath = pathRoot + "/" + qry.Name.String()
					fmt.Printf("xx root is a non-object response: %T   fieldPath . %s\n", responseItems, fieldPath)
					qry.Resolver = p.Resolver.GetFunc(fieldPath)

					if qry.Resolver == nil {

						fmt.Println("NO RESOLVER...")
						//
						// implicit resolver - assign response value by field name
						//
						if responseItems == nil {
							p.addErr(`responseItems is empty at scalar resolve execution`)
							p.abort = true
							return
						}
						writeout(pathRoot, out, fieldName+"-X")
						writeout(pathRoot, out, ":", noNewLine)
						var resp sdl.InputValueProvider
						switch r := responseItems.(type) {
						case sdl.ObjectVals:
							for _, response := range r {
								//
								// find response field by matching name against root field and  grab the associated response field data.
								//
								if response.Name.EqualString(rootFld.Name_.String()) { // name
									resp = response.Value.InputValueProvider
								}
							}
						}
						//
						// found matching response field
						//
						if _, ok := resp.(sdl.List_); ok {
							if rootFld.Type.Depth == 0 {
								p.addErr(fmt.Sprintf(`Expected single value got List for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
								p.abort = true
								return
							}
						}
						if _, ok := resp.(sdl.List_); !ok {
							if rootFld.Type.Depth > 0 {
								p.addErr(fmt.Sprintf(`Expected List of values got single value instead for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
								p.abort = true
								return
							}
						}
						switch riv := resp.(type) { // value

						case sdl.List_:
							//type List_ []*InputValue_ . type InputValue_ struct {InputValueProvider	,Loc  *Loc_}
							// [                                                             ]     sdl.List_        depth=3
							//  [                          ] [              ] [             ]       sdl.List_        depth=2
							//   [1 2 3] [1 2 3 12] [23 32]   [23 23] [2 5]    [3 5] [3 6 6]         sdl.List_        depth=1
							//    1 2 3                                                               int values       depth=0
							// string() len(l)  2 *ast.InputValue_  ast.List_ 0
							// string() len(l)  3 *ast.InputValue_  ast.Int_ 0
							// string() len(l)  3 *ast.InputValue_  ast.Int_ 1
							// string() len(l)  3 *ast.InputValue_  ast.Int_ 2
							// string() len(l)  2 *ast.InputValue_  ast.List_ 1
							// string() len(l)  4 *ast.InputValue_  ast.Int_ 0
							// string() len(l)  4 *ast.InputValue_  ast.Int_ 1
							// string() len(l)  4 *ast.InputValue_  ast.Int_ 2
							// string() len(l)  4 *ast.InputValue_  ast.Int_ 3
							// [2]x
							// x[0] -> s[3] -> scalar
							// x[1] -> s[4] -> scalar
							// root type should be List_

							if rootFld.Type.Depth == 0 {
								p.addErr(fmt.Sprintf(`Expected a single value for "%s" , response returned a List  %s`, rootFld.Name, qry.Name.AtPosition()))
							}

							var f func(y sdl.List_, d int)
							// f will output sdl.List_ for any level of nesting
							// d is the nesting depth of List_
							f = func(y sdl.List_, d int) {

								for i := 0; i < len(y); i++ {
									if x, ok := y[i].InputValueProvider.(sdl.List_); ok {
										writeout(fieldPath, out, "[ ", noNewLine)
										d++ // nesting depth of List_
										if d > rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Exceeds nesting of List type for "%s" %s`, qry.Name, qry.Name.AtPosition()))
										}
										f(x, d)
										writeout(fieldPath, out, "] ", noNewLine)
										d--
									} else {
										if d < rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Expect a nesting level of %d, got %d, for scalar values in List for "%s" %s`, rootFld.Type.Depth, d, qry.Name, qry.Name.AtPosition()))
										}
										// optimise by performing loop here rather than use outer for loop
										for i := 0; i < len(y); i++ {
											// for scalar only Type.Name contains the scalar type name i.e. Int, Float, Boolean etc
											if y[i].IsType().String() != rootFld.Type.Name.String() {
												if _, ok := y[i].InputValueProvider.(sdl.Null_); !ok {
													p.addErr(fmt.Sprintf(`Expected "%s" got %s for "%s" %s`, rootFld.Type.Name_.String(), y[i].IsType(), qry.Name, qry.Name.AtPosition()))
												} else {
													var bit byte = 1
													bit &= rootFld.Type.Constraint >> uint(d)
													if bit == 1 {
														p.addErr(fmt.Sprintf(`Expected non-null got null for "%s" %s`, qry.Name, qry.Name.AtPosition()))
													}
												}
											}
											writeout(fieldPath, out, y[i].String(), noNewLine)
										}
										break
									}
								}
							}
							fmt.Println("List - scalar.....")
							writeout(fieldPath, out, "[ ", noNewLine)
							f(riv, 1)
							writeout(fieldPath, out, "] ", noNewLine)
							fmt.Println("List - scalar.....finished ")

						case sdl.String_:
							// TODO: remove this case - using "null" to represent null value in response string
							var bit byte = 1
							if rootFld.Type.Name.String() != riv.IsType().String() {
								p.addErr(fmt.Sprintf(`2 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
								return
							}
							bit &= rootFld.Type.Constraint
							if bit == 1 && riv.String() == "null" {
								p.addErr(fmt.Sprintf(`Cannot be null for %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
							}
							if !(rootFld.Type.Name_.String() == sdl.STRING.String() || rootFld.Type.Name_.String() == sdl.RAWSTRING.String()) {
								p.addErr(fmt.Sprintf(`3 Expected String got %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
							}
							s_ := string(`"` + riv.String() + `"`)
							writeout(fieldPath, out, s_, noNewLine)

						case sdl.RawString_:
							if rootFld.Type.Name.String() != riv.IsType().String() {
								p.addErr(fmt.Sprintf(`4 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
								return
							}
							s_ := string(`"""` + riv.String() + `"""`)
							writeout(fieldPath, out, s_, noNewLine)

						case sdl.Null_:
							if rootFld.Type.Name.String() != riv.IsType().String() {
								p.addErr(fmt.Sprintf(`5 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
								return
							}
							var bit byte = 1
							bit &= rootFld.Type.Constraint
							if bit == 1 {
								p.addErr(fmt.Sprintf(`Value cannot be null %s %s`, rootFld.Type.Name_.String(), qry.Name.AtPosition()))
							}

						default:
							if rootFld.Type.Name.String() != riv.IsType().String() {
								p.addErr(fmt.Sprintf(`6 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
							} else {
								s_ := resp.String()
								writeout(fieldPath, out, s_, noNewLine)
							}
						}
						//
						// only field for an Input type must be present if not-null constraint enabled. Normal query field may or may not be present
						//
						// if !foundResp {
						// 	fmt.Println("NOT FOUND ", qry.Name)
						// 	var bit byte = '1'
						// 	bit &= rootFld.Type.Constraint
						// 	fmt.Printf("No field value bit: %08b Depth: %d \n", bit, rootFld.Type.Depth)
						// 	if bit == 1 {
						// 		p.addErr(fmt.Sprintf(`Expected %s Value, resolver returned no result for "%s" %s`, rootFld.Type.Name_.String(), qry.Name, qry.Name.AtPosition()))
						// 	}
						// }

					} else {
						//  --- DDD ---
						//
						// scalar Resolver exists
						//
						// find relevant response field associated with current query/root field
						//
						var resp sdl.InputValueProvider
						switch y := responseItems.(type) {
						case sdl.ObjectVals:
							for _, response := range y {
								// find response field by matching name against root field and grab the associated response field data.
								if response.Name.EqualString(rootFld.Name_.String()) { // name
									resp = response.Value.InputValueProvider
								}
							}
						}
						if resp == nil {
							p.addErr(fmt.Sprintf("No corresponding root field found from response "))
							p.abort = true
							return
						}
						// execute resolver using response data for field
						fmt.Println("Input to resolver:", resp.String())
						response = qry.Resolver(resp, qry.Arguments)
						fmt.Println("Response >>>>> ", response)

						errCnt := len(p.perror)
						// generate  AST from response JSON { name: value name: value ... }
						l := lex.New(response)
						p2 := pse.New(l)
						responseItems := p2.ParseResponse() // similar to sdl.parseArguments
						fmt.Println("Response >>>>> ", responseItems)
						if len(p2.Getperror()) > 0 {
							// error in parsing stmt from db - this should not happen as only valid stmts are saved.
							p.perror = append(p.perror, p2.Getperror()...)
						}
						writeout(pathRoot, out, fieldName)
						writeout(pathRoot, out, ":", noNewLine)
						fmt.Printf("+++ newRoot %T\n", root)
						//
						switch r := responseItems.(type) {
						case sdl.ObjectVals:
							// developer wraps resolver output in { name: value } where name is query field name e.g. age
							for _, response := range r {
								//
								// find response field by matching name against root field and  grab the associated response field data.
								//
								if response.Name.EqualString(rootFld.Name_.String()) { // name
									resp = response.Value.InputValueProvider
									break
								}
							}
						default:
							// developers does not wrap resolver output
							resp = r
						}
						switch riv := resp.(type) {

						case sdl.List_: // type List_ []*InputValue_ - respresents many sdl.ObjectVals
							//
							// does response match expected root type
							//
							var f func(y sdl.List_, d int)
							// f will output sdl.List_ for any level of nesting
							// d is the nesting depth of List_
							f = func(y sdl.List_, d int) {

								for i := 0; i < len(y); i++ {
									if x, ok := y[i].InputValueProvider.(sdl.List_); ok {
										writeout(fieldPath, out, "[ ", noNewLine)
										d++ // nesting depth of List_
										if d > rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Exceeds nesting of List type for "%s" %s`, qry.Name, qry.Name.AtPosition()))
										}
										f(x, d)
										writeout(fieldPath, out, "] ", noNewLine)
										d--
									} else {
										if d < rootFld.Type.Depth {
											p.addErr(fmt.Sprintf(`Expect a nesting level of %d, got %d, for scalar values in List for "%s" %s`, rootFld.Type.Depth, d, qry.Name, qry.Name.AtPosition()))
										}
										// optimise by performing loop here rather than use outer for loop
										for i := 0; i < len(y); i++ {
											// for scalar only Type.Name contains the scalar type name i.e. Int, Float, Boolean etc
											if y[i].IsType().String() != rootFld.Type.Name.String() {
												if _, ok := y[i].InputValueProvider.(sdl.Null_); !ok {
													p.addErr(fmt.Sprintf(`Expected "%s" got %s for "%s" %s`, rootFld.Type.Name_.String(), y[i].IsType(), qry.Name, qry.Name.AtPosition()))
												} else {
													var bit byte = 1
													bit &= rootFld.Type.Constraint >> uint(d)
													if bit == 1 {
														p.addErr(fmt.Sprintf(`Expected non-null got null for "%s" %s`, qry.Name, qry.Name.AtPosition()))
													}
												}
											}
											writeout(fieldPath, out, y[i].String(), noNewLine)
										}
										break
									}
								}
							}
							writeout(fieldPath, out, "[ ", noNewLine)
							fmt.Println("List_ type")
							f(riv, 1)
							writeout(fieldPath, out, "] ", noNewLine)

						// sdl.ObjectVals - represents Objects which is not appropriate in the scalar section
						//
						// case sdl.ObjectVals: // type ArgumentS []*ArgumentT  -  represents object with fields
						// 	fmt.Println("Reponse is a single object")
						// 	writeout(fieldPath, out, "{", noNewLine)

						// 	for _, v := range riv {
						// 		fmt.Println("v.Value.InputValueProvider ", v.Value.InputValueProvider.IsType(), fieldPath, root)
						// 		p.executeStmt_(root, qry.SelectionSet, fieldPath, v.Value.InputValueProvider, out)
						// 	}

						// 	writeout(fieldPath, out, "}", noNewLine)
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
	fmt.Println("XXXCCCCCCCCCCECEEGEGEGEEJEHLJGHLKJHLKJHLKJHLKJHLKJHLKJHLKJHLKJHLKJHLKJHLKJKLJ")
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
		//typedef ast.TypeFlag_ // token defines SCALAR types only. All other types will be populated in astType map.
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
