package ast

import (
	"fmt"
	"testing"
)

// {
// me {
// Ross
// } }

func TestString(t *testing.T) {
	document := &Document{
		Statements: []StatementDef{
			&OperationStmt{
				Type: "query",
				Name: "me",
				SelectionSet: []SelectionSetI{
					&Field{
						Alias: "myalias",
						Name:  "myName",
					},
					&Field{
						Name: "myName2",
					},
					&Field{
						Name: "myName3",
						Arguments: []*Argument{
							{Name: "age", Value: InputValue_{Value: "6197.45", Type: "Float"}}, // go will auto take pointer to arg
							{Name: "height", Value: InputValue_{Value: "1223", Type: "Int"}},
						},
						SelectionSet: []SelectionSetI{
							&Field{
								Alias: "inneralias",
								Name:  "inner",
							},
							&Field{
								Name: "innername2",
							},
							&Field{
								Name: "myName3",
								SelectionSet: []SelectionSetI{
									&Field{
										Alias: "inneralias3",
										Name:  "inner3",
									},
									&Field{
										Name: "innername3",
									},
									&Field{
										Name: "myName5",
										SelectionSet: []SelectionSetI{
											&Field{
												Alias: "inneralias5",
												Name:  "inner5",
											},
											&Field{
												Name: "innername5",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			&OperationStmt{ // pointer receiver
				Type: "mutation",
				Name: "yuuo",
				SelectionSet: []SelectionSetI{
					&Field{
						Name: "likeStory",
						Arguments: []*Argument{
							{Name: "storyid", Value: InputValue_{Value: "1223", Type: "Int"}},
						},
						SelectionSet: []SelectionSetI{
							&Field{
								Name: "Name",
								SelectionSet: []SelectionSetI{
									&Field{
										Name: "likeCount",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	fmt.Println(document.String())
	if document.String() == "{ me { Ross } }" {
		t.Errorf("program.String() wrong. got=%q", document.String())
	}

}
