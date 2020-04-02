package ast

import (
	"context"
	"fmt"
	"strings"

	sdl "github.com/graph-sdl/ast"
	pse "github.com/graph-sdl/parser"
)

type UnresolvedMap sdl.UnresolvedMap //¬333map[Name_]*sdl.GQLtype

// ============ InputValue VALUE node - must satisfy ValueI =======================

type Scalar_ string

func (sc *Scalar_) ValueNode() {}
func (sc *Scalar_) String() string {
	return string(*sc)
}
func (sc *Scalar_) Exists() bool {
	if len(*sc) > 0 {
		return true
	}
	return false
}

// ======== type system =========

//type NamedGQLtype sdl.Name_

// func (n *NamedGQLtype) String() string {
// 	return n.String()
// }

// ======== Document =================================================================

const (
	OPERATION byte = iota
	FRAGMENT
)

type Statement struct {
	Type    string // Operational | Fragment
	Name    string
	AST     GQLStmtProvider     // AST of operational stmt (query,mutation,sub) or Fragment stmt
	RootAST sdl.GQLTypeProvider // AST of type which represents the entry point to the graph, from schema(query:<entryType>,... Typically type Query.
}
type Document struct {
	Statements []*Statement
}

func (d Document) String() string {
	var s strings.Builder
	tc = 2

	for _, stmt := range d.Statements {
		s.WriteString(stmt.AST.String())
	}
	return s.String()
}

//========== Selection Set =============

// these are the ast structures that have a selectionset collection, which
// maybe different to the objects contained in the selectionset
//  *OperationStmt) AppendSelectionSet(ss SelectionSetProvider) {
//  *FragmentStmt) AppendSelectionSet(ss SelectionSetProvider) {
//  *InlineFragment) AppendSelectionSet(ss SelectionSetProvider) {
//  *Field) AppendSelectionSet(ss SelectionSetProvider) {
type HasSelectionSetProvider interface {
	AppendSelectionSet(ss SelectionSetProvider) // TODO - this method may not be appropriate for this interface.
}

// SelectionSetProvider can be
//   Field
//   FragmentSpread
//	 InlineFragment

type SelectionSetProvider interface {
	SelectionSetNode()
	SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) // TODO: don't like this here.
	//	Resolve()
	String() string
}

type StmtName_ string

//========= statement def ============

type GQLStmtProvider interface {
	//Node()
	StatementNode()
	//TypeSystemNode()
	SolicitNonScalarTypes(unresolved sdl.UnresolvedMap)
	CheckIsInputType(err *[]error)
	CheckInputValueType(err *[]error)
	StmtName() StmtName_
	StmtType() string
	String() string
}

// ======== Document Statements -

// ** currently on Field has an alias so don't bother with interface
// type HasAlias interface {
// 	SetAlias(n string) error
// }

// == ExecutableDefinition - start

// type Executable interface { // TODO - what is this? remove if possible
// 	ExecutableDefinition()
// }

type OperationDef interface {
	OperationNode()
	//	Executable
}

type FragmentDef interface {
	FragmentNode()
	GetSelectionSet() []SelectionSetProvider
	AssignTypeCond(string, *sdl.Loc_, *[]error)
	//	Executable
	AssignTypeCondAST(*[]error, *pse.Cache_)
}

// // == ExecutableDefinition - end

// type TypeSystemDef interface {
// 	TypeSystemNode()
// }

// type TypeExtDef interface {
// 	TypeExtNode()
// }

// =========== OperationDef Instances ==============
// OperationDefinition
//		OperationType	Name-opt	VariableDefinitions-opt	 Directives-opt	 SelectionSet
//
//OperationType
//		query	mutation	subscription
// SelectionSet
//		{ Selection-list }
//Selection
//		Field
//		FragmentSpread
//		InlineFragment

type OperationStmt struct {
	//
	Type string // query, mutation, subscription	SelectionSet []SelectionSetProvider // { only fields and ... fragments
	//
	//NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name     sdl.Name_ // validated
	Variable []*VariableDef
	sdl.Directives_
	SelectionSet []SelectionSetProvider // { selection-List: fields,... }
}

