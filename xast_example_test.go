package xast_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"github.com/OneOfOne/xast"
)

func ExampleWalk() {
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
			return n.Delete() // delete this node

		}

		return n
	}

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, xast.Walk(file, rewriteFn))
	fmt.Println(buf.String())
	// Output:
	// package main
	//
	// type Bar struct{}
	//
	// type NotFoo struct{}
}
