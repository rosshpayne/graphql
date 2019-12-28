package parser

import (
	"context"
	"errors"
	"fmt"
	_ "os"
	"strconv"
	"strings"
	"sync"
	"time"

	sdl "github.com/graph-sdl/ast"
	"github.com/graph-sdl/db"
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
	//
	ResolverTimeoutMS = 800
	// operation types
	QUERY        = `query`
	MUTATION     = `mutation`
	SUBSCRIPTION = `subscription`
)

type Argument struct {
	Name  string
	Value string
}

type stmtType string

type (
	parseFn func(op string) ast.StatementDef

	Parser struct {
		l        *lexer.Lexer
		document string
		xStmt    string // stmt name to be executed

		extend bool
		abort  bool
		// schema rootAST

		curToken  *token.Token
		peekToken *token.Token

		tyCache   *pse.Cache_
		stmtCache *Cache_

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
	FragmentStmts  map[sdl.NameValue_]*ast.FragmentStmt
	OperationStmts map[sdl.NameValue_]*ast.OperationStmt
	noName         string = "__NONAME__"
)

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	// GL type cache
	p.tyCache = pse.NewCache()
	// GL statement cache
	p.stmtCache = NewCache()
	// cache for resolver functions
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

// astsitory of all types defined in the graph

func init() {
	FragmentStmts = make(map[sdl.NameValue_]*ast.FragmentStmt)
	OperationStmts = make(map[sdl.NameValue_]*ast.OperationStmt)
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
	if p.curToken != nil && p.curToken.Illegal {
		//if p.curToken.Illegal {
		p.addErr(fmt.Sprintf("Illegal %s token, [%s]", p.curToken.Type, p.curToken.Literal))
	}
	// if $variable present then mark the identier as a VALUE
	if p.curToken != nil && p.curToken.Literal == token.DOLLAR {
		//if p.curToken.Literal == token.DOLLAR {
		p.peekToken.Cat = token.VALUE
	}
}

// ==================== Start =========================

func (p *Parser) ParseDocument(doc ...string) (*ast.Document, []error) {
	api := &ast.Document{}
	//	api.Statements = []ast.Statement{} // contains operational stmts (query, mutation, subscriptions) and fragment stmts
	//
	// preparation - get Schema ast from db
	//
	var (
		schemaAST sdl.GQLTypeProvider
		SrootAST  sdl.GQLTypeProvider
		MrootAST  sdl.GQLTypeProvider
		QrootAST  sdl.GQLTypeProvider
		schema    *sdl.Schema_
		allErrors []error
		err       error
	)
	//
	// set document
	//
	if len(p.document) == 0 {
		p.document = defaultDoc
	}
	db.SetDefaultDoc(p.document)
	if len(doc) == 0 {
		db.SetDocument(p.document)
	} else {
		p.document = doc[0]
		db.SetDocument(doc[0])
	}
	//
	// fetch schema within the document
	//
	schemaAST, err = p.tyCache.FetchAST(sdl.NameValue_("schema"))
	if err != nil {
		p.addErr(err.Error())
	}
	if schemaAST == nil {
		p.addErr("Abort. There is no schema defined")
		return nil, p.perror
	}
	schema = schemaAST.(*sdl.Schema_)
	//
	// Phase 1: parse all statements (query, fragment) in the document and add to cache if statement has no errors
	//
	var failed bool
	for p.curToken.Type != token.EOF {
		//
		var stmt *ast.Statement
		stmtAST, stmtType := p.parseStatement()
		if stmtAST == nil {
			return nil, p.perror
		}

		if stmtAST != nil {
			stmt = &ast.Statement{Type: stmtType, AST: stmtAST, Name: string(stmtAST.StmtName())}
			api.Statements = append(api.Statements, stmt)
		} else {
			stmt = &ast.Statement{Type: stmtType, AST: nil, Name: string(stmtAST.StmtName())}
			api.Statements = append(api.Statements, stmt)
			failed = true
		}

		if stmtAST != nil {
			p.stmtCache.AddEntry(stmt.AST.StmtName(), stmt.AST) //	ast.Add2StmtCache(stmt.AST.StmtName(), stmt.AST)
		}
		allErrors = append(allErrors, p.perror...)
		p.perror = nil
	}
	//
	if failed {
		return nil, allErrors
	}
	for _, stmt := range api.Statements {
		if stmt.Type == "fragment" {
			continue
		}
		stmtAST := stmt.AST.(*ast.OperationStmt)
		switch stmtAST.Type {
		case "query":
			// get query rootAST
			if QrootAST == nil {
				QrootAST, err = p.tyCache.FetchAST(schema.Query.Name)
				if err != nil {
					p.addErr(err.Error())
				}
				if QrootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Query))
					return nil, p.perror
				}
			}
			stmt.RootAST = QrootAST
		case "mutation":
			// get mutation rootAST
			if MrootAST == nil {
				MrootAST, err = p.tyCache.FetchAST(schema.Mutation.Name)
				if err != nil {
					p.addErr(err.Error())
				}
				if MrootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Mutation))
					return nil, p.perror
				}
			}
			stmt.RootAST = MrootAST
		case "subscription":
			// get subscription rootAST
			if SrootAST == nil {
				SrootAST, err = p.tyCache.FetchAST(schema.Subscription.Name)
				if err != nil {
					p.addErr(err.Error())
				}
				if SrootAST == nil {
					p.addErr(fmt.Sprintf(`query root "%s" does not exist`, schema.Subscription))
					return nil, p.perror
				}
			}
			stmt.RootAST = SrootAST
		}
	}
	allErrors = append(allErrors, p.perror...)
	p.perror = nil
	//
	// phase 2  - check statment names
	//
	if len(OperationStmts) > 1 {
		var (
			nm    string
			short int
		)
		// look for shortened version of statments
		for i := 0; ; i++ {
			if i == 0 {
				nm = noName
			} else {
				nm = noName + "/" + strconv.Itoa(i)
			}
			if _, ok := OperationStmts[sdl.NameValue_(nm)]; ok {
				short++
			} else {
				break
			}
		}
		if short > 0 {
			p.addErr(fmt.Sprintf(" %d shorted stmt detected. Shortened operation statment not allowed when more than one statement exists in document.. Please provide all statements with names", short))
		}
	}
	// Note: statement name duplicates are handled during parsing of the statement
	//
	//
	// phase 3a: validate any fragment stmt - resolve ALL types. Once complete all type's AST will reside in the cache
	//                  			  and  *Type.AST assigned where applicable
	//
	for _, stmt := range api.Statements {
		if stmt.Type != "fragment" {
			continue
		}
		x, ok := stmt.AST.(*ast.FragmentStmt)
		if !ok {
			continue
		}
		// execute fragment statements first
		// generic checks
		p.resolveAllTypes(stmt.AST, p.tyCache)
		if p.hasError() {
			return nil, p.perror
		}
		// check all fields belong to their respective root type & check for duplicate fields
		p.checkFields(nil, stmt.AST)
		x.CheckOnCondType(&p.perror, p.tyCache)
		x.CheckIsInputType(&p.perror)
		//
		// add to cache
		//
		if len(p.perror) == 0 {
			p.stmtCache.AddEntry(stmt.AST.StmtName(), stmt.AST) // ast.Add2StmtCache(stmt.AST.StmtName(), stmt.AST)
		} else {
			failed = true
		}
		allErrors = append(allErrors, p.perror...)
		p.perror = nil
	}
	//
	// phase 3b: validate operational stmts
	//
	if failed {
		return nil, allErrors
	}
	for _, stmt := range api.Statements {
		if stmt.Type == "fragment" {
			continue
		}
		// execute fragment statements first
		// generic checks
		p.resolveAllTypes(stmt.AST, p.tyCache)
		if p.hasError() {
			return nil, p.perror
		}
		// check all fields belong to their respective root type & check for duplicate fields
		p.checkFields(stmt.RootAST, stmt.AST)
		// type specific checks
		stmt.AST.CheckIsInputType(&p.perror)
		stmt.AST.CheckInputValueType(&p.perror)
		//
		// add to cache
		//
		if len(p.perror) == 0 {
			p.stmtCache.AddEntry(stmt.AST.StmtName(), stmt.AST) //ast.Add2StmtCache(stmt.AST.StmtName(), stmt.AST)
		} else {
			failed = true
		}
		allErrors = append(allErrors, p.perror...)
		p.perror = nil

	}
	allErrors = append(allErrors, p.perror...)
	if failed {
		return nil, allErrors
	}
	//
	// Execute phase
	//
	var executed bool
	for _, stmt := range api.Statements {
		if stmt.Type == "fragment" {
			continue
		}
		if len(p.xStmt) > 0 && stmt.Name != p.xStmt {
			continue
		}
		//p.executeStmt(stmt.RootAST, stmt.AST)
		p.executeStmt(stmt)
		executed = true
		allErrors = append(allErrors, p.perror...)
	}
	if !executed {
		p.addErr(fmt.Sprintf(`Statement "%s" not found`, p.xStmt))
	}
	allErrors = append(allErrors, p.perror...)

	return api, allErrors
}

