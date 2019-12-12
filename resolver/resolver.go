package resolver

import (
	"fmt"
	"strings"

	sdl "github.com/graph-sdl/ast"
)

type fieldResolver interface {
	TypeName() string
	String() string
}

// Resolvers

type ResolverFunc func(obj sdl.InputValueProvider, args sdl.ObjectVals) string

type ResolverPath string

type Resolvers struct {
	resolverMap map[ResolverPath]ResolverFunc
}

func New() *Resolvers {
	return &Resolvers{resolverMap: make(map[ResolverPath]ResolverFunc)}
}

func (r Resolvers) Register(path string, f ResolverFunc, override ...bool) error {

	var pathField ResolverPath

	pathField = ResolverPath(path)

	if _, ok := r.resolverMap[pathField]; !ok {
		r.resolverMap[pathField] = f
		return nil
	}
	if len(override) > 0 && override[0] {
		r.resolverMap[pathField] = f
		return nil
	}
	return fmt.Errorf(`Resolver function already registered against path "%s"`, pathField)

}

func (r Resolvers) GetFunc(path string) ResolverFunc {

	if f, ok := r.resolverMap[ResolverPath(path)]; ok {
		return f
	}
	return nil

}

func (r Resolvers) String() string {
	var s strings.Builder

	for k := range r.resolverMap {
		s.WriteString(string(k))
		s.WriteString("\n")
	}
	return s.String()

}
