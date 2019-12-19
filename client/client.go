package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdl "github.com/graph-sdl/ast"
)

type Person struct {
	id    int
	name  string
	age   [][]interface{}
	other []string
	posts []int
}

func (p *Person) String() string {
	var s strings.Builder
	s.WriteString("{\n")
	s.WriteString(` name : "`)
	s.WriteString(p.name)
	s.WriteString(`"`)
	s.WriteString("\n age: [")
	for _, v := range p.age {
		s.WriteString("[")
		for i, v2 := range v {
			switch x := v2.(type) {
			case int:
				s.WriteString(strconv.Itoa(x))
			case string:
				if x == "null" {
					s.WriteString(x)
				} else {
					s.WriteString(fmt.Sprintf("%q", x))
				}
			}
			if i < len(v)-1 {
				s.WriteString(" ")
			}
		}
		s.WriteString("] ")
	}
	s.WriteString("]\n")
	s.WriteString("other : [")
	for _, v := range p.other {
		s.WriteString(`"`)
		s.WriteString(v)
		s.WriteString(`" `)
	}
	s.WriteString(" ]\n")
	s.WriteString(" posts : [")
	for _, v := range p.posts {
		//s.WriteString(strconv.Itoa(v) + " ")
		s.WriteString(posts[v-1].String())
	}
	s.WriteString(" ]\n")
	s.WriteString("}\n")
	return s.String()
}

func (p *Person) ShortString() string {
	var s strings.Builder
	s.WriteString("{")
	s.WriteString(`name : "`)
	s.WriteString(p.name)
	s.WriteString(`"`)
	s.WriteString(" ")
	s.WriteString("\n age: [")
	for _, v := range p.age {
		s.WriteString("[")
		for i, v2 := range v {
			switch x := v2.(type) {
			case int:
				s.WriteString(strconv.Itoa(x))
			case string:
				if x == "null" {
					s.WriteString(x)
				} else {
					s.WriteString(fmt.Sprintf("%q", x))
				}
			}
			if i < len(v)-1 {
				s.WriteString(" ")
			}
		}
		s.WriteString("] ")
	}
	s.WriteString("]\n")
	s.WriteString(` }`)
	return s.String()
}

func (p *Person) StringPartial() string {
	var s strings.Builder
	s.WriteString("{\n")
	s.WriteString(` name : "`)
	s.WriteString(p.name)
	s.WriteString(`"`)
	s.WriteString("\n age: [")
	for _, v := range p.age {
		s.WriteString("[")
		for i, v2 := range v {
			switch x := v2.(type) {
			case int:
				s.WriteString(strconv.Itoa(x))
			case string:
				if x == "null" {
					s.WriteString(x)
				} else {
					s.WriteString(fmt.Sprintf("%q", x))
				}
			}
			if i < len(v)-1 {
				s.WriteString(" ")
			}
		}
		s.WriteString("] ")
	}
	s.WriteString("]\n")
	s.WriteString("\n")
	s.WriteString("other : [")
	for _, v := range p.other {
		s.WriteString(`"`)
		s.WriteString(v)
		s.WriteString(`" `)
	}
	s.WriteString(" ]\n")
	s.WriteString(" posts : [")
	for _, v := range p.posts {
		s.WriteString(strconv.Itoa(v) + " ")
		//s.WriteString(v.String())
	}
	s.WriteString(" ]\n")
	s.WriteString("}\n")
	return s.String()
}

type Post struct {
	id     int
	title  string
	author int
}

func (p *Post) String() string {
	var s strings.Builder
	s.WriteString("\n")
	if len(p.title) > 0 {
		s.WriteString(`	{ title : "`)
		s.WriteString(p.title)
	}
	s.WriteString(`"	 author : [`)
	s.WriteString(persons[p.author-100].ShortString())
	s.WriteString("]	}")
	return s.String()
}