// ==================== End  =========================

var opt bool = true // is optional
func (p *Parser) parseStatement() (ast.StatementDef, string) {
	var (
		stmtType string
	)
	tokType := p.curToken.Type
	switch tokType {
	case token.QUERY, token.MUTATION, token.SUBSCRIPTION, token.FRAGMENT:
		stmtType = p.curToken.Literal
	default:
		// presume shorthand form of operation
		tokType = token.QUERY
		stmtType = QUERY
	}
	if f, ok := p.parseFns[tokType]; ok {
		return f(stmtType), stmtType
	}
	p.addErr(fmt.Sprintf(`Non QL statement detected, "%s" at line: %d column: %d. Aborted`, stmtType, p.l.Line, p.l.Col))
	return nil, ""
}

func (p *Parser) SetDocument(doc string) error {
	p.document = doc
	//TODO check document exists in db
	return nil
}

func (p *Parser) SetExecStmt(xStmt string) error {
	p.xStmt = xStmt
	return nil
}

// ===================  resolveAllTypes  ==========================
// resolveAllTypes in the couple of cases where types are explicitly defined in operation statements (query,mutation,subscription)
// It is also in the selectionset that objects are sourced and resolved.
// Once resolved we have the AST of all types referenced to in the operational & fragment (non-type) statements saved in the ql-cache
//
func (p *Parser) resolveAllTypes(stmt ast.StatementDef, t *pse.Cache_) {
	//returns slice of unresolved types from the statement passed in
	unresolved := make(sdl.UnresolvedMap)
	stmt.SolicitNonScalarTypes(unresolved)

	resolved := make(ast.UnresolvedMap)
	for tyName := range unresolved {
		if _, ok := t.Cache[tyName.String()]; ok {
			resolved[tyName] = nil
			//delete(unresolved, tyName)
		}
	}
	//  unresolved should only contain non-scalar types known upto that point.
	for tyName, ty := range unresolved { // unresolvedMap: [name]*Type
		// resolve type
		ast_, err := p.tyCache.FetchAST(tyName.Name)
		// type ENUM values will have nil *Type
		if ast_ != nil {
			if ty != nil {
				// purpose of resolving type is to assign the AST to the field's type. We can now bypass the cache for all field types.
				ty.AST = ast_
				// if not scalar then check for unresolved types in nested type
				if !ty.IsScalar() {
					if _, ok := resolved[tyName]; !ok {
						// recursively resolve the type only not already resolved
						p.resolveNestedType(ast_, t)
					}

				}
			}

		} else {
			// nil ast_ means not found in db
			if err != nil {
				p.addErr(err.Error())
			}
			p.addErr(fmt.Sprintf(`Type "%s" does not exist %s`, tyName, tyName.AtPosition()))
		}
	}

}