func (o *OperationStmt) StatementNode() {} // validates type is appropriate during load into ast struct
func (o *OperationStmt) OperationNode() {} // validates type is appropriate during load into ast struct
//func (o *OperationStmt) ExecutableDefinition() {}
func (o *OperationStmt) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	o.Name = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (o *OperationStmt) GetSelectionSet() []SelectionSetProvider {
	return o.SelectionSet
}

func (o *OperationStmt) AppendSelectionSet(ss SelectionSetProvider) {
	o.SelectionSet = append(o.SelectionSet, ss)
}

func (o *OperationStmt) CheckInputValueType(err *[]error) {
	for _, v := range o.Variable {
		v.checkInputValueType(err)
	}
	o.Directives_.CheckInputValueType(err)
	// for _, v := range SelectionSet {// performed in checkfields
	// 	v.CheckInputValueType(o, err)
	// }
}

func (o *OperationStmt) CheckIsInputType(err *[]error) {
	for _, p := range o.Variable {
		if !sdl.IsInputType(p.Type) {
			*err = append(*err, fmt.Errorf(`Argument "%s" type "%s", is not an input type %s`, p.Name_, p.Type.Name, p.Type.Name_.AtPosition()))
		}
		//	_ := p.DefaultVal.isType() // e.g. scalar, int | List
	}
}

// SetName, validates input string and assigns to field Name

func (o *OperationStmt) String() string { // Query will now satisfy Node interface and complete GQLStmtProvider
	var s strings.Builder
	//
	// Name may be system supplied for short stmts, starting with "__NONAME__"
	//
	if o.Name.Name[:2] == "__" || !o.Name.Exists() {
		s.WriteString(fmt.Sprintf("\n %s ", o.Type))
	} else {
		s.WriteString(fmt.Sprintf("\n %s %s ", o.Type, o.Name))
	}
	// if !o.Name.Exists() || o.Name.Name[:2] == "__" {
	// 	if o.Name.Name[:2] == "__" {
	// 		s.WriteString(fmt.Sprintf("\n %s ", o.Type))
	// 	} else {
	// 		s.WriteString(fmt.Sprintf("\n %s %s", o.Type, o.Name))
	// 	}
	// } else {
	// 	s.WriteString(fmt.Sprintf(" %s %s", o.Type, o.Name))
	// }
	if len(o.Variable) > 0 {
		s.WriteString("(")
		for _, v := range o.Variable {
			s.WriteString(" " + v.String())
		}
		s.WriteString(") ")
	}
	s.WriteString("{ ")
	for _, v := range o.SelectionSet {
		s.WriteString(v.String())
	}
	for i := tc - 1; i > 0; i-- {
		s.WriteString("\n")
		for i := i; i > 0; i-- {
			s.WriteString(fmt.Sprintf("\t"))
		}
		s.WriteString("}")
	}
	return s.String()
}

func (o *OperationStmt) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {
	// statements can have two entities that are unresolved.
	//  either a type (SDL) or a fragment (statement). However SDL are only checked during checkField function.
	// check any unresolved fragments
	for _, v := range o.Variable {
		v.SolicitNonScalarTypes(unresolved)
	}
	o.Directives_.SolicitAbstractTypes(unresolved) // TODO: should directives be included
	for _, v := range o.SelectionSet {
		v.SolicitNonScalarTypes(unresolved)
	}
}

func (o *OperationStmt) StmtType() string {
	return o.Type
}

func (o *OperationStmt) StmtName() StmtName_ {
	// if !o.Name.Exists() {
	// 	return StmtName_(o.StmtType())
	// }
	if !o.Name.Exists() {
		return StmtName_("")
	}
	return StmtName_(o.Name.String())
}

// ========================= FragmentStmt  ==================================

type FragmentStmt struct {
	//NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name sdl.Name_
	// on <type>
	TypeCond    sdl.Name_
	TypeCondAST sdl.GQLTypeProvider
	sdl.Directives_
	SelectionSet []SelectionSetProvider // inline fragments, fragment spreads, sdl field from sdl type TypeCond.
}

