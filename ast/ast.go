package ast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/graphql/token"
)

// ================= InputValueType  =================================

// don't turn this into an interface as it will only ever be a string type - atleast at this point in time.

type inputValueType_ string

func (i inputValueType_) String() string {
	return string(i)
}
func (i *inputValueType_) InputValueTypeNode() {}

func (i *inputValueType_) assign(type_ string) error { // helper method: do not expose this type as its accessed thru AssignType method below
	if len(type_) == 0 {
		return errors.New(fmt.Sprintf("error in validateName_() - no argument supplied"))
	}
	switch type_ {
	case token.INT, token.FLOAT, token.STRING, token.RAWSTRING, token.NULL, token.BOOL, "List":
		*i = inputValueType_(type_)
	case token.TRUE, token.FALSE:
		*i = inputValueType_(token.BOOL)
	default:
		return errors.New(fmt.Sprintf("Input value type not supported [%s]", type_))
	}
	return nil
}

func (i *inputValueType_) AssignType(type_ string) error {
	var t inputValueType_ = inputValueType_("")
	err := t.assign(type_) // let go run: (&t).Assign. This will assign the argument value to t.
	if err != nil {
		return err
	}
	*i = t
	return nil
}

// =============== Name_  ==========================

type Name_ string

func (n Name_) String() string {
	return string(n)
}
func (n Name_) Exists() bool {
	if len(n) > 0 {
		return true
	}
	return false
}

// =================  InputValue =================================

// type InputValue_ struct {
// 	Value           string //  Token.Literal
// 	inputValueType_        //  Token.Type - scalar types, List, Object, Enum, Null
// }

type ValueI interface {
	ValueNode()
	String() string
	Exists() bool
}

type InputValue_ struct {
	Value           ValueI //  Three concrete types;Int, Float, String, Enum, List, Object,null
	inputValueType_        //  Token.Type - scalar types, List, Object, Enum, Null
}

func (iv *InputValue_) InputValueNode() {}

func (iv *InputValue_) String() string {
	switch iv.inputValueType_.String() {
	case token.RAWSTRING:
		return token.RAWSTRINGDEL + iv.Value.String() + token.RAWSTRINGDEL //+ iv.inputValueType_.String()
	case token.STRING:
		return token.STRINGDEL + iv.Value.String() + token.STRINGDEL //+ iv.inputValueType_.String()
	}
	return iv.Value.String() + " " //+ iv.inputValueType_.String()

}

func (iv *InputValue_) Exists() bool {
	return (*iv).Value.Exists()
}

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

type Null_ bool // moved from Scalar to it's own type. No obvious reason why - no obvious advantage at this stage

func (n Null_) ValueNode() {}
func (n Null_) String() string {
	if n == false {
		return ""
	}
	return "null"
}
func (n Null_) Exists() bool {
	return bool(n)
}

// type Int_ int
// func (i Int_) ValueNode() {}

// type Float_ float
// func (f Floatt_) ValueNode() {}

// type String_ string
// func (s String_) ValueNode() {}

// type Bool_ bool
// func (i Bool_) ValueNode() {}

type Enum_ Name_

func (e Enum_) ValueNode() {}

type List_ []InputValue_

func (l List_) ValueNode() {}
func (l List_) String() string {
	var s strings.Builder
	s.WriteString("[")
	for _, v := range l {
		s.WriteString(v.String() + " ")
	}
	s.WriteString("]")
	return s.String()
}
func (l List_) Exists() bool {
	if len(l) > 0 {
		return true
	}
	return false
}

type Object_ []*IVobject_

// Object { name : value , name : value , ... }
type IVobject_ struct {
	Name  Name_
	Value ValueI
}

func (o Object_) ValueNode() {}
func (o Object_) String() string {
	var s strings.Builder
	s.WriteString("{")
	for _, v := range o {
		s.WriteString(v.Name.String() + " : " + v.Value.String())
	}
	s.WriteString("}")
	return s.String()
}
func (o Object_) Exists() bool {
	if len(o) > 0 {
		return true
	}
	return false
}

//=========== Variable Def =============