func (p *Parser) resolveNestedType(v sdl.GQLTypeProvider, t *pse.Cache_) {
	//returns slice of unresolved types from the statement passed in
	unresolved := make(sdl.UnresolvedMap)

	v.SolicitNonScalarTypes(unresolved)

	for tyName := range unresolved {
		if _, ok := t.Cache[tyName.String()]; ok {
			delete(unresolved, tyName)
		}
	}

	//  unresolved should only contain non-scalar types known upto that point.
	for tyName, ty := range unresolved { // unresolvedMap: [name]*Type
		ast_, err := p.tyCache.FetchAST(tyName.Name)
		if err != nil {
			p.addErr(err.Error())
		}
		// resolve type type ENUM values will have nil *Type
		if ast_ != nil {
			if ty != nil {
				ty.AST = ast_
				// if not scalar then check for unresolved types in nested type
				if !ty.IsScalar() {
					p.resolveNestedType(ast_, t)
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

// ================== checkFields ======================================

func (p *Parser) checkFields(root sdl.GQLTypeProvider, stmt_ ast.StatementDef) {

	p.responseMap = make(map[string]*sdl.InputValueProvider) // map[sdl.NameValue_]map[sdl.NameValue_]sdl.GQLTypeProvider

	switch stmt := stmt_.(type) {

	case *ast.OperationStmt:
		// only for operational Query
		if stmt.Type != "query" {
			return
		}
		fmt.Println("operationstmt ", len(stmt.SelectionSet))
		for i, k := range stmt.SelectionSet {
			fmt.Printf("ss %d type %T\n", i, k)
		}
		p.checkFields_(root, stmt.SelectionSet, string(root.TypeName()))

	case *ast.FragmentStmt:
		fmt.Println("Fragmentnstmt")
		var err error
		root, err = p.tyCache.FetchAST(stmt.TypeCond.Name)
		if err != nil {
			p.addErr(err.Error())
		}
		p.checkFields_(root, stmt.SelectionSet, string(root.TypeName()))
	}
	// if len(p.perror) == before {
	// 	p.ResponseMap = make(ResponseMapT)
	// 	for k := range p.responseMap {
	// 		p.ReponseMap[k] = ast.ResponseValue{}
	// 	}
	// }
	// p.responseMap = nil
}

func (p *Parser) checkFields_(root sdl.GQLTypeProvider, set []ast.SelectionSetProvider, pathRoot string) {
	// ty_ (object):  type Query { allPersons(last : Int ) : [Person!]! }	<== root
	//
	// 	stmt:	query XYZ {
	//      allPersons(last: 2) {											<== set
	//          name
	//          age
	//      }
	// }
	// var (
	// 	rootObj *sdl.Object_
	// )
	if root == nil {
		p.addErr("In checkFields_, passed in a root of nil")
		return
	}
	fmt.Println("********************************************************* checkFields_ *********************************************** ")
	//	rootObj = root.(*sdl.Object_) // 	type Query { allPersons(last: Int): [Person!]	}
	// rootObj := root.(sdl.SelectionGetter) - moved 8 lines down
	for _, qryFld := range set {
		fmt.Printf("*** *** checkFields: qryFld %T\n", qryFld)

		switch qry := qryFld.(type) { // allPersons(last:3)

		case *ast.Field:
			var (
				newRoot sdl.GQLTypeProvider
				found   bool
				rootFld *sdl.Field_
			)
			rootObj := root.(sdl.SelectionGetter)
			fmt.Printf("**** qryFld %s\n", qry.Name)
			//
			// Confirm argument value type against type definition
			//
			for _, rootFld = range rootObj.GetSelectionSet() { // root object nested fields

				// find matching root type
				//
				fmt.Println("+++ qry.Name.Equals(rootFld.Name_)", qry.Name, rootFld.Name_)
				if !qry.Name.Equals(rootFld.Name_) {
					continue
				}
				fmt.Println("+++++++ found root ++++++++++++")
				found = true
				//
				// found field, now validate argument inputs
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
						p.addErr(fmt.Sprintf(`Field argument "%s" is not defined in type "%s", %s`, argVal.Name_, root.TypeName(), argVal.Name_.AtPosition()))
						p.abort = true
					}
				}
				//
				// check the Field "Type.AST" is populated. Better to access the AST through the Type metadata rather than the type-cache which is a shared resource.
				// Parsing should populate all the type metadata, but it may be miss the AST depending on order of the type processing.
				//
				if !rootFld.Type.IsScalar() && rootFld.Type.AST == nil {
					var err error
					rootFld.Type.AST, err = p.tyCache.FetchAST(rootFld.Type.Name)
					if err != nil {
						p.addErr(err.Error())
					}
					if rootFld.Type.AST == nil {
						panic(fmt.Sprintf("Type %s not found", rootFld.Type.Name))
					}
				}
				//
				// determine if matching root type is an object based type
				//
				switch x := rootFld.Type.AST.(type) {
				case *sdl.Object_:
					fmt.Println("matching root is an object")
					newRoot = x
					fmt.Println("************** use newroot for new object type ", len(qry.SelectionSet), pathRoot)
					var fieldPath string
					fieldPath = pathRoot + "/" + qry.GenNameAliasPath()
					fmt.Println("************** fieldPath", fieldPath, newRoot.TypeName())
					p.respOrder = append(p.respOrder, fieldPath)
					p.tyCache.FetchAST(sdl.NameValue_(newRoot.TypeName()))
					p.checkFields_(newRoot, qry.SelectionSet, fieldPath)
				case *sdl.Interface_:
					fmt.Println("matching root is an interface")
					newRoot = x
					fmt.Println("************** use newroot for new object type ", len(qry.SelectionSet), pathRoot)
					var fieldPath string
					fieldPath = pathRoot + "/" + qry.GenNameAliasPath()
					fmt.Println("************** fieldPath", fieldPath, newRoot.TypeName())
					p.respOrder = append(p.respOrder, fieldPath)
					p.tyCache.FetchAST(sdl.NameValue_(newRoot.TypeName()))
					p.checkFields_(newRoot, qry.SelectionSet, fieldPath)
				case *sdl.Union_:
					fmt.Println("matching root is a Union")
					fmt.Printf("*** *** checkFields: qryFld %T #selectionSet %d\n", qryFld, len(qry.SelectionSet))
					newRoot = x
					fmt.Println("************** use newroot for new object type ", len(qry.SelectionSet), pathRoot)
					var fieldPath string
					fieldPath = pathRoot + "/" + qry.GenNameAliasPath()
					fmt.Println("************** fieldPath", fieldPath, newRoot.TypeName())
					p.respOrder = append(p.respOrder, fieldPath)
					p.tyCache.FetchAST(sdl.NameValue_(newRoot.TypeName()))
					p.checkFields_(newRoot, qry.SelectionSet, fieldPath)
				default:
					fmt.Println("matching root is not an object or interface ")
				}
				//	qry.Type = rootFld.Type // assign sdl Type to *ast.Field
				qry.ParentObj = root
				qry.ParentFld = rootFld

				if !(newRoot != nil && len(qry.SelectionSet) != 0) {
					//
					// scalar field - append to response map
					//
					fmt.Println("scalar field - append to response map ")
					var fieldPath strings.Builder
					fieldPath.WriteString(pathRoot)
					fieldPath.WriteString("/")
					fieldPath.WriteString(qry.Name.String())
					if qry.Alias.Exists() {
						fieldPath.WriteString("(")
						fieldPath.WriteString(qry.Alias.String())
						fieldPath.WriteString(")")
					}
					//	qryFldMap[fieldPath] = rootFld
					fmt.Println("Scalar fieldPath: ", fieldPath.String())
					if _, ok := p.responseMap[fieldPath.String()]; ok {
						p.addErr(fmt.Sprintf(`Field "%s.%s" has already been specified %s`, root.TypeName(), fieldPath.String(), qry.Name.AtPosition()))
					} else {
						p.responseMap[fieldPath.String()] = nil
						p.respOrder = append(p.respOrder, fieldPath.String())
					}
				}
				break
			}
			if !found {
				p.addErr(fmt.Sprintf(`Field "%s" is not in %s "%s" %s`, qry.Name, root.Type(), root.TypeName(), qry.Name.AtPosition()))
				// p.abort = true
				// return
			}

		case *ast.FragmentSpread:
			// fragment spread would have its associated fragment statement checked in phase 2a so can ignore further field checks.

			fmt.Println("checkFields_ : for Fragment Spread - qry.Name ", qry.Name)
			stmtAST := p.stmtCache.FetchAST(ast.StmtName_(qry.Name.String()))
			if stmtAST == nil {
				p.addErr(fmt.Sprintf(`Associated Fragment definition "%s" not found in document %s`, qry.Name, qry.Name_.AtPosition()))
				p.abort = true
				return
			} else {
				if x, ok := stmtAST.(*ast.FragmentStmt); ok {
					qry.FragStmt = x
					fmt.Println("Assigned FRG stmt to fragment spread ...........................")
				} else {
					p.addErr(fmt.Sprintf(`Expected a Fragment Statment from cache during check-field operation  %s`, qry.Name_.AtPosition()))
				}
			}

			// if qry.Fragdef, ok = stmtAST.(ast.FragmentDef); !ok {
			// 	p.addErr(fmt.Sprintf(`Fragment spread "%s" not a FragmentDef type `, qry.Name))
			// 	p.abort = true
			// 	return
			// }
			// p.checkFields_(root, qry.Fragdef.GetSelectionSet(), pathRoot)

		case *ast.InlineFragment:

			// TODO - create inline fragment go tests with type condition and without a type condition
			rootFrag := root
			rootPath := pathRoot
			fmt.Println("checkFields_ : for Inline fragment Spread - qry.Name ", qry.TypeCond)
			if _, ok := rootFrag.(*sdl.Union_); ok {
				if !qry.TypeCond.Exists() {
					p.addErr(fmt.Sprintf(`As root type, "%s"is a union inline fragment must have an on clause`, rootFrag.TypeName()))
					return
				}
			}
			if !qry.TypeCond.Exists() {
				// base type cond on query field's parent  type
				//qry.TypeCond = sdl.Name_{Name: sdl.NameValue_(root.TypeName())}
				qry.TypeCondAST = rootFrag
				qry.TypeCond = sdl.Name_{Name: sdl.NameValue_(rootFrag.TypeName())}
			}
			// check cond type is appropriate i.e object, interface, union
			qry.CheckOnCondType(&p.perror, p.tyCache)

			if x, ok := rootFrag.(*sdl.Union_); ok {
				// check type cond satisifies union
				var found bool
				for _, v := range x.NameS {
					if v.Equals(qry.TypeCond) {
						found = true
					}
				}
				if !found {
					p.addErr(fmt.Sprintf(`On condition type not a member of union type, "%s"`, rootFrag.TypeName()))
					return
				}
				rootFrag = qry.TypeCondAST
			} else {
				rootFrag = qry.TypeCondAST
			}
			if !qry.TypeCond.Exists() {
				rootPath += "/" + string(root.TypeName())
			} else {
				rootPath += "/" + qry.TypeCond.Name.String()
			}

			if len(qry.Directives) == 0 {

				fmt.Println("checkFields: before checkFields ", rootFrag.TypeName(), len(qry.SelectionSet), rootPath)

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
							// parse wil have populated argument value with variable value. //TODO - not working when no default value specified
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
		} // switch
	}
	fmt.Println("++++++++++++++++++++++++++++++++++++++++++++ leave  checkFields_ --------------------------------------* ")

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

//func (p *Parser) executeStmt(stmt_ ast.StatementDef) {
func (p *Parser) executeStmt(stmt_ *ast.Statement) {

	var (
		stmt *ast.OperationStmt
		ok   bool
		out  []strings.Builder
		wg   sync.WaitGroup
	)
	var ()
	if stmt, ok = stmt_.AST.(*ast.OperationStmt); !ok {
		p.addErr(fmt.Sprintf("Expected an OperationStmt in execute phase. Aborting. "))
		return
	}
	// only for operational Query
	if stmt.Type != "query" {
		p.addErr(fmt.Sprintf("Expected an Query OperationStmt in execute phase. Aborting. "))
		return
	}
	wg.Add(len(stmt.SelectionSet))
	out = make([]strings.Builder, len(stmt.SelectionSet))
	root := stmt_.RootAST
	//
	// execute all stmt root fields concurrently
	//
	for i, opFld := range stmt.SelectionSet {
		opFld := opFld
		go p.executeStmtOp(root, opFld, string(root.TypeName()), nil, &out[i], &wg)
	}

	wg.Wait()
	//
	// combine all stmt outputs
	//
	var ts strings.Builder
	fmt.Println("==== output ====== ")
	if len(stmt.SelectionSet) > 1 {
		ts.WriteString(" { data : [ ")
	} else {
		ts.WriteString("\n{\ndata: {")
	}

	for i, _ := range stmt.SelectionSet {
		ts.WriteString(out[i].String())
	}
	if len(stmt.SelectionSet) > 1 {
		ts.WriteString(" \n ] } ")
	} else {
		ts.WriteString("\n}\n}")
	}
	println(ts.String())

}

func (p *Parser) executeStmtOp(root sdl.GQLTypeProvider, qryFld ast.SelectionSetProvider, pathRoot string, responseItems sdl.InputValueProvider, out *strings.Builder, wg *sync.WaitGroup) {
	var set = []ast.SelectionSetProvider{qryFld}
	var responseType string = ""
	p.executeStmt_(root, set, pathRoot, responseType, responseItems, out)
	wg.Done()
}

func (p *Parser) executeStmt_(root sdl.GQLTypeProvider, set []ast.SelectionSetProvider, pathRoot string, responseType string, responseItems sdl.InputValueProvider, out *strings.Builder) { //type ObjectVals []*ArgumentT - serialized object
	//func (p *Parser) executeStmt_(root sdl.FieldSetter, set []ast.SelectionSetProvider, pathRoot string, responseItems sdl.InputValueProvider, out *strings.Builder) { //type ObjectVals []*ArgumentT - serialized object
	//
	// ty_ (object):  type Query { allPersons(last : Int ) : [Person!]! }	<== root type, Person.  Defines the type associated with the query field.
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
	//										** High level description **
	//
	// loop on query fields (selectionset passed in - initially "query stmt" selectionset, usually one field)
	//
	//	 for root field associated with query field
	//
	//		field type is Object -- AAA ----
	//
	//        field has no resolver
	//			loop on response fields (from previous resolver call)
	//            on match by name (query/root-type/data is now matched)
	//             get resp field value (data)
	//				 validate response for current field
	//					recusively executeStmt (passing field response), ultimately executing scalar type and any resolvers on the way
	//
	//
	//  	  field has resolver --- BBB ---
	//			match field to response data if response from previous execute exists (will not on first initial execution)
	//              assign response data to "resp" argument (as field is an object.) TODO - make input object
	//          ** execute resolver, generating JSON
	//          generate AST (from JSON) using SDL parser
	//			validate response for current field type
	//         	recusively executeStmt (passing field response), ultimately executing scalar type and any resolvers on the way
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
	//          ** execute resolver, generating JSON
	//          generate AST (from JSON) using SDL parser
	//			validate response for current field type
	//			output query JSON for field (as either List or single field)
	//
	//  on inline-Fragment type...
	//  on fragmentspread type...
	//

	if p.hasError() {
		return
	}

	for _, qryFld := range set {

		switch qry := qryFld.(type) {

		case *ast.Field:

			fmt.Printf("\n\n*** Query field: %#v\n", qry)
			var (
				newRoot   sdl.GQLTypeProvider
				fieldPath string
				fieldName string
				response  string
				rootFld   *sdl.Field_
				root      sdl.GQLTypeProvider
			)
			// ParentFld & ParentObj populated during CheckField
			//
			if qry.ParentFld == nil {
				err := fmt.Errorf(`ParentFld for field "%s" not assigned. Abort`, qry.Name)
				panic(err)
			}
			rootFld = qry.ParentFld

			fmt.Println("qry ")

			if qry.Alias.Exists() {
				fieldName = qry.Alias.String()
			} else {
				fieldName = qry.Name.String()
			}
			//
			// associated GraphQL type (type system)
			//
			//if rootFld.Type.Name
			switch rootFld.Type.AST.(type) {

			case *sdl.Object_, *sdl.Interface_:
				//
				//  -- AAA ----
				//
				// object field, details in AST (as it is not a scalar)
				//
				newRoot = rootFld.Type.AST // qryFld's matching type.  Scalar has AST of nil and field name defines what scalar.
				fieldPath = pathRoot + "/" + rootFld.Name_.String()

				fmt.Println("********** pathRoot, fieldPath: ", rootFld.Name_.String(), fieldPath, qry.Name)

				qry.Resolver = p.Resolver.GetFunc(fieldPath)

				if qry.Resolver == nil {
					//
					// use data from last resolver execution, passed in via argument "responseItems"
					//
					if responseItems == nil {
						p.addErr(fmt.Sprintf(`No responseItem provided. Default Resolver must have a responseItem. Field "%s" has no resolver function, %s %s`, qry.Name, root.TypeName(), qry.Name.AtPosition()))
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
							writeout(pathRoot, out, fieldName)
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
								//fmt.Println("+++++ rootFld.Type.IsType2(), riv.IsType() = ", rootFld.Type.IsType2(), riv.IsType())
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

												writeout(fieldPath, out, "{")

												p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseType, y[i].InputValueProvider, out)

												writeout(fieldPath, out, "}")
											}
											break
										}
									}
								}

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

								p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseType, riv, out)

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
						p.addErr(fmt.Sprintf(`Resolver response returned something other than name:value pairs. Got %T for  %s %s`, responseItems, root.TypeName(), qry.Name.AtPosition()))
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
					fmt.Println("In Resolver section . .....")
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
					//response := qry.Resolver(resp, qry.Arguments)
					var ctxMsg string = `Resolver for "%s" successfully returned but`
					ctx, cancel := context.WithTimeout(context.Background(), ResolverTimeoutMS*time.Millisecond)
					defer cancel()
					//
					rch := qry.Resolver(ctx, resp, qry.Arguments)
					// blocking wait
					select {
					case <-ctx.Done():
						ctxMsg = `Resolver for "%s" timed out and consequently`
					case response = <-rch:
					}

					fmt.Printf(`>>>>>>>  response: %s`, response)
					fmt.Println()
					//
					// respone returns an object that matches the root i.e. Person, Pet, Address, Business
					//
					if len(response) == 0 {
						fldNm := qry.Name
						if qry.Alias.Exists() {
							fldNm = qry.Alias
						}
						p.addErr(fmt.Sprintf(ctxMsg+` produced no content, %s\n`, fldNm, qry.Name.AtPosition()))
						p.abort = true
						return
					}
					errCnt := len(p.perror)
					//
					// generate AST from response JSON { name: value name: value ... }
					//
					l := lex.New(response)
					p2 := pse.New(l)
					respItems := p2.ParseResponse() // similar to sdl.parseArguments. Populates responseItems with parsed values from response.
					if respItems == nil {
						p.addErr(fmt.Sprintf(`Empty response from resolver for "%s" %s`, rootFld.Name, qry.Name.AtPosition()))
					}
					if len(p2.Getperror()) > 0 {
						// error in parsing stmt from db - this should not happen as only valid stmts are saved.
						p.perror = append(p.perror, p2.Getperror()...)
					}
					fmt.Printf("finished ParseResponse: %T %s\n\n", respItems, respItems)
					fmt.Println("** RootFld Type ", rootFld.Type, rootFld.Type.IsType2().String())       // [Post!] List
					fmt.Println("*** RootFld Type.IsType().String() ", rootFld.Name, newRoot.TypeName()) // Object posts Post
					fmt.Printf("*** RootFld Type.Depth %s %T %#v, %d \n", rootFld.Name_, rootFld.Type.AST, rootFld, rootFld.Type.Depth)
					//
					// *** Commented out as CheckInputValueType is nolonger suitable as response maynot match root type completely. ***
					//
					// validate response against type defined in schema statemen
					// respname_ := sdl.Name_{Name: sdl.NameValue_("response"), Loc: nil}
					// iv := sdl.InputValue_{InputValueProvider: respItems, Loc: nil}
					// iv.CheckInputValueType(rootFld.Type, respname_, &p.perror)
					// if errCnt != len(p.perror) {...
					//
					// process each reqponse item and generate output based on query fields in operational statement
					//
					// respItems - InputValueProvider						respItems = nil
					writeout(pathRoot, out, fieldName)
					writeout(pathRoot, out, ":", noNewLine)
					// find field "data" and access its data
					//
					// response in the form: {Type: data}     {name: value} e.g {data:["abc" "def"] interested only in ["abc" "def"]
					//
					if x, ok := respItems.(sdl.ObjectVals); ok {
						responseType = x[0].Name.String() // pass as argument to executeStmt_
						responseItems = x[0].Value.InputValueProvider
						fmt.Println("responseType, responseItems, ", responseType, responseItems)
					} else {
						p.addErr(fmt.Sprintf(`Response should be a {name:value}, where name repesents the datatype name`))
						p.abort = true
						return
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
						// d is the depth of the listing
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

										writeout(fieldPath, out, "{")

										p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseType, y[i].InputValueProvider, out)

										writeout(fieldPath, out, "}")
									}
									break
								}
							}
						}

						writeout(fieldPath, out, "[ ", noNewLine)
						f(resp, 1)
						writeout(fieldPath, out, " ]", noNewLine)

					case sdl.ObjectVals: // type ArgumentS []*ArgumentT  -  represents object with fields
						if rootFld.Type.Depth > 0 {
							p.addErr(fmt.Sprintf("Expected %s, got a response of name value pairs (ObjectVals) \n", rootFld.Type.IsType2().String()))
							p.abort = true
							return
						}
						fmt.Println("Response is a single object")
						writeout(pathRoot, out, qry.Name.String()+" : ")
						writeout(fieldPath, out, "{", noNewLine)

						p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseType, responseItems, out)

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

			case *sdl.Union_:
				//
				// object field, details in AST (as it is not a scalar)
				//
				newRoot = rootFld.Type.AST // qryFld's matching type.  Scalar has AST of nil and field name defines what scalar.
				fieldPath = pathRoot + "/" + rootFld.Name_.String()

				fmt.Println("********** pathRoot, fieldPath: ", rootFld.Name_.String(), rootFld.Type.Name_, fieldPath, qry.Name)

				qry.Resolver = p.Resolver.GetFunc(fieldPath)

				if qry.Resolver == nil {
					//
					// use data from last resolver execution, passed in via argument "responseItems"
					//
					if responseItems == nil {
						p.addErr(fmt.Sprintf(`No responseItem provided. Default Resolver must have a responseItem. Field "%s" has no resolver function, %s %s`, qry.Name, root.TypeName(), qry.Name.AtPosition()))
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
							responseType = respfld.Name_.String()
							responseItems = respfld.Value
							//
							// check response type is member of Union
							//
							var found bool
							newRoot_ := newRoot.(*sdl.Union_)
							for _, v := range newRoot_.NameS {
								if v.EqualString(responseType) {
									found = true
								}
							}
							if !found {
								p.addErr(fmt.Sprintf("Response type is not member of union, %s", rootFld.Type.Name_.String()))
								return
							}
							//
							// assign root to the response type
							//
							var err error
							newRoot, err = p.tyCache.FetchAST(sdl.NameValue_(responseType))
							if err != nil {
								p.addErr(err.Error())
								return
							}

							p.executeStmt_(newRoot, qry.SelectionSet, fieldPath, responseType, responseItems, out)

						}

					}
				} else {

				}

			default:
				//  --- CCC ----
				//
				// scalar (AST is nil) or List of scalar
				//
				fieldPath = pathRoot + "/" + qry.Name.String()
				fmt.Printf("xx root is a non-object response: %T   fieldPath . %s rootFld %T %s\n", responseItems, fieldPath, rootFld.Type.AST, rootFld.Type.Name)
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
					writeout(pathRoot, out, fieldName)
					writeout(pathRoot, out, ":", noNewLine)
					//
					// match response field for given qry field (root field have been matched already, so we know the type of the qry field)
					//
					var resp sdl.InputValueProvider

					switch r := responseItems.(type) {

					case sdl.ObjectVals:
						//  { name:InputValue_ name:InputValue_ ... }
						fmt.Println("** responseItems are ObjectVals")
						for _, response := range r {
							//
							// find response field by matching name against root field and  grab the associated response field data.
							//
							fmt.Println(" response.Name.EqualString(rootFld.Name_.String()) ", response.Name, rootFld.Name_.String())
							if response.Name.EqualString(rootFld.Name_.String()) { // name
								resp = response.Value.InputValueProvider
								break
							}
						}

					case *sdl.InputValue_:
						// typical response from Union member field. response name field matches inline fragment "on" clause.
						// {name : value } -> {dataType : []*ArgumentT } -> {dataType: { name:InputValue_ name: InputValue_} } ->
						switch x := r.InputValueProvider.(type) {
						case sdl.ObjectVals:
							// loop thru matching response field with root field
							for _, v := range x {
								if v.Name.EqualString(qry.Name.String()) { //rootFld.Name_.String()) { // name qry.Name.String()
									resp = v.Value.InputValueProvider
									break
								}
							}
						}
					}
					if resp == nil {
						p.addErr(fmt.Sprintf(`No corresponding root field found from response field, "%s"`, fieldName))
						p.abort = true
						return
					}
					//
					// got matching response field, now output the response
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
					// resp is InputValue_ type
					switch riv := resp.(type) { // value

					case sdl.ObjectVals:

						fmt.Println("in ObjectVals: ", riv)

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
										// for scalar only Type.Name contains the scalar type name i.e. Int, Float, Boolean etc. For ENUM and Scalar types, Name does not identify type, use BaseType, passing in the type AST.
										fmt.Println("y[i].IsType().String(), rootFld.Type.Name.String() -=-", y[i].IsType().String(), rootFld.Type.Name.String())
										if y[i].IsType().String() != rootFld.Type.Name.String() {
											if !(y[i].IsType().String() == "Enum" && sdl.BaseType(rootFld.Type.AST) == "E") {
												if _, ok := y[i].InputValueProvider.(sdl.Null_); !ok {
													p.addErr(fmt.Sprintf(`XX Expected "%s" got %s for "%s" %s`, rootFld.Type.Name_.String(), y[i].IsType(), qry.Name, qry.Name.AtPosition()))
												} else {
													var bit byte = 1
													bit &= rootFld.Type.Constraint >> uint(d)
													if bit == 1 {
														p.addErr(fmt.Sprintf(`Expected non-null got null for "%s" %s`, qry.Name, qry.Name.AtPosition()))
													}
												}
											}
										}
										fmt.Println("======================================= writeout scalar ================================", y[i].String())
										writeout(fieldPath, out, y[i].String(), noNewLine)
									}
									break
								}
							}
						}

						writeout(fieldPath, out, "[ ", noNewLine)
						f(riv, 1)
						writeout(fieldPath, out, "] ", noNewLine)

					case sdl.String_:
						// TODO: remove this case - using "null" to represent null value in response string
						var bit byte = 1
						if rootFld.Type.Name.String() != riv.IsType().String() {
							p.addErr(fmt.Sprintf(`Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
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

					case sdl.Int_:
						if rootFld.Type.Name.String() != riv.IsType().String() {
							p.addErr(fmt.Sprintf(`4 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
							return
						}
						s_ := riv.String()
						writeout(fieldPath, out, s_, noNewLine)

					case sdl.Float_:
						if rootFld.Type.Name.String() != riv.IsType().String() {
							p.addErr(fmt.Sprintf(`4 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
							return
						}
						s_ := riv.String()
						writeout(fieldPath, out, s_, noNewLine)

					default:
						if rootFld.Type.Name.String() != riv.IsType().String() {
							p.addErr(fmt.Sprintf(`6 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
						} else {
							s_ := resp.String()
							p.addErr(fmt.Sprintf(`6 Expected "%s" got %s %s`, rootFld.Type.Name_.String(), riv.IsType().String(), qry.Name.AtPosition()))
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
						fmt.Println("123 responseItems : ", y.String())
						for _, response := range y {
							// find response field by matching name against root field and grab the associated response field data.
							if response.Name.EqualString(rootFld.Name_.String()) { // name
								resp = response.Value.InputValueProvider
							}
						}
					}
					if resp == nil {
						p.addErr(fmt.Sprintf("yy No corresponding root field found from response "))
						p.abort = true
						return
					}
					// create timeout context and pass to Resolver
					ctx, cancel := context.WithTimeout(context.Background(), ResolverTimeoutMS*time.Millisecond)
					defer cancel()
					// execute resolver using response data for field
					rch := qry.Resolver(ctx, resp, qry.Arguments)
					// blocking wait
					select {
					case <-ctx.Done():
					case response = <-rch:
					}

					fmt.Println("Response >>>>> ", response)
					// generate  AST from response JSON { name: value name: value ... }
					l := lex.New(response)
					p2 := pse.New(l)
					// scope of responseItems restricted to Section --- DDD --- to hide argument responseItems
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
						// developer does not wrap resolver output
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
												p.addErr(fmt.Sprintf(`66 Expected "%s" got %s for "%s" %s`, rootFld.Type.Name_.String(), y[i].IsType(), qry.Name, qry.Name.AtPosition()))
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
						f(riv, 1)
						writeout(fieldPath, out, "] ", noNewLine)
					//
					// sdl.ObjectVals - represents Objects which is not appropriate in the scalar section
					//
					default:
						fmt.Printf(" responseItems NOT EITHER %T\n", responseItems)
					}
				}
			}
			// for object fields recursively call its fields, otherwise return
			// if newRoot != nil && len(x.SelectionSet) != 0 {
			// 	// new root object
			// 	p.executeStmt_(newRoot, x.SelectionSet, pathRoot+"/"+string(newRoot.TypeName()), responseItems, out)

			// }

		case *ast.FragmentSpread:

			// FragmentSpread
			// ...FragmentName	Directives-opt
			//  TODO - check that type of parent object of field matches the fragment typeCond
			fmt.Println("FRAGMENT SPREAD.........")
			var (
				displayFrg bool = true
				// dir        sdl.Directives_
				// sset       sdl.SelectionGetter
			)
			//
			// check include directive present
			//
			for _, d := range qry.Directives {
				//... @include(if: $expandedInfo) {
				if d.Name_.String() == "@include" {
					for _, arg := range d.Arguments {
						if arg.Name.String() != "if" {
							p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
							return
						}
						// parse wil have populated argument value with variable value.
						if argv, ok := arg.Value.InputValueProvider.(sdl.Bool_); ok {
							displayFrg = bool(argv)
						}
					}
				} else {
					fmt.Println("no @include directive")
				}
			}
			if !displayFrg {
				break // continue to next query field
			}
			//
			//  validate response against field type
			//
			respType, err := p.tyCache.FetchAST(sdl.NameValue_(responseType))
			if err != nil {
				p.addErr(err.Error())
			}
			if respType == nil {
				p.addErr(fmt.Sprintf(`Response type "%s" not defined in Graphql repository"`, responseType))
				return
			}
			respObj, ok := respType.(*sdl.Object_)
			if !ok {
				p.addErr(fmt.Sprintf(`Response type "%s" is not a Graphql Object`, responseType))
				p.abort = true
				return
			}
			//
			// confirm response type matches fragment type (expected type - expType )
			//
			expType, err := p.tyCache.FetchAST(qry.FragStmt.TypeCond.Name)
			if err != nil {
				p.addErr(err.Error())
			}
			if expType == nil {
				p.addErr(fmt.Sprintf(`Fragment typecondition "%s" not found in cache`, qry.FragStmt.TypeCond.Name))
			} else {
				//
				//Fragments cannot be specified on any input value (scalar, enumeration, or input object).
				//
				switch x := expType.(type) {

				case *sdl.Object_:
					//
					// check response Type name must match expected type name e.g. Person is the type name for a sdl.Object_
					//
					if responseType != x.TypeName().String() { //respObj.TypeName() != x.TypeName() {
						fmt.Printf(`Response type "%s" does not match Fragment type "%s" %s`, responseType, x.TypeName(), "\n")
						continue
					}
					//
					// Directives on fragment (based on Object)	// TODO create go test cases with directives
					//
					for _, d := range qry.FragStmt.Directives {
						//... @include(if: $expandedInfo) {
						if d.Name_.String() == "@include" {
							for _, arg := range d.Arguments {
								if arg.Name.String() != "if" {
									p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
									return
								}
								// parse wil have populated argument value with variable value.
								if argv, ok := arg.Value.InputValueProvider.(sdl.Bool_); ok {
									displayFrg = bool(argv)
								}
							}
						} else {
							fmt.Println("no @include directive")
						}
					}
					if displayFrg {
						//
						p.executeStmt_(root, qry.FragStmt.SelectionSet, pathRoot, responseType, responseItems, out)
					}

				case *sdl.Interface_:
					//
					// expected type to which responseType must match is an Interface. So does expected type implement the interface.
					//
					var implements bool
					for _, itf := range respObj.Implements {
						if itf.Equals(x.Name_) {
							implements = true
							fmt.Printf(`Response type "%s" does  implement interface "%s"`, responseType, x.Name_)
							fmt.Println()
							break
						}
					}
					if !implements {
						p.addErr(fmt.Sprintf(`Response type "%s" does not implement interface "%s"`, responseType, x.Name_))
						p.abort = true
						return
					}
					for _, d := range qry.FragStmt.Directives {
						//... @include(if: $expandedInfo) {
						if d.Name_.String() == "@include" {
							for _, arg := range d.Arguments {
								if arg.Name.String() != "if" {
									p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
									return
								}
								// parse wil have populated argument value with variable value.
								if argv, ok := arg.Value.InputValueProvider.(sdl.Bool_); ok {
									displayFrg = bool(argv)
								}
							}
						} else {
							fmt.Println("no @include directive")
						}
					}
					if displayFrg {
						p.executeStmt_(root, qry.FragStmt.SelectionSet, pathRoot, responseType, responseItems, out)
					}

				case *sdl.Union_:
					//TODO implement
				}
			}

		case *ast.InlineFragment:

			fmt.Println(" ... inline fragment ", qry.TypeCond.String())
			// InlineFragment
			// ...TypeCondition-opt	Directives-opt	SelectionSet-list
			//
			var rootFrag sdl.GQLTypeProvider

			rootPath := pathRoot
			//
			//  existence of type condition determines query root type (i.e. the type associated with the query field)
			//
			if !qry.TypeCond.Exists() {
				rootFrag = root
				rootPath += "/" + string(root.TypeName())
				fmt.Println("rootPath = ", rootPath)
			} else {
				rootFrag = qry.TypeCondAST // inline frag points to parent type (the root)
				if rootFrag == nil {
					var err error
					rootFrag, err = p.tyCache.FetchAST(qry.TypeCond.Name)
					if err != nil {
						p.addErr(err.Error())
						return
					}
				}
				rootPath += "/" + qry.TypeCond.Name.String()
				fmt.Println("xrootPath , rootFrag = ", rootPath, rootFrag.TypeName())
			}
			//
			// check response data {reponseType:responseItems} against the field type (determined by type condition for inline frags - see prevous stmt)
			respType, err := p.tyCache.FetchAST(sdl.NameValue_(responseType))
			if err != nil {
				p.addErr(err.Error())
			}
			if respType == nil {
				p.addErr(fmt.Sprintf(`Response type "%s" not defined in Graphql respository"`, responseType))
				return
			}
			respObj, ok := respType.(*sdl.Object_)
			if !ok {
				p.addErr(fmt.Sprintf(`Response type "%s" is not a Graphql Object`, responseType))
				p.abort = true
				return
			}
			fmt.Println("respObj: ", respObj.TypeName())
			//
			// depending on the inline frag type (its root), verify response object satisfies it. Note it is not an error if response does not match query field type, we merely ignore the field.
			//
			switch rtg := rootFrag.(type) {

			case *sdl.Interface_:
				var found bool
				// check if response object implements interface
				for _, v := range respObj.Implements {
					if v.Equals(rtg.Name_) {
						found = true
					}
				}
				if !found {
					// does not implement interface - ignore this field and proceed to next
					continue
				}

			case *sdl.Union_:
				// does response type match a union member
				var found bool
				for _, v := range rtg.NameS {
					if v.EqualString(responseType) {
						found = true
					}
				}
				if !found {
					p.addErr(fmt.Sprintf(`Response type "%s" does not match any member in the Union type %s`, responseType, rootFrag.TypeName()))
					continue
				}
				// TODO complete implementation

			default:
				if responseType != rootFrag.TypeName().String() {
					fmt.Printf(`Response type "%s" does not match Fragment type "%s" %s`, responseType, rootFrag.TypeName(), "\n")
					continue
				}
			}

			if len(qry.Directives) == 0 {

				fmt.Println("Just before executestmt: ", rootFrag.TypeName(), len(qry.SelectionSet), rootPath, responseType, responseItems)
				//p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseType, responseItems, out)
				p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseType, responseItems, out)

			} else {
				//
				// process directives
				//
				for _, v := range qry.Directives {
					switch v.Name_.String() {
					case "@include":
						//... @include(if: $expandedInfo) {
						for _, arg := range v.Arguments {
							if arg.Name.String() != "if" {
								p.addErr(fmt.Sprintf(`Expected argument name of "if", got %s %s`, arg.Name, arg.AtPosition()))
								return
							}
							// parse wil have populated argument value with variable value.
							argv := arg.Value.InputValueProvider.(sdl.Bool_)
							if argv == true {
								p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseType, responseItems, out)

							}
						}
					default:
						fmt.Println("no @include directive")
						p.executeStmt_(rootFrag, qry.SelectionSet, rootPath, responseType, responseItems, out)
					}
				}
			}

		}
	}
}

