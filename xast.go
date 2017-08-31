package xast

import (
	"fmt"
	"go/ast"
	"reflect"
)

// NewNode returns a new node with the givent parent and ast.Node.
func NewNode(parent *Node, cur ast.Node) *Node {
	return &Node{p: parent, n: cur}
}

// Node holds the current ast.Node and a parent *Node.
type Node struct {
	p      *Node
	n      ast.Node
	delete bool
	skip   bool
}

// Parent returns Parent Node.
func (n *Node) Parent() *Node {
	if n == nil {
		return nil
	}
	return n.p
}

// Node returns the current ast.Node.
func (n *Node) Node() ast.Node {
	if n == nil {
		return nil
	}
	return n.n
}

// SetNode replaces the current ast.Node.
func (n *Node) SetNode(nn ast.Node) *Node {
	if n != nil {
		n.n = nn
	}
	return n
}

// Delete marks the node as nil and returns it, making Walk delete it from its parent.
func (n *Node) Delete() *Node {
	if n != nil {
		n.delete = true
	}
	return n
}

// Break skips the current node from farther processing.
func (n *Node) Break() *Node {
	if n != nil {
		n.skip = true
	}
	return n
}

// Canceled returns true if n is nil or Break/Delete got called.
func (n *Node) Canceled() bool {
	return n == nil || n.skip || n.delete || n.n == nil
}

func (n *Node) assign(dst interface{}) (assigned bool) {
	rv := reflect.ValueOf(dst).Elem()
	if assigned = n != nil && !n.delete && n.n != nil; assigned {
		rv.Set(reflect.ValueOf(n.n))
	} else {
		rv.Set(reflect.Zero(rv.Type()))
	}
	return
}

// WalkFunc describes a function to be called for each node during a Walk.
// The returned node can be used to rewrite the AST.
type WalkFunc func(*Node) *Node

func isNil(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return !rv.IsValid() || rv.IsNil()
}

// Walk calls WalkNode() with the given node and returns the result.
func Walk(root ast.Node, fn WalkFunc) ast.Node {
	return WalkNode(&Node{n: root}, fn).Node()
}

