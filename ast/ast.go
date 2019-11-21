package ast

import (
	"fmt"
	"strings"

	sdl "github.com/graph-sdl/ast"
	_ "github.com/graphql/token"
)

type UnresolvedMap sdl.UnresolvedMap //map[Name_]*sdl.Type_

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

//type NamedType_ sdl.Name_

// func (n *NamedType_) String() string {
// 	return n.String()
// }

// ======== Document =================================================================

type Document struct {
	Statements []StatementDef
}

func (d Document) String() string {
	var s strings.Builder
	tc = 2

	for _, iv := range d.Statements {
		s.WriteString(iv.String())
	}
	return s.String()
}

//========== Selection Set =============

// these are the ast structures that have a selectionset collection, which
// maybe different to the objects contained in the selectionset
type HasSelectionSetI interface {
	AppendSelectionSet(ss SelectionSetI) // TODO - this method may not be appropriate for this interface.
}

type SelectionSetI interface {
	SelectionSetNode()
	checkUnresolvedTypes_(unresolved sdl.UnresolvedMap)
	Resolve()
	String() string
}

//========= statement def ============

type StatementDef interface {
	//Node()
	StatementNode()
	//TypeSystemNode()
	CheckUnresolvedTypes(unresolved sdl.UnresolvedMap)
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
	AssignTypeCond(string, *sdl.Loc_, *[]error)
	//	Executable
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
	Type string // query, mutation, subscription	SelectionSet []SelectionSetI // { only fields and ... fragments
	//
	//NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name     sdl.Name_ // validated
	Variable []*VariableDef
	sdl.Directives_
	SelectionSet []SelectionSetI // { selection-List: fields,...,SelectionSet }
}

func (o *OperationStmt) StatementNode() {} // validates type is appropriate during load into ast struct
func (o *OperationStmt) OperationNode() {} // validates type is appropriate during load into ast struct
//func (o *OperationStmt) ExecutableDefinition() {}
func (o *OperationStmt) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	o.Name = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (o *OperationStmt) GetSelectionSet() []SelectionSetI {
	return o.SelectionSet
}

func (o *OperationStmt) AppendSelectionSet(ss SelectionSetI) {
	o.SelectionSet = append(o.SelectionSet, ss)
}