var persons = []*Person{
	&Person{100, "Jack Smith", [][]interface{}{[]interface{}{53, 54, 55, 56}, []interface{}{25, 26, 28, 27}}, []string{"abc", "def", "hij"}, []int{1, 2, 3}},
	&Person{101, "Jenny Hawk", [][]interface{}{[]interface{}{25, 26, 27}, []interface{}{44, 45, 46}}, []string{"aaaaabc", "def", "hij"}, []int{3, 7, 4}},
	&Person{102, "Sabastian Jackson", [][]interface{}{[]interface{}{44, 45, 46}, []interface{}{54, 55, 56, 57}}, []string{"123", "def", "hij"}, nil},
	&Person{103, "Phillip Coats", [][]interface{}{[]interface{}{54, 55, 56, 57}}, []string{"xyz", "def", "hij"}, nil},
	&Person{104, "Kathlyn Host", [][]interface{}{[]interface{}{33, 32, 31}, []interface{}{33, 32, 31}}, []string{"abasdc", "def", "hij"}, []int{5}},
}
var posts = []*Post{
	&Post{1, "GraphQL for Begineers", 100}, &Post{2, "Holidays in Tuscany", 101}, &Post{3, "Sweet", 102}, &Post{4, "Programming in GO", 102}, &Post{5, "Skate Boarding Blog", 101},
	&Post{6, "GraphQL for Architects", 100}, &Post{id: 7, title: "xyz", author: 100},
}

var ResolverAll = func(ctx context.Context, resp sdl.InputValueProvider, args sdl.ObjectVals) <-chan string {

	f := func() string {
		var s strings.Builder
		var last_ int = 2
		var err error
		fmt.Println(args.String())
		if len(args) > 0 {
			if args[0].Name.EqualString("last") {
				last := args[0].Value.InputValueProvider.(sdl.Int_)
				fmt.Println("Limited to: ", string(last))
				if last_, err = strconv.Atoi(string(last)); err != nil {
					fmt.Println(err)
				}
			}
		}
		s.WriteString("{data: [")
		for i, v := range persons {
			if i > last_-1 {
				break
			}
			s.WriteString(v.String())
		}
		s.WriteString("]}")
		return s.String()
	}

	gql := make(chan string)
	go func() {
		select {
		case <-ctx.Done():
			return
		case gql <- f(): // gql channel unblocks immediately when calling routine (GraphQL server) starts listening on channel
			return
		}
	}()

	return gql
}

var ResolvePartial = func(ctx context.Context, resp sdl.InputValueProvider, args sdl.ObjectVals) <-chan string {

	f := func() string {
		var s strings.Builder
		var last_ int = 2
		var err error
		if len(args) > 0 {
			if args[0].Name.EqualString("last") {
				last := args[0].Value.InputValueProvider.(sdl.Int_)
				fmt.Println("Limited to: ", string(last))
				if last_, err = strconv.Atoi(string(last)); err != nil {
					fmt.Println(err)
				}
			}
		}
		s.WriteString("{data: [")
		for i, v := range persons {
			if i > last_-1 {
				break
			}
			s.WriteString(v.StringPartial())
		}
		s.WriteString("] }")
		return s.String()
	}

	gql := make(chan string)
	go func() {
		select {
		case <-ctx.Done():
			return
		case gql <- f(): // gql channel unblocks immediately when calling routine (GraphQL server) starts listening on channel
			return
		}
	}()

	return gql
}

var ResolvePosts = func(ctx context.Context, resp sdl.InputValueProvider, args sdl.ObjectVals) <-chan string {

	f := func() string {
		var s strings.Builder

		for _, v := range args {
			if v.Name_.EqualString("resp") {
				resp := v.Value.InputValueProvider
				switch x := resp.(type) {
				case sdl.List_:
					if len(x) > 0 {
						s.WriteString("{data: [")
					}
					for i, v := range x {
						k := string(v.InputValueProvider.(sdl.Int_))
						if ki, err := strconv.Atoi(k); err != nil {
							fmt.Println(err.Error())
						} else {
							s.WriteString(posts[ki-1].String())
						}
						if i < len(x)-1 {
							s.WriteString(",")
						}
					}
					if len(x) > 0 {
						s.WriteString(" ] }")
					}
				}
			}
		}

		return s.String()
	}

	gql := make(chan string)
	go func() {
		select {
		case <-ctx.Done():
			return
		case gql <- f(): // gql channel unblocks immediately when calling routine (GraphQL server) starts listening on channel
			return
		}
	}()

	return gql
}

