package parser

import (
	"fmt"

	sdl "github.com/graphql/internal/graph-sdl/ast"
)

func (p *Parser) validateArguments(qArguments *[]*sdl.ArgumentT, argDefs sdl.InputValueDefs, item sdl.Name_, root sdl.GQLTypeProvider) {

	var found bool
	for _, argVal := range *qArguments {
		found = false
		for _, argDef := range argDefs {
			if argVal.Name_.Equals(argDef.Name_) {
				found = true
				// validate argument value against type expected by schema definition
				argVal.Value.CheckInputValueType(argDef.Type, argVal.Name_, &p.perror)
				break
			}
		}
		if !found {
			p.addErr(fmt.Sprintf(`Argument %q is not defined in type %q, %s`, argVal.Name_, root.TypeName(), item.AtPosition()))
			p.abort = true
		}
	}
	//
	// find arguments that are not specified. If schema defined default value then create argument with default value, otherwise error.
	//
	for _, argDef := range argDefs {
		found = false
		for _, argVal := range *qArguments {
			if argVal.Name_.Equals(argDef.Name_) {
				found = true
			}
		}
		if !found {
			if argDef.DefaultVal != nil {
				// create argument with system defaults
				iv := &sdl.ArgumentT{Name_: argDef.Name_, Value: argDef.DefaultVal}
				*qArguments = append(*qArguments, iv)
				for _, v := range *qArguments {
					fmt.Printf("* Argument: %#v\n", *v)
				}

			} else {
				p.addErr(fmt.Sprintf(`Argument %q must be defined (type %q) %s`, argDef.Name_, argDef.Type.String(), item.AtPosition()))
			}
		}
	}
}

//
// Directive_ Definition AST -   @DirName (Name:Value Name:Value . . . ).
//
// Any directive can be used in a QL not necessarily those assoicated with the type in SDL.
//
func (p *Parser) validateDirectives(qDirectives []*sdl.DirectiveT, root sdl.GQLTypeProvider, dirLoc sdl.DirectiveLoc, item sdl.Name_) { //todo - pass in valid location e.g. FIELD, INLINE_FRAGMENT, FRAGMENT_SPREAD

	for _, qDir := range qDirectives {
		// get sdl.Directive_ AST from cache.
		// note:resolveDependents() will have caught non-existent directives, so no need to check for not-exist errors
		sdDirAST, _ := p.tyCache.FetchAST(qDir.Name_.Name)
		if sdDirAST == nil {
			p.abort = true
			return
		}
		sdDir := sdDirAST.(*sdl.Directive_)
		p.validateArguments(&qDir.Arguments, sdDir.ArgumentDefs, item, root)
		//
		var found bool
		for _, loc := range sdDir.Location {
			if loc == dirLoc {
				found = true
				break
			}
		}
		if !found {
			p.addErr(fmt.Sprintf("Directive %q is not defined for %s (see schema doc, %s), %s", qDir.Name_, sdl.DirectiveLocationMap[dirLoc], p.document, qDir.Name_.AtPosition()))
		}
	}
}

func (p *Parser) confirmASTassigned(sdlFld *sdl.Field_) {

	//
	// this should be redundant code - as AST should have already been assigned.
	//
	if !sdlFld.Type.IsScalar() && sdlFld.Type.AST == nil {
		var err error
		p.logr.Println("Assign GQLtype.AST for sdlFld.Type.Name from cache")
		sdlFld.Type.AST, err = p.tyCache.FetchAST(sdlFld.Type.Name)
		if err != nil {
			p.addErr(err.Error())
		}
		if sdlFld.Type.AST == nil {
			p.addErr(fmt.Sprintf("Type %q not found in document %q", sdlFld.Type.Name, p.document))
			p.abort = true
		}
	}
}