// ====================================================================================

func (p *Parser) parseOperationStmt(op string) ast.StatementDef {
	// Types: query, mutation, subscription
	var (
		f func(sdl.NameValue_)
		i int
	)
	switch p.curToken.Type {
	case token.QUERY, token.MUTATION, token.SUBSCRIPTION, token.FRAGMENT:
		p.nextToken() // read over query, mutation keywords
	}
	stmt := &ast.OperationStmt{Type: op}
	p.root = stmt //TODO - what is this??

	p.parseName(stmt, opt).parseVariables(stmt, opt).parseDirectives(stmt, opt).parseSelectionSet(stmt)

	f = func(nw sdl.NameValue_) {
		if _, ok := OperationStmts[nw]; !ok {
			OperationStmts[nw] = stmt
			stmt.Name.Name = nw
			return
		} else {
			i++
			if !stmt.Name.EqualString(noName) {
				// dev specified name duplicated
				p.addErr(fmt.Sprintf(`Duplicate statement name "%s" %s`, stmt.Name, stmt.Name.AtPosition()))
			}
			s := stmt.Name.String() + "/" + strconv.Itoa(i)
			f(sdl.NameValue_(s))
		}
	}

	if !stmt.Name.Exists() {
		stmt.Name = sdl.Name_{Name: sdl.NameValue_(noName)}
		f(stmt.Name.Name)
	} else {
		f(stmt.Name.Name)
	}

	return stmt

}