type VariableDef struct {
	Name            string
	inputValueType_ // string specifying types: int, float, string, ID, Bool, Enum, Object. V
	// variableDef will inherit field "inputValueType_ string" plus its methods, String & AssignType. Go will automagically change receiver to pointer to reciever and execute method.
	DefaultVal InputValue_
	Value      InputValue_ // assigned by variable statment, defined outside of operationalStmt
}

func (v *VariableDef) String() string {
	if v.DefaultVal.Value.Exists() {
		return "$" + v.Name + " : " + v.inputValueType_.String() + " = " + v.DefaultVal.String()
	} else {
		return "$" + v.Name + " : " + v.inputValueType_.String()
	}
}

//========== Arguments =================

type Argument struct {
	//( name:value )
	Name  string
	Value InputValue_ // could use string as this value is mapped directly to get function - at this stage we don't care about its type maybe?
}

// Enum_,List_,Object_ not currently supported

// func (n *Name_) SetName(s string) error {
// 	if err := validateName_(s); err != nil {
// 		return err
// 	}
// 	n2 := Name_(s)
// 	n = &n2
// 	return nil
// }
// func (n *Name_) Name() Name_ {
// 	return *n
// }

// ================= InputValues End =================================

// // ===== Type =============

// type Type_ interface {
// 	gType()
// 	String()
// 	Assign(s string)
// }

// type IntT int

// func (i *IntT) Type()           {}
// func (i *IntT) InputValueNode() {}
// func (i *IntT) Assign(s string) {
// 	i=strconv.Atoi(s)
// }
// func (*i IntT) String() string {
// 	return "Int"
// }

// type FloatT float64

// func (f FloatT) Type()           {}
// func (f FloatT) InputValueNode() {}
// func (f FloatT) String() string {
// 	return "Float"
// }

// =======  node  ===============

type Node interface {
	TokenLiteral() string
	String() string
}

type NodeDef struct {
	Token token.Token
}

func (n *NodeDef) TokenLiteral() string {
	return n.Token.Literal
}

// ======== Document =================================================================

type Document struct {
	Statements []StatementDef
}

func (d Document) String() string {
	var s strings.Builder

	for _, iv := range d.Statements {
		s.WriteString(iv.String())
	}
	return s.String()
}

//========== Selection Set =============

// these are the ast structures that have a selectionset collection, which
// maybe different to the objects contained in the selectionset
type HasSelectionSetI interface { //TODO merge or replace with HasSelectionSet
	AppendSelectionSet(ss IsSelectionSetI) *string
	SelectionSetNode()
}

// note there is distinction between what nodes contain a collection of selectionset objects
// and the object itself - which may not contain a collection of selectionset objects. T
// this distinction was not made previously
// There are the objects that can appear in a selectionset
type IsSelectionSetI interface { //TODO merge or replace with HasSelectionSet
	String() string
	IsSelectionSetNode()
}

//========= statement def ============

type StatementDef interface {
	Node
	StatementNode()
}

type DirectiveI interface {
	HasDirective()
}

// ======== Document Statements -

// ** currently on Field has an alias so don't bother with interface
// type HasAlias interface {
// 	SetAlias(n string) error
// }

// == ExecutableDefinition - start

type Executable interface {
	ExecutableDefinition()
}

type OperationDef interface {
	OperationNode()
	Executable
}

type FragmentDef interface {
	FragmentNode()
	Executable
}

// == ExecutableDefinition - end

type TypeSystemDef interface {
	TypeSystemNode()
}

type TypeExtDef interface {
	TypeExtNode()
}

const (
	QUERY        string = "query"
	MUTATION            = "mutation"
	SUBSCRIPTION        = "subscription"
	FRAGMENT            = "fragment"
)

// =========== OperationDef Instances ==============
//
//
type OperationStmt struct {
	//
	Type string // query, mutation, subscription
	// note: operation types are defined by their own type - see below
	//
	NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name     string // validated in parse
	Variable []*VariableDef
	//	Directives   []directiveT
	SelectionSet []IsSelectionSetI // { selection-List: fields,...,SelectionSet }
}