func (f *FragmentStmt) StatementNode() {} // validates type is appropriate during load into ast struct
func (f *FragmentStmt) FragmentNode()  {} // validates type is appropriate during load into ast struct
//func (f *FragmentStmt) ExecutableDefinition() {}
// func (f *FragmentStmt) CheckInputValueType(err *[]error) {
// 	f.Directives_.CheckInputValueType(err)
// 	for _, v := range f.SelectionSet {
// 		v.CheckInputValueType()
// 	}
// }

func (f *FragmentStmt) GetSelectionSet() []SelectionSetProvider {
	return f.SelectionSet
}

func (f *FragmentStmt) AppendSelectionSet(ss SelectionSetProvider) {
	// usual suspects for SS
	//	Selection :
	//		Field from TypeCond type
	//		FragmentSpread
	//		InlineFragment
	f.SelectionSet = append(f.SelectionSet, ss)
}

func (f *FragmentStmt) CheckIsInputType(err *[]error) {}

func (f *FragmentStmt) StmtType() string {
	return "Fragment"
}

func (f *FragmentStmt) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.Name = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *FragmentStmt) AssignTypeCond(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.TypeCond = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *FragmentStmt) String() string { // Query will now satisfy Node interface and complete GQLStmtProvider
	var s strings.Builder
	tc = 1
	s.WriteString("\nfragment ")
	s.WriteString(fmt.Sprintf("%s on %s ", f.Name, f.TypeCond))

	s.WriteString("{ ")
	for _, v := range f.SelectionSet {
		s.WriteString(v.String())
	}
	for i := tc; i > 0; i-- {
		s.WriteString("\n")
		for i := i; i > 0; i-- {
			s.WriteString(fmt.Sprintf("\t"))
		}
		s.WriteString("}")
	}
	return s.String()
}

func (f *FragmentStmt) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {

	unresolved[f.TypeCond] = nil //&ty
	// }
	f.Directives_.SolicitAbstractTypes(unresolved)
	for _, v := range f.SelectionSet {
		v.SolicitNonScalarTypes(unresolved)
	}
}

func (f *FragmentStmt) AssignTypeCondAST(err *[]error, cache *pse.Cache_) {
	// TODO - fix  dont use cache in type stmt methos
	if f.TypeCondAST == nil {
		x, err_ := cache.FetchAST(f.TypeCond.Name)
		if err_ != nil {
			*err = append(*err, err_)
		}
		if x == nil {
			*err = append(*err, fmt.Errorf(`Type Condition for fragment "%s" not found`, f.Name))
			return
		}
		switch x.(type) {
		case *sdl.Object_, *sdl.Union_, *sdl.Interface_:
			f.TypeCondAST = x
		default:
			*err = append(*err, fmt.Errorf(`Type Condition "%s" for fragment "%s" must be an Object, Union or Interface %s`, f.TypeCond, f.Name, f.Name.AtPosition()))

		}
	} else {
		switch f.TypeCondAST.(type) {
		case *sdl.Object_, *sdl.Union_, *sdl.Interface_:
		default:
			*err = append(*err, fmt.Errorf(`Type Condition "%s" for fragment "%s" must be an Object, Union or Interface %s`, f.TypeCond, f.Name, f.Name.AtPosition()))

		}
	}
}

func (f *FragmentStmt) StmtName() StmtName_ {
	return StmtName_(f.Name.String())
}

var tc = 2

// =============== SelectionSet Types =====================

// Fragment Spread - consumes Fragment Statements.
// FragmentSpread
// ...FragmentName  Directives-opt

type FragmentSpread struct {
	sdl.Name_ // AST only contains reference to Fragment. At evaluation time it will be expanded to its enclosed fields.
	sdl.Directives_
	FragStmt *FragmentStmt // associated fragment statement
	//	SelectionSet []SelectionSetProvider // expanded results are added here - no do not include this. Name is reference to Fragment Statement object
}

func (f *FragmentSpread) SelectionSetNode() {}

// not SDL type information can be specified in spread, however we should resolve associated fragment stmt. NO. Fragment stmt will always be defined in query document.
// TODO: consider Fragment library stored in db. This could be dangerous as defnition is not stored with query and could lead to errors very easily e.g. db def gets changed by someone.
func (f *FragmentSpread) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {}

//func (f *FragementSpread) ExecutableDefinition() {}

func (f *FragmentSpread) CheckInputValueType(err *[]error) {}