func (o *OperationStmt) CheckInputValueType(err *[]error) {
	for _, v := range o.Variable {
		v.checkInputValueType(err)
	}
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

func (o *OperationStmt) String() string { // Query will now satisfy Node interface and complete StatementDef
	var s strings.Builder

	if len(o.Name.Name) > 0 {
		s.WriteString(fmt.Sprintf("%s %s", o.Type, o.Name))
	} else {
		s.WriteString(fmt.Sprintf("%s ", o.Type))
	}
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

func (o *OperationStmt) CheckUnresolvedTypes(unresolved sdl.UnresolvedMap) {
	for _, v := range o.Variable {
		v.checkUnresolvedTypes_(unresolved)
	}
	o.Directives_.CheckUnresolvedTypes(unresolved)
	for _, v := range o.SelectionSet {
		v.checkUnresolvedTypes_(unresolved)
	}
}

func (o *OperationStmt) StmtType() string {
	return o.Type
}

func (o *OperationStmt) StmtName() StmtName_ {
	return StmtName_(o.Name.String())
}

// ========================= FragmentStmt  ==================================

type FragmentStmt struct {
	//NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name sdl.Name_
	// on <type>
	TypeCond sdl.Name_
	sdl.Directives_
	SelectionSet []SelectionSetI // { only fields and ... fragments
}

func (f *FragmentStmt) StatementNode() {} // validates type is appropriate during load into ast struct
func (f *FragmentStmt) FragmentNode()  {} // validates type is appropriate during load into ast struct
//func (f *FragmentStmt) ExecutableDefinition() {}
func (f *FragmentStmt) CheckInputValueType(err *[]error) {}

func (f *FragmentStmt) GetSelectionSet() []SelectionSetI {
	return f.SelectionSet
}
func (f *FragmentStmt) AppendSelectionSet(ss SelectionSetI) {
	// usual suspects for SS
	//	Selection :
	//		Field
	//		FragmentSpread
	//		InlineFragment
	f.SelectionSet = append(f.SelectionSet, ss)
}
func (f *FragmentStmt) CheckIsInputType(err *[]error) {
}

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

func (f *FragmentStmt) String() string { // Query will now satisfy Node interface and complete StatementDef
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

func (f *FragmentStmt) CheckUnresolvedTypes(unresolved sdl.UnresolvedMap) {

	f.Directives_.CheckUnresolvedTypes(unresolved)
	for _, v := range f.SelectionSet {
		v.checkUnresolvedTypes_(unresolved)
	}
}
func (f *FragmentStmt) StmtName() StmtName_ {
	return StmtName_(f.Name.String())
}

var tc = 2

// =============== SelectionSet Types =====================

// Fragment Spread - consumes Fragment Statements.

type FragementSpread struct {
	Name    sdl.Name_     // AST only contains reference to Fragment. At evaluation time it will be expanded to its enclosed fields.
	FragDef *FragmentStmt // or use the cache to find the statement based on Name.
	//	SelectionSet []SelectionSetI // expanded results are added here - no do not include this. Name is reference to Fragment Statement object
}

func (f *FragementSpread) SelectionSetNode()                                  {}
func (f *FragementSpread) Resolve()                                           {}
func (f *FragementSpread) checkUnresolvedTypes_(unresolved sdl.UnresolvedMap) {} // TODO - do we want to add Name to unresolved to check that its associated with a actual Fragment Statement

//func (f *FragementSpread) ExecutableDefinition() {}

func (f *FragementSpread) AssignName(input string, loc *sdl.Loc_, err *[]error) {
	sdl.ValidateName(input, err, loc)
	f.Name = sdl.Name_{Name: sdl.NameValue_(input), Loc: loc}
}

func (f *FragementSpread) String() string {
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
	Parent   HasSelectionSetI
	TypeCond sdl.Name_ // supplied by typeCondition if specified, otherwise its the type of the parent object's selectionset.
	//
	sdl.Directives_
	SelectionSet []SelectionSetI // { only fields and ... fragments
}

func (f *InlineFragment) SelectionSetNode() {}
func (f *InlineFragment) Resolve()          {}
func (f *InlineFragment) FragmentNode()     {}
func (f *InlineFragment) checkUnresolvedTypes_(unresolved sdl.UnresolvedMap) {
	if f.TypeCond.Exists() {
		unresolved[sdl.Name_(f.TypeCond)] = nil
	}
	for _, v := range f.SelectionSet {
		v.checkUnresolvedTypes_(unresolved)
	}
}

//func (f *InlineFragment) ExecutableDefinition() {}

func (f *InlineFragment) AppendSelectionSet(ss SelectionSetI) {
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

func (f *InlineFragment) String() string { // Query will now satisfy Node interface and complete StatementDef
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

// Field

type Field struct {
	//Type  int // Fragment, InlineFragment, Field
	Alias sdl.Name_
	Name  sdl.Name_ // must have atleast a name - all else can be empty
	Path  string    // path to field in statement
	sdl.Arguments_
	sdl.Directives_
	SelectionSet []SelectionSetI                                              //a Field may contain a SS or it may not
	Resolver     func(obj sdl.InputValueProvider, args sdl.ObjectVals) string // fieldResolver //, info QLInfo)
}

func (f *Field) SelectionSetNode() {}
func (f *Field) Resolve()          {}

//func (f *Field) ExecutableDefinition() {} // removed as Field is not a statement

func (f *Field) AppendSelectionSet(ss SelectionSetI) {
	f.SelectionSet = append(f.SelectionSet, ss)
}

func (f *Field) checkUnresolvedTypes_(unresolved sdl.UnresolvedMap) {
	f.Directives_.CheckUnresolvedTypes(unresolved)
	for _, v := range f.SelectionSet {
		v.checkUnresolvedTypes_(unresolved)
	}
}

// func (f *Field) checkInputValueType(reftype *sdl.Type, argName sdlName_, err *[]error) {

// 	for _, v := range f.Arguments_ {
// 		v.CheckInputValueType__(reftype, argName, err)
// 	}
// }

// func (f *Field) AppendArgument(ss *ArgumentT) {
// 	f.Arguments = append(f.Arguments, ss)
// }

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
	Type       *sdl.Type_
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

func (n *VariableDef) AssignType(t *sdl.Type_) {
	n.Type = t
}

func (n *VariableDef) checkUnresolvedTypes_(unresolved sdl.UnresolvedMap) {
	if !n.Type.IsScalar() {
		if n.Type.AST == nil {
			// check in cache only at this stage.
			// When control passes back to parser we resolved the unresolved using the DB and parse stmt if found.
			if ast, ok := sdl.CacheFetch(n.Type.Name); !ok {
				unresolved[n.Type.Name_] = n.Type
			} else {
				n.Type.AST = ast
			}
		}
	}
}

func (a *VariableDef) checkInputValueType(err *[]error) {

	a.DefaultVal.CheckInputValueType(a.Type, a.Name_, err)

}

type NameI interface {
	AssignName(name string, loc *sdl.Loc_, err *[]error)
}

// =================================================================
