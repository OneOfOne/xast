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
	src := `
package main

// Foo is a foo!
type Foo struct{}

// NotFoo is not a foo!
type NotFoo struct{}

// DeleteMe needs to be deleted with this comment.
func DeleteMe() {}

// DeleteMeToo says hi.
func DeleteMeToo() {}

// GoodBoy is a good boy, yes he is!
func GoodBoy() {
	var nf NotFoo
	_ = nf
}
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
			if _, ok := n.Parent().Node().(*ast.GenDecl); ok {
				x.List = nil
				return n.Break() // won't delete the node but Walk won't go down its children list.
			}
		case *ast.FuncDecl:
			switch x.Name.Name {
			case "DeleteMe", "DeleteMeToo":
				return n.Delete()
			case "GoodBoy":
				x.Doc.List = nil // remove the goodboy's comment :-/
			}
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
	//
	// func GoodBoy() {
	// 	var nf NotFoo
	// 	_ = nf
	// }
}