func (f *FragmentSpread) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.Name_ = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *FragmentSpread) StmtType() string {
	return "FragmentSpread"
}

func (f *FragmentSpread) String() string {
	var s strings.Builder
	s.WriteString("\n")
	for i := tc; i > 0; i-- {
		s.WriteString(fmt.Sprintf("\t"))
	}
	s.WriteString("..." + f.Name.String())
	return s.String()
}

// InlineFragment
// ...TypeCondition-opt	Directives-opt	SelectionSet

type InlineFragment struct {
	//
	//Parent      HasSelectionSetProvider
	TypeCond    sdl.Name_           // supplied by typeCondition if specified, otherwise its the type of the parent object's selectionset.
	TypeCondAST sdl.GQLTypeProvider // populated during checkField??
	//
	sdl.Directives_
	SelectionSet []SelectionSetProvider // { only fields and ... fragments. Nil when no TypeCond and adopts selectionSet of enclosing context.
}

func (f *InlineFragment) SelectionSetNode() {}

//func (f *InlineFragment) Resolve()          {}
func (f *InlineFragment) FragmentNode() {}
func (f *InlineFragment) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {
	if f.TypeCond.Exists() {
		unresolved[f.TypeCond] = nil
	}
	for _, v := range f.SelectionSet {
		v.SolicitNonScalarTypes(unresolved)
	}
}

//func (f *InlineFragment) ExecutableDefinition() {}

func (f *InlineFragment) AppendSelectionSet(ss SelectionSetProvider) {
	// usual suspects for SS
	//	Selection :
	//		Field
	//		FragmentSpread
	//		InlineFragment
	f.SelectionSet = append(f.SelectionSet, ss)
}

func (f *InlineFragment) AssignTypeCond(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.TypeCond = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *InlineFragment) GetSelectionSet() []SelectionSetProvider {
	return f.SelectionSet
}

func (f *InlineFragment) AssignTypeCondAST(err *[]error, cache *pse.Cache_) {
	if f.TypeCondAST == nil {
		x, err_ := cache.FetchAST(f.TypeCond.Name)
		if err_ != nil {
			*err = append(*err, err_)
		}
		if x == nil {
			*err = append(*err, fmt.Errorf(`Type Condition for inline fragment not found`))
			return
		}
		switch x.(type) {
		case *sdl.Object_, *sdl.Union_, *sdl.Interface_:
			f.TypeCondAST = x
		default:
			*err = append(*err, fmt.Errorf(`Type Condition "%s" for inline fragment must be an Object, Union or Interface`, f.TypeCond))

		}
	} else {
		switch f.TypeCondAST.(type) {
		case *sdl.Object_, *sdl.Union_, *sdl.Interface_:
		default:
			*err = append(*err, fmt.Errorf(`Type Condition "%s" for inline fragment must be an Object, Union or Interface`, f.TypeCond))

		}
	}

}
func (f *InlineFragment) CheckInputValueType(err *[]error) {}

func (f *InlineFragment) String() string { // Query will now satisfy Node interface and complete GQLStmtProvider
	var s strings.Builder
	s.WriteString("\n")
	tabs := tc
	for i := tc; i > 0; i-- {
		s.WriteString(fmt.Sprintf("\t"))
	}
	s.WriteString("...")
	if len(f.TypeCond.Name) > 0 {
		s.WriteString(" on ")
		s.WriteString(f.TypeCond.String())
	}
	//
	// Directives
	//
	for _, v := range f.Directives {
		s.WriteString(v.String())
	}
	//
	//  SelectionSet
	//
	if len(f.SelectionSet) > 0 {
		tc++
		s.WriteString(" {")
		for i := 0; i < tc; i++ {
			s.WriteString("\t")
		}
		//s.WriteString("Len " + strconv.Itoa(len(f.SelectionSet)))
		for _, v := range f.SelectionSet {
			s.WriteString(v.String())
		}
		s.WriteString("\n")
		for i := 0; i < tc; i++ {
			s.WriteString("\t")
		}
		s.WriteString("}")
		tc--
	}
	for i := tc; i > tabs; i-- {
		s.WriteString("\n")
		for i := i; i > tabs; i-- {
			s.WriteString(fmt.Sprintf("\t"))
		}
		s.WriteString("}")
	}
	return s.String()
}