var ResolveAge = func(ctx context.Context, resp sdl.InputValueProvider, args sdl.ObjectVals) <-chan string {
	var (
		s     strings.Builder
		scale float32
	)

	for _, v := range args {
		if v.Name_.EqualString("ScaleBy") {
			if x, ok := v.Value.InputValueProvider.(sdl.Float_); ok {
				if num, err := strconv.ParseFloat(string(x), 32); err != nil {
					fmt.Println("error ", err.Error())
				} else {
					scale = float32(num)
				}
			}
		}
	}

	fx := func() string {
		resp_ := resp.(sdl.List_)

		var f func(sdl.List_)

		f = func(y sdl.List_) {

			for i := 0; i < len(y); i++ {
				if x, ok := y[i].InputValueProvider.(sdl.List_); ok {
					s.WriteString("[")
					f(x)
					s.WriteString("]")
				} else {
					// optimise by performing loop here rather than use outer for loop
					for i := 0; i < len(y); i++ {
						switch x := y[i].InputValueProvider.(type) {
						case sdl.Float_:
							if num, err := strconv.ParseFloat(string(x), 32); err != nil {
								fmt.Println("error ", err.Error())
							} else {
								result := scale * float32(num)
								s.WriteString(strconv.FormatFloat(float64(result), 'f', -1, 32))
							}
						case sdl.Int_:
							if num, err := strconv.Atoi(string(x)); err != nil {
								fmt.Println("error ", err.Error())
							} else {
								result := scale * float32(num)
								s.WriteString(strconv.FormatFloat(float64(result), 'f', -1, 32))
							}
						default:
							s.WriteString("NoValue")
						}
						if i < len(y)-1 {
							s.WriteString(",")
						}
					}
					break
				}
			}
		}
		s.WriteString("[") // alternatively, "{age: ["
		f(resp_)
		s.WriteString("]") // "]}"
		return s.String()
	}

	gql := make(chan string)
	go func() {
		select {
		case <-ctx.Done():
			return
		case gql <- fx(): // gql channel unblocks immediately when calling routine (GraphQL server) starts listening on channel
			return
		}
	}()

	return gql

}

type Starship struct {
	name   string
	length float32
}

func (ss Starship) String() string {
	var s strings.Builder

	s.WriteString(`{ `)
	s.WriteString(`name: "`)
	s.WriteString(ss.name)
	s.WriteString(`"`)
	s.WriteString(`, length: `)
	s.WriteString(strconv.FormatFloat(float64(ss.length), 'g', -1, 32))
	s.WriteString("}")
	return s.String()

}

var starships []*Starship = []*Starship{
	&Starship{name: "Falcon", length: 23.4},
	&Starship{name: "Cruiser", length: 68.2},
	&Starship{name: "BattleStar", length: 138.2},
}

type Character struct {
	i    int
	id   string
	name string
}

func (c Character) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("id: %q, name: %q", c.id, c.name))
	return s.String()
}

var characters = []*Character{
	&Character{i: 1, id: "wejnJ3", name: "Han Solo"},
	&Character{i: 2, id: "xjnJ4", name: "Leia Organa"},
	&Character{i: 3, id: "sjnJ5", name: "C-3PO"},
	&Character{i: 4, id: "ksejnJ6", name: "R2-D2"},
	&Character{i: 5, id: "lwewJ6", name: "Luke Skywalker"},
}

type Episode string

func (e Episode) String() string {
	// eval input type - so not quoted values
	return string(fmt.Sprintf("%s ", string(e)))
}

var episodes []Episode = []Episode{"NEWHOPE", "EMPIRE", "JEDI"}

type Human struct {
	i            int
	id           string
	name         string
	friends      []int
	appearsIn    []int
	starships    []int
	totalCredits int
}