func (o *OperationStmt) StatementNode()        {} // validates type is appropriate during load into ast struct
func (o *OperationStmt) OperationNode()        {} // validates type is appropriate during load into ast struct
func (o *OperationStmt) ExecutableDefinition() {}
func (o *OperationStmt) HasDirective()         {}
func (o *OperationStmt) SelectionSetNode()     {}

//func (o *OperationStmt) SelectionSetNode()     {}

func (o *OperationStmt) AppendSelectionSet(ss IsSelectionSetI) *string {
	o.SelectionSet = append(o.SelectionSet, ss)
	// all IsSelectionSetI objects can be appended
	return nil
}

// SetName, validates input string and assigns to field Name

func (o *OperationStmt) String() string { // Query will now satisfy Node interface and complete StatementDef
	var s strings.Builder

	if len(o.Name) > 0 {
		s.WriteString(fmt.Sprintf("%s %s", o.Type, o.Name))
	} else {
		s.WriteString(fmt.Sprintf("%s ", o.Type))
	}
	if len(o.Variable) > 0 {
		s.WriteString("(")
		for _, v := range o.Variable {
			s.WriteString(" " + v.String())
		}
		s.WriteString(") { ")
	}
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

type FragmentStmt struct {
	// note: operation types are defined by their own type - see below
	NodeDef // partially implements Node interface - concrete type must assign its own String method
	//
	Name string
	//  TypeC TypeCondition
	//	Directives   []directiveT
	SelectionSet []*Field // { selection-List: fields,... only I think }
}

func (o *FragmentStmt) StatementNode()        {} // validates type is appropriate during load into ast struct
func (o *FragmentStmt) FragmentNode()         {} // validates type is appropriate during load into ast struct
func (o *FragmentStmt) ExecutableDefinition() {}
func (o *FragmentStmt) HasDirective()         {}
func (o *FragmentStmt) SelectionSetNode()     {}

func (o *FragmentStmt) AppendSelectionSet(ss IsSelectionSetI) *string {
	// only Field ss can be appended to a Fragment stmt
	if f, ok := ss.(*Field); ok {
		o.SelectionSet = append(o.SelectionSet, f)
		return nil
	} else {
		s := " Only fields supported in Fragment Statments"
		return &s
	}
}

func (o *FragmentStmt) String() string { // Query will now satisfy Node interface and complete StatementDef
	var s strings.Builder

	s.WriteString("\nFragment %s ")

	for _, v := range o.SelectionSet {
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

// ====================================
// this object consumes Fragment Sttements
type FragmentSpread struct{}

func (f *FragmentSpread) IsSelectionSetI() {} // is a selectionset object

// ====================================
var tc = 2

type Field struct {
	Alias     string
	Name      string
	Arguments []*Argument
	//	directives   []directive
	SelectionSet []IsSelectionSetI // field as object
}

func (f *Field) HasDirective()       {}
func (f *Field) IsSelectionSetNode() {}
func (f *Field) SelectionSetNode()   {}
func (f *Field) AppendSelectionSet(ss IsSelectionSetI) *string {
	f.SelectionSet = append(f.SelectionSet, ss)
	// all IsSelectionSetI objects can be appended
	return nil
}

func (f *Field) String() string {
	var s strings.Builder
	s.WriteString("\n")
	for i := 0; i < tc; i++ {
		s.WriteString("\t")
	}
	if len(f.Alias) > 0 {
		s.WriteString(fmt.Sprintf("%s : %s ", f.Alias, f.Name))
	} else {
		s.WriteString(f.Name)
	}
	//
	// Arguments
	//
	if len(f.Arguments) > 0 {
		s.WriteString("( ")
		for _, v := range f.Arguments {
			s.WriteString(v.Name + " : " + v.Value.String())
		}
		s.WriteString(")")
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

type InlineFragment struct {
	//	typecondition []typecond
	//	directives    []directtive
	SelectionSet []IsSelectionSetI
}

func (i *InlineFragment) String() {}

func (i *InlineFragment) AppendSelectionSet(ss IsSelectionSetI) *string {
	i.SelectionSet = append(i.SelectionSet, ss)
	// all IsSelectionSetI objects can be appended
	return nil
}

func (i *InlineFragment) SelectionSetNode()   {}
func (i *InlineFragment) IsSelectionSetNode() {}

//