// info

type QLInfo struct {
	Dummy string
}

// Field.		===========================================================================================================

type Field struct {
	//Type  int // Fragment, InlineFragment, Field
	Alias sdl.Name_
	Name  sdl.Name_ // must have atleast a name - all else can be empty
	//
	SDLRootAST sdl.GQLTypeProvider // Parent type, populated during checkField. Could be sdl.Object_, sdl.Interface, sdl.Union_(??)
	SDLfld     *sdl.Field_         // matching field in sdl object. populated during checkField
	Path       string              // path to field in statement
	//
	sdl.Arguments_ // promoted from struct: "Arguments []*ArgumentT"   ArgumenT {Name_, value InputValue_}, InputValue_ { InputValueProvider, Loc}
	sdl.Directives_
	SelectionSet []SelectionSetProvider //a Field whose type is an object (within the parent type to which field belongs) will have associated fields. For scalars SS wll be nil
	//
	Resolver func(context.Context, sdl.InputValueProvider, sdl.ObjectVals) <-chan string
}

func (f *Field) SelectionSetNode() {}

//func (f *Field) Resolve()          {}

//func (f *Field) ExecutableDefinition() {} // removed as Field is not a statement

func (f *Field) AppendSelectionSet(ss SelectionSetProvider) {
	f.SelectionSet = append(f.SelectionSet, ss)
}

func (f *Field) StmtType() string { // to support NameI - not relevant for Field
	return ""
}

// type GQLtype struct {
// 	Constraint byte            // each bit from right represents not-null constraint applied e.g. in nested list type [type]! is 00000010, [type!]! is 00000011, type! 00000001
// 	AST        GQLTypeProvider // AST instance of type. WHen would this be used??. Used for non-Scalar types. AST in cache(typeName), then in GQLtype(typeName). If not in GQLtype, check cache, then DB.
// 	Depth      int             // depth of nested List e.g. depth 2 is [[type]]. Depth 0 implies non-list type, depth > 0 is a list type
// 	Name_                      // type name. inherit AssignName(). Use Name_ to access AST via cache lookup. ALternatively, use AST above or TypeFlag_ instead of string.
// 	Base       string          // base type e.g. Name_ = "Episode" has Base = E(num)
// }
// type Field_ struct {
// Desc string
// Name_
// ArgumentDefs InputValueDefs  // InputValueDefs []InputValueDef
// Type *GQLtype
// Directives_

// type InputValueDef struct {
// 	Desc string
// 	Name_
// 	Type       *GQLtype
// 	DefaultVal *InputValue_
// 	Directives_
// }
// type Arguments_ struct {
// 	Arguments []*ArgumentT
// }
// type ArgumentT struct {
// 	//( name : value )
// 	Name_
// 	Value *InputValue_
// }
// func (a *Arguments_) AppendArgument(ss *ArgumentT) {
// 	a.Arguments = append(a.Arguments, ss)
// }
func (f *Field) ExpandArguments(root *sdl.Field_, err *[]error) (failed bool) {
	fmt.Println(" = = = = = = = = = = = = = = = = = = = = = = =  ExpandArguments = = = = = = = = = = = = = = = = ")
	for _, rfa := range root.ArgumentDefs {
		var found bool
		for _, fa := range f.Arguments { // sdl.Arguments_
			if rfa.Name.Equals(fa.Name) {
				found = true
				break
			}
			// 	for _,dir := range rfa.Directives {
			// 		if dir.Name == "Include" {

			// 		}

			// 	break
			// }
		}
		if !found && !rfa.Type.IsNullable() {
			if rfa.DefaultVal != nil {
				// has a default vaue - append to field arguments
				arg := sdl.ArgumentT{Name_: sdl.Name_{Name: sdl.NameValue_(rfa.Name)}, Value: rfa.DefaultVal}
				f.Arguments_.AppendArgument(&arg)
			} else {
				failed = true
				*err = append(*err, fmt.Errorf(`Argument "%s" is not nullable from field "%s" %s`, rfa.Name, root.Name, rfa.Name_.AtPosition()))
			}
		}
	}
	return failed
}

