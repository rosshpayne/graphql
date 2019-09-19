package gtype

// ast:
// type VariableDef struct {
// 	Name       gtype.Name
// 	Type       gtype.Type
// 	DefaultVal gtype.InputValue
// }

type Name string

type intV struct {
    value int
}

type FloatV struct {
    value float64
}

type StringV strcut {
    value string
}

type IntT string
type FloatT interface {
    gfloat()
    String()
}
type Float_ struct {
    name  string
    value float
}

type StringT string

func (i IntT) String() string {
    return "int"
}
func (f FloatT) String() string {
    return "float"
}
func (s StringT) String() string {
    return "string"
}
type Name_ string

func (n Name_) Assign(s string) {
    n=Name_(s)
}

var s type.StringT

s.String()
s.Assign("abc")