func (p *Parser) parseFragmentStmt(op string) ast.StatementDef {
	var (
		f func(sdl.NameValue_)
		i int
	)
	p.nextToken()               // read over Fragment keyword
	stmt := &ast.FragmentStmt{} // TODO: alternative to Stmt field could simply use check len(Name) to determine if Stmt or inline

	_ = p.parseName(stmt).parseTypeCondition(stmt).parseDirectives(stmt, opt).parseSelectionSet(stmt)

	f = func(nw sdl.NameValue_) {
		if _, ok := FragmentStmts[nw]; !ok {
			FragmentStmts[nw] = stmt
			stmt.Name.Name = nw
			return
		} else {
			i++
			if !stmt.Name.EqualString(noName) {
				// dev specified name duplicated
				p.addErr(fmt.Sprintf(`Duplicate fragment name "%s" %s`, stmt.Name, stmt.Name.AtPosition()))
			}
			s := stmt.Name.String() + "/" + strconv.Itoa(i)
			f(sdl.NameValue_(s))
		}
	}

	if !stmt.Name.Exists() {
		stmt.Name = sdl.Name_{Name: sdl.NameValue_(noName)}
		f(stmt.Name.Name)
	} else {
		f(stmt.Name.Name)
	}
	return stmt
}

func (p *Parser) parseFragmentSpread() ast.SelectionSetProvider {
	p.nextToken("parseFragmentSpread..") // read over ...
	if p.curToken.Type != token.IDENT {
		p.addErr("Identifer expected for fragment spread after ...")
	}
	expnd := &ast.FragmentSpread{}

	p.parseName(expnd).parseDirectives(expnd, opt)

	return expnd
}

