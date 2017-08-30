# xast [![GoDoc](http://godoc.org/github.com/OneOfOne/genx?status.svg)](http://godoc.org/github.com/OneOfOne/xast) [![Build Status](https://travis-ci.org/OneOfOne/genx.svg?branch=master)](https://travis-ci.org/OneOfOne/xast)

xast provides a `Walk()` function, similar to [`astrewrite.Walk`](https://godoc.org/github.com/fatih/astrewrite#example-Walk) from the
[astrewrite](https://github.com/fatih/astrewrite) package. The main difference is that the passed walk function can also
check a node's parent.

# Example

```go
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"github.com/OneOfOne/xast"
)

func main() {
	src := `package main

// Foo is a foo!
type Foo struct{}

// NotFoo is not a foo!
type NotFoo struct{}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	rewriteFn := func(n *xast.Node) *xast.Node {
		switch x := n.Node().(type) {
		case *ast.TypeSpec:
			if x.Name.Name == "Foo" {
				x.Name.Name = "Bar"
			}
		case *ast.CommentGroup:
			// check n.Text()?
			x.List = nil
			return n.Break() // don't delete the node but don't go down it's children list.

		// or if you want to remove a single comment out of a group
		case *ast.Comment: // won't ever get here since we return n.Break() from case *ast.CommentGroup.
			return n.Nil() // delete this node

		}

		return n
	}

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, xast.Walk(file, rewriteFn))
	fmt.Println(buf.String())
}
```

