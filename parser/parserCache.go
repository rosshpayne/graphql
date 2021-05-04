package parser

import (
	"fmt"

	"github.com/rosshpayne/graphql/ast"
)

// for fragment  & operatinal statments
type entry struct {
	ready chan struct{}
	data  ast.GQLStmtProvider
}

type Cache_ struct {
	//	sync.Mutex
	cache map[string]*entry
}

func newCache() *Cache_ {
	return &Cache_{cache: make(map[string]*entry)}
}

// addEntry is not concurrent safe. Intended for a single thread operation.
func (t *Cache_) addEntry(name ast.StmtName_, stmt ast.GQLStmtProvider) {
	e := &entry{data: stmt}
	t.cache[string(name)] = e
}

// FetchAST - TODO: copy code from sdl.??
func (t *Cache_) fetchAST(name ast.StmtName_) ast.GQLStmtProvider {

	name_ := string(name)
	e, ok := t.cache[name_]

	if !ok {
		return nil
	}

	return e.data

}

func (t *Cache_) cacheClear() {
	fmt.Println("******************************************")
	fmt.Println("************ CLEAR CACHE *****************")
	fmt.Println("******************************************")
	t.cache = make(map[string]*entry)
}