// InlineFragment
// ...TypeCondition-opt	Directives-opt	SelectionSet

func (p *Parser) parseInlineFragment(f ast.HasSelectionSetProvider) ast.SelectionSetProvider {

	frag := &ast.InlineFragment{}               //{Parent: f}
	p.nextToken("inlinefragment read over ...") // read over ...

	p.parseTypeCondition(frag, opt).parseDirectives(frag, opt).parseSelectionSet(frag)

	fmt.Printf("ineline fragment:  %#v %s\n", frag, frag.String())

	return frag
}

// type Field struct {
// 	Alias     string
// 	Name      string
// 	Arguments []*Argument
// 	//	directives   []directive
// 	SelectionSet []SelectionSetProvider // field as object
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

//func (p *Parser) extractFragment() ast.HasSelectionSetProvider     { return nil }
//func (p *Parser) parseInlineFragment() ast.HasSelectionSetProvider { return nil }

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
		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			p.abort = true
			return p.addErr(fmt.Sprintf(`Expected an argument name followed by colon got an "%s %s"`, p.curToken.Literal, p.peekToken.Literal))
		}
		v.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
		p.nextToken() // read over :
		p.nextToken() // argument value
		// if !((p.curToken.Cat == token.VALUE && (p.curToken.Type == token.DOLLAR && p.peekToken.Cat == token.VALUE)) ||
		// 	(p.curToken.Cat == token.VALUE && (p.peekToken.Cat == token.NONVALUE || p.peekToken.Type == token.RPAREN)) ||
		// 	(p.curToken.Type == token.LBRACKET || p.curToken.Type == token.LBRACE)) { // [  or {
		// 	return p.addErr(fmt.Sprintf(`Expected an argument Value followed by IDENT or RPAREN got an %s:%s:%s %s:%s:%s`, p.curToken.Cat, p.curToken.Type, p.curToken.Literal, p.peekToken.Cat, p.peekToken.Type, p.peekToken.Literal))
		// }
		v.Value = p.parseInputValue_()
		p.printToken("After parseINputValue..")
		return nil
	}
	// (
	p.printToken("parseArguments ")
	if p.curToken.Type == token.LPAREN {
		p.nextToken() // read over (
	}
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
		p.nextToken("in parseDirectives . Read over @") // read over @
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
func (p *Parser) parseSelectionSet(f ast.HasSelectionSetProvider, optional ...bool) *Parser {
	// TODO - sometimes SS is optional other times its mandatory.  How to handle. Idea: method SelectionSetOptional() - which souces data from optional field, array.
	if p.hasError() {
		return p
	}
	p.printToken("parseSelectionSEt")
	if p.curToken.Type != token.LBRACE {
		if len(optional) == 0 {
			p.addErr(fmt.Sprintf("Expect a selection set %s", p.l.AtPosition()))
		}
		return p
	}
	parseSSet := func() ast.SelectionSetProvider {
		var node ast.SelectionSetProvider

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
			//p.nextToken("SbS next ")

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
	for p.nextToken("SS "); p.curToken.Type != token.RBRACE; {

		node := parseSSet()

		if p.hasError() {
			break
		}

		f.AppendSelectionSet(node) // append each selection set current receiver.

	}
	p.nextToken("read over }") // read over }
	//
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
// VariableASTs :
//		( VariableAST ... )
// VariableAST
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
			return false
		}
		p.nextToken() // read over name identifer

		if !(p.curToken.Type == token.IDENT && p.peekToken.Type == token.COLON) {
			p.addErr(fmt.Sprintf(`Expected an identifer got an "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return false
		}
		v.AssignName(p.curToken.Literal, p.Loc(), &p.perror)
		p.nextToken()
		// :
		if p.curToken.Type != token.COLON {
			p.addErr(fmt.Sprintf(`Expected : got a "%s" value "%s"`, p.curToken.Type, p.curToken.Literal))
			return false
		}
		p.nextToken() // read over :
		p.parseType(v)
		if p.curToken.Type == token.ASSIGN {
			//	p.nextToken() // read over Datatype
			p.nextToken() // read over ASSIGN
			v.DefaultVal = p.parseInputValue_()
		}
		return true
	}

	switch stmt := st.(type) {
	case *ast.OperationStmt:
		p.printToken("In variable..")
		if p.curToken.Type == token.LPAREN {
			for p.curToken.Type != token.RPAREN { //p.nextToken("Next... should be )") {

				v := ast.VariableDef{}

				if parseVariable(&v) {
					stmt.Variable = append(stmt.Variable, &v)
				} else {
					return p
				}
				fmt.Printf("variable: %#v\n", v)
			}
			p.rootVar = stmt.Variable
			p.nextToken("read over )..") //read over )
		} else if len(optional) == 0 { // if argument exists its optional
			p.addErr("Variables are madatory")
		}
		//p.nextToken()
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
		// if p.curToken.Cat != token.VALUE {
		// 	p.addErr(fmt.Sprintf("Expect an Input Value followed by another Input Value or a ], got %s %s ", p.curToken.Literal, p.peekToken.Literal))
		// 	return &sdl.InputValue_{}
		// }
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
		// 	ast_ = p.stmtCache.FetchAST(name_)
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