func (h *Human) String() string {
	var s strings.Builder
	s.WriteString("{")
	s.WriteString(`id: "`)
	s.WriteString(h.id)
	s.WriteString(`"`)
	s.WriteString(`, name: "`)
	s.WriteString(h.name)
	s.WriteString(`"`)
	s.WriteString(`, friends: [`)
	for i, v := range h.friends {
		s.WriteString("{")
		s.WriteString(characters[v-1].String())
		s.WriteString("}")
		if i < len(h.friends)-1 {
			s.WriteString(`,`)
		}
	}
	s.WriteString(`]`)
	s.WriteString(`, appearsIn: [`)
	for i, v := range h.appearsIn {
		s.WriteString(episodes[v].String())
		if i < len(h.appearsIn)-1 {
			s.WriteString(`,`)
		}
	}
	s.WriteString(`]`)
	s.WriteString(`, starships: [`)
	for i, v := range h.starships {
		s.WriteString(starships[v-1].String())
		if i < len(h.starships)-1 {
			s.WriteString(`,`)
		}
	}
	s.WriteString(`] `)
	s.WriteString(", totalCredits: ")
	s.WriteString(strconv.Itoa(h.totalCredits))
	s.WriteString(" }")
	return s.String()
}

type Droid struct {
	i               int
	id              string
	name            string
	friends         []int
	appearsIn       []int
	primaryFunction string
}

func (d *Droid) String() string {
	var s strings.Builder
	s.WriteString("{")
	s.WriteString(`id: "`)
	s.WriteString(d.id)
	s.WriteString(`"`)
	s.WriteString(`,name: "`)
	s.WriteString(d.name)
	s.WriteString(`"`)
	s.WriteString(`, friends: [`)
	for i, v := range d.friends {
		characters[v-1].String()
		if i < len(d.friends)-1 {
			s.WriteString(`,`)
		}
	}
	s.WriteString(`]`)
	s.WriteString(`, appearsIn: [`)
	for i, v := range d.appearsIn {
		episodes[v].String()
		if i < len(d.appearsIn)-1 {
			s.WriteString(`,`)
		}
	}
	s.WriteString(`, primaryFunction : `)
	s.WriteString(d.primaryFunction)
	s.WriteString("}")
	return s.String()
}

// type Query {
//   hero(episode: Episode): Character
//   droid(id: ID!): Droid
// }

var humans = []*Human{
	&Human{i: 1, id: "jklw2ike", name: "Luke Skywalker", friends: []int{2, 3, 4}, appearsIn: []int{0, 2}, starships: []int{1, 2}, totalCredits: 5532},
	&Human{i: 2, id: "dfw23e", name: "Leia Organa", friends: []int{5, 3, 4}, appearsIn: []int{0, 1}, starships: []int{3}, totalCredits: 2532},
}

var droid = []*Droid{
	&Droid{i: 1, id: "Ljeiike", name: "C-3PO", friends: []int{2, 3, 4}, appearsIn: []int{0, 1, 2}, primaryFunction: "Diplomat"},
	&Droid{i: 2, id: "ewxdfw23e", name: "R2-D2", friends: []int{5, 3, 4}, appearsIn: []int{0, 1, 2}, primaryFunction: "Multifunction"},
}

var ResolverHero = func(ctx context.Context, resp sdl.InputValueProvider, args sdl.ObjectVals) <-chan string {
	var (
		episode string
		index   int
		s       strings.Builder
	)

	f := func() string {
		for _, v := range args {

			if v.Name_.EqualString("episode") {
				if x, ok := v.Value.InputValueProvider.(*sdl.EnumValue_); ok {
					episode = x.String()
				}
			}
		}
		for i, v := range episodes {
			if strings.ToUpper(episode) == string(v) {
				index = i
			}
		}
		//	s.WriteString("[" + fmt.Sprintf("%d", index) + " " + episode)

		//s.WriteString("{Droid:  [")
		s.WriteString("{Human:  [") // becomes respType in parser executeStmt_(). When type of response is an Interface then "on Human" & "on Droid" will use respType to determine which to use.

		for _, v := range humans {
			var found bool
			for _, k := range v.appearsIn {
				if k == index {
					found = true
				}
			}
			if found {
				s.WriteString(v.String())
				s.WriteString(",")
			}
		}
		s.WriteString("] }")
		time.Sleep(650 * time.Millisecond)
		return s.String()
	}

	gql := make(chan string)
	go func() {
		select {
		case <-ctx.Done():
			return
		case gql <- f(): // gql channel unblocks and executes f() immediately when GraphQL server starts listening on channel
			return
		}
	}()

	return gql
}