// WalkNode traverses an AST in depth-first order.
// Panics if you call node.SetNode(new-node) the returned type is not the same type as the original one.
func WalkNode(node *Node, fn WalkFunc) *Node {
	if isNil(node.Node()) {
		return node
	}

	if node = fn(node); node.Canceled() {
		return node
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := node.n.(type) {
	case *ast.CommentGroup:
		out := n.List[:0]
		for _, c := range n.List {
			if WalkNode(&Node{p: node, n: c}, fn).assign(&c) {
				out = append(out, c)
			}
		}
		n.List = out

	case *ast.Field:
		n.Names = walkIdentList(node, n.Names, fn)
		if !WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type) {
			return node.Delete()
		}

		WalkNode(&Node{p: node, n: n.Tag}, fn).assign(&n.Tag)
		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)
		WalkNode(&Node{p: node, n: n.Comment}, fn).assign(&n.Comment)

	case *ast.FieldList:
		if len(n.List) == 0 {
			break
		}
		out := n.List[:0]
		for i, f := range n.List {
			if WalkNode(&Node{p: node, n: f}, fn).assign(&f) {
				out = append(out, f)
			} else {
				nukeComments(n.List[i])
			}
		}
		if n.List = out; len(n.List) == 0 {
			return node.Delete()
		}

	case *ast.Ellipsis:
		if !WalkNode(&Node{p: node, n: n.Elt}, fn).assign(&n.Elt) {
			return node.Delete()
		}

	case *ast.FuncLit:
		if !WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type) {
			return node.Delete()
		}

		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.CompositeLit:
		WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type)
		n.Elts = walkExprList(node, n.Elts, fn)

	case *ast.ParenExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)

	case *ast.SelectorExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Sel}, fn).assign(&n.Sel)

	case *ast.IndexExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Index}, fn).assign(&n.Index)

	case *ast.SliceExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Low}, fn).assign(&n.Low)
		WalkNode(&Node{p: node, n: n.High}, fn).assign(&n.High)
		WalkNode(&Node{p: node, n: n.Max}, fn).assign(&n.Max)

	case *ast.TypeAssertExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type)

	case *ast.CallExpr:
		if !WalkNode(&Node{p: node, n: n.Fun}, fn).assign(&n.Fun) {
			return node.Delete()
		}
		n.Args = walkExprList(node, n.Args, fn)

	case *ast.StarExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)

	case *ast.UnaryExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)

	case *ast.BinaryExpr:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Y}, fn).assign(&n.Y)

	case *ast.KeyValueExpr:
		WalkNode(&Node{p: node, n: n.Key}, fn).assign(&n.Key)
		WalkNode(&Node{p: node, n: n.Value}, fn).assign(&n.Value)

	case *ast.ArrayType:
		WalkNode(&Node{p: node, n: n.Len}, fn).assign(&n.Len)
		if !WalkNode(&Node{p: node, n: n.Elt}, fn).assign(&n.Elt) {
			return node.Delete()
		}

	case *ast.StructType:
		if !WalkNode(&Node{p: node, n: n.Fields}, fn).assign(&n.Fields) {
			return node.Delete()
		}

	case *ast.FuncType:
		WalkNode(&Node{p: node, n: n.Params}, fn).assign(&n.Params)
		WalkNode(&Node{p: node, n: n.Results}, fn).assign(&n.Results)

	case *ast.InterfaceType:
		WalkNode(&Node{p: node, n: n.Methods}, fn).assign(&n.Methods)

	case *ast.MapType:
		if !WalkNode(&Node{p: node, n: n.Key}, fn).assign(&n.Key) {
			return node.Delete()
		}
		if !WalkNode(&Node{p: node, n: n.Value}, fn).assign(&n.Value) {
			return node.Delete()
		}

	case *ast.ChanType:
		if !WalkNode(&Node{p: node, n: n.Value}, fn).assign(&n.Value) {
			return node.Delete()
		}

	case *ast.DeclStmt:
		if !WalkNode(&Node{p: node, n: n.Decl}, fn).assign(&n.Decl) {
			return node.Delete()
		}

	case *ast.LabeledStmt:
		WalkNode(&Node{p: node, n: n.Label}, fn).assign(&n.Label)
		WalkNode(&Node{p: node, n: n.Stmt}, fn).assign(&n.Stmt)

	case *ast.ExprStmt:
		if !WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X) {
			return node.Delete()
		}

	case *ast.SendStmt:
		WalkNode(&Node{p: node, n: n.Chan}, fn).assign(&n.Chan)
		WalkNode(&Node{p: node, n: n.Value}, fn).assign(&n.Value)

	case *ast.IncDecStmt:
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)

	case *ast.AssignStmt:
		n.Lhs = walkExprList(node, n.Lhs, fn)
		n.Rhs = walkExprList(node, n.Rhs, fn)

	case *ast.GoStmt:
		WalkNode(&Node{p: node, n: n.Call}, fn).assign(&n.Call)

	case *ast.DeferStmt:
		WalkNode(&Node{p: node, n: n.Call}, fn).assign(&n.Call)

	case *ast.ReturnStmt:
		n.Results = walkExprList(node, n.Results, fn)

	case *ast.BranchStmt:

		WalkNode(&Node{p: node, n: n.Label}, fn).assign(&n.Label)

	case *ast.BlockStmt:
		n.List = walkStmtList(node, n.List, fn)

	case *ast.IfStmt:
		WalkNode(&Node{p: node, n: n.Init}, fn).assign(&n.Init)
		WalkNode(&Node{p: node, n: n.Cond}, fn).assign(&n.Cond)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)
		WalkNode(&Node{p: node, n: n.Else}, fn).assign(&n.Else)

	case *ast.CaseClause:
		n.List = walkExprList(node, n.List, fn)
		n.Body = walkStmtList(node, n.Body, fn)

	case *ast.SwitchStmt:

		WalkNode(&Node{p: node, n: n.Init}, fn).assign(&n.Init)
		WalkNode(&Node{p: node, n: n.Tag}, fn).assign(&n.Tag)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.TypeSwitchStmt:
		WalkNode(&Node{p: node, n: n.Init}, fn).assign(&n.Init)
		WalkNode(&Node{p: node, n: n.Assign}, fn).assign(&n.Assign)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.CommClause:
		WalkNode(&Node{p: node, n: n.Comm}, fn).assign(&n.Comm)
		n.Body = walkStmtList(node, n.Body, fn)

	case *ast.SelectStmt:
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.ForStmt:
		WalkNode(&Node{p: node, n: n.Init}, fn).assign(&n.Init)
		WalkNode(&Node{p: node, n: n.Cond}, fn).assign(&n.Cond)
		WalkNode(&Node{p: node, n: n.Post}, fn).assign(&n.Post)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.RangeStmt:
		WalkNode(&Node{p: node, n: n.Key}, fn).assign(&n.Key)
		WalkNode(&Node{p: node, n: n.Value}, fn).assign(&n.Value)
		WalkNode(&Node{p: node, n: n.X}, fn).assign(&n.X)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)

	case *ast.ImportSpec:
		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)
		WalkNode(&Node{p: node, n: n.Name}, fn).assign(&n.Name)
		WalkNode(&Node{p: node, n: n.Path}, fn).assign(&n.Path)
		WalkNode(&Node{p: node, n: n.Comment}, fn).assign(&n.Comment)

	case *ast.ValueSpec:
		n.Names = walkIdentList(node, n.Names, fn)
		WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type)
		n.Values = walkExprList(node, n.Values, fn)
		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)
		WalkNode(&Node{p: node, n: n.Comment}, fn).assign(&n.Comment)

	case *ast.TypeSpec:
		WalkNode(&Node{p: node, n: n.Name}, fn)
		WalkNode(&Node{p: node, n: n.Type}, fn)
		WalkNode(&Node{p: node, n: n.Comment}, fn).assign(&n.Comment)

	case *ast.GenDecl:
		if n.Specs = walkSpecList(node, n.Specs, fn); len(n.Specs) == 0 {
			return node.Delete()
		}

		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)
	case *ast.FuncDecl:
		if !WalkNode(&Node{p: node, n: n.Recv}, fn).assign(&n.Recv) {
			return node.Delete()
		}
		WalkNode(&Node{p: node, n: n.Name}, fn).assign(&n.Name)
		WalkNode(&Node{p: node, n: n.Type}, fn).assign(&n.Type)
		WalkNode(&Node{p: node, n: n.Body}, fn).assign(&n.Body)
		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)

	case *ast.File:
		WalkNode(&Node{p: node, n: n.Doc}, fn).assign(&n.Doc)
		WalkNode(&Node{p: node, n: n.Name}, fn).assign(&n.Name)
		n.Decls = walkDeclList(node, n.Decls, fn)

		// don't walk n.Comments - they have been
		// visited already through the individual
		// nodes

	case *ast.Package:
		for _, f := range n.Files {
			WalkNode(&Node{p: node, n: f}, fn).assign(&f)
		}

	case *ast.BadStmt, *ast.BadDecl, *ast.BadExpr, *ast.Ident,
		*ast.BasicLit, *ast.Comment, *ast.EmptyStmt:
		// nothing to do

	default:
		panic(fmt.Sprintf("ast.Walk: unexpected node type %T", n))
	}

	return node
}

