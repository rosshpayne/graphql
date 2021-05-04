package resolver

import (
	"context"
	"fmt"
	"strings"

	sdl "github.com/graphql/internal/graph-sdl/ast"
)

type fieldResolver interface {
	TypeName() string
	String() string
}

// Resolvers

type ResolverFunc func(context.Context, sdl.InputValueProvider, sdl.ObjectVals) <-chan string

type resolverPath string

type Resolvers struct {
	resolverMap map[resolverPath]ResolverFunc
}

func New() *Resolvers {
	return &Resolvers{resolverMap: make(map[resolverPath]ResolverFunc)}
}

func (r Resolvers) Register(path string, f ResolverFunc, override ...bool) error {

	var pathField resolverPath = resolverPath(path)

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

	if f, ok := r.resolverMap[resolverPath(path)]; ok {
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