func (f *Field) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {
	// get type of the field
	// TODO need to have type name associated with this field either in Field struct or passed into checkUnresolvedType
	//      also sdl.CacheFetch need type name passed in not field name
	//      CANNOT resolve field Type as it is not known at this point (populated during checkField which will then resolve the type)
	// if len(f.SelectionSet) != 0 { // non-Scalar type
	// 	unresolved[f.Name] = nil
	// }
	f.Directives_.SolicitAbstractTypes(unresolved)
	// TODO - need the type nane for the arguments, as its the type we want to resolve.
	//f.Arguments_.SolicitAbstractTypes(unresolved) // added 31 March 2020
	for _, v := range f.SelectionSet {
		v.SolicitNonScalarTypes(unresolved)
	}
}

func (f *Field) GenNameAliasPath() string {
	if f.Alias.Exists() {
		return f.Name.String() + "(" + f.Alias.String() + ")"
	}
	return f.Name.String()
}

// func (f *Field) CheckInputValueType(reftype *sdl.Type, argName sdlName_, err *[]error) {
// 	//arguments performed in parser's checkFields_()
// 	f.Diectivies_.CheckInputValueType(err)
// }

// func (f *Field) CheckInputValueType( err *[]error) {
// 	//arguments performed in parser's checkFields_()
// 	f.Diectivies_.CheckInputValueType(err)
// }

func (f *Field) AppendArgument(ss *sdl.ArgumentT) {
	f.Arguments = append(f.Arguments, ss)
}

func (f *Field) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.Name = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *Field) AssignAlias(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.Alias = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *Field) String() string {
	var s strings.Builder
	s.WriteString("\n")
	for i := 0; i < tc; i++ {
		s.WriteString("\t")
	}
	if len(f.Alias.Name) > 0 {
		s.WriteString(fmt.Sprintf("%s : %s ", f.Alias.String(), f.Name.String()))
	} else {
		s.WriteString(f.Name.String())
	}
	//
	if len(f.Arguments) > 0 {
		s.WriteString(f.Arguments_.String())
	}
	if len(f.Directives) > 0 {
		s.WriteString(f.Directives_.String())
	}
	//
	//  SelectionSet
	//
	if len(f.SelectionSet) > 0 {
		tc++
		s.WriteString(" {")
		for i := 0; i < tc; i++ {
			s.WriteString("\t")
		}
		//	s.WriteString("Len " + strconv.Itoa(len(f.SelectionSet)))
		for _, v := range f.SelectionSet {
			s.WriteString(v.String())
		}
		s.WriteString("\n")
		for i := 0; i < tc; i++ {
			s.WriteString("\t")
		}
		s.WriteString("}")
		tc--
	}

	return s.String()
}

//=========== Variable Def =============

type VariableDef struct {
	sdl.Name_
	Type       *sdl.GQLtype
	DefaultVal *sdl.InputValue_
	Value      *sdl.InputValue_ // assigned by variable statment, defined outside of operationalStmt
}

func (v *VariableDef) String() string {
	if v.DefaultVal != nil {
		return "$" + v.Name.String() + " : " + v.Type.String() + " = " + v.DefaultVal.String()
	} else {
		return "$" + v.Name.String() + " : " + v.Type.String()
	}
}

func (v *VariableDef) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	v.Name_.AssignName(input, loc, err)
}

func (n *VariableDef) AssignType(t *sdl.GQLtype) {
	n.Type = t
}

func (n *VariableDef) SolicitNonScalarTypes(unresolved sdl.UnresolvedMap) {
	if !n.Type.IsScalar() {
		if n.Type.AST == nil {
			// check in cache only at this stage.
			// When control passes back to parser we resolved the unresolved using the DB and parse stmt if found.
			// TODO - fix - cannot use cache in non parser methods?
			// if ast, ok := sdl.CacheFetch(n.Type.Name); !ok {
			unresolved[n.Type.Name_] = n.Type
			// } else {
			// 	n.Type.AST = ast
			// }
		}
	}
}

func (a *VariableDef) checkInputValueType(err *[]error) {

	a.DefaultVal.CheckInputValueType(a.Type, a.Name_, err)

}

type NameI interface {
	AssignName(name string, loc *sdl.Loc_, err *[]error)
	StmtType() string
}

// =================================================================