func nukeComments(root ast.Node) {
	if root == nil {
		return
	}
	ast.Inspect(root, func(n ast.Node) bool {
		if isNil(n) {
			return false
		}

		switch n := n.(type) {
		case *ast.CommentGroup:
			n.List = nil
			return false
		case nil:
			return false
		default:
			return true
		}
	})
}

func walkIdentList(node *Node, list []*ast.Ident, fn WalkFunc) (out []*ast.Ident) {
	out = list[:0]
	for i, x := range list {
		if WalkNode(&Node{p: node, n: x}, fn).assign(&x) {
			out = append(out, x)
		} else {
			nukeComments(list[i])
		}
	}
	return
}

func walkExprList(node *Node, list []ast.Expr, fn WalkFunc) (out []ast.Expr) {
	out = list[:0]
	for i, x := range list {
		if WalkNode(&Node{p: node, n: x}, fn).assign(&x) {
			out = append(out, x)
		} else {
			nukeComments(list[i])
		}
	}
	return
}

func walkStmtList(node *Node, list []ast.Stmt, fn WalkFunc) (out []ast.Stmt) {
	out = list[:0]
	for i, x := range list {
		if WalkNode(&Node{p: node, n: x}, fn).assign(&x) {
			out = append(out, x)
		} else {
			nukeComments(list[i])
		}
	}
	return
}

func walkDeclList(node *Node, list []ast.Decl, fn WalkFunc) (out []ast.Decl) {
	out = list[:0]
	for i, x := range list {
		if WalkNode(&Node{p: node, n: x}, fn).assign(&x) {
			out = append(out, x)
		} else {
			nukeComments(list[i])
		}
	}
	return
}

func walkSpecList(node *Node, list []ast.Spec, fn WalkFunc) (out []ast.Spec) {
	out = list[:0]
	for i, x := range list {
		if WalkNode(&Node{p: node, n: x}, fn).assign(&x) {
			out = append(out, x)
		} else {
			nukeComments(list[i])
		}
	}

	return
}
