// Copyright 2016 Aleksandr Demakin. All rights reserved.

package deadscanner

import (
	"fmt"
	"go/ast"
	"go/token"
)

type identInfo struct {
	node ast.Node
	used bool
}

type context struct {
	decls map[string]identInfo
}

type stack []context

func (stk stack) current() context {
	return stk[len(stk)-1]
}

func (stk stack) top() bool {
	return len(stk) == 1
}

func (stk stack) mark(name string) {
	if name == "_" {
		return
	}
	for i := len(stk) - 1; i >= 0; i-- {
		if info, found := stk[i].decls[name]; found {
			stk[i].decls[name] = identInfo{node: info.node, used: true}
			return
		}
	}
}

func (stk stack) add(name string, node ast.Node, used bool) {
	if name == "_" {
		return
	}
	info := stk.current().decls[name]
	info.node = node
	if !info.used { // do not reset 'used' flag
		info.used = used
	}
	stk.current().decls[name] = info
}

// Report is a record about unused symbol.
type Report struct {
	Pos  token.Pos
	Name string
}

// Reports is a sorted collection of records.
type Reports []Report

func (l Reports) Len() int           { return len(l) }
func (l Reports) Less(i, j int) bool { return l[i].Pos < l[j].Pos }
func (l Reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// Scanner scans for unused symbols in a package
type Scanner struct {
	pkg *ast.Package
}

// New returns new scanner for the given package.
func New(pkg *ast.Package) *Scanner {
	return &Scanner{pkg: pkg}
}

// Do analyzes the package and returns results.
func (s *Scanner) Do() Reports {
	var reports Reports
	main := s.pkg.Name == "main"
	for _, file := range s.pkg.Files {
		nv := &nodeVisitor{main: main}
		nv.walk(file)
		reports = append(reports, nv.reports...)
	}
	return reports
}

type nodeVisitor struct {
	stk     stack
	main    bool
	reports Reports
}

func (nv *nodeVisitor) push() {
	nv.stk = append(nv.stk, context{
		decls: make(map[string]identInfo),
	})
}

func (nv *nodeVisitor) walk(node ast.Node) {
	nv.push()
	ast.Walk(nv, node)
	nv.pop()
}

func (nv *nodeVisitor) pop() {
	cur := nv.stk[len(nv.stk)-1]
	for name, info := range cur.decls {
		if !info.used {
			nv.reports = append(nv.reports, Report{Name: name, Pos: info.node.Pos()})
		}
	}
	nv.stk = nv.stk[:len(nv.stk)-1]
}

func (nv *nodeVisitor) Visit(node ast.Node) ast.Visitor {
	var ret ast.Visitor
	switch node.(type) {
	case *ast.File:
		f := node.(*ast.File)
		for _, decl := range f.Decls {
			ast.Walk(nv, decl)
		}
	case *ast.ValueSpec, *ast.TypeSpec, *ast.GenDecl, *ast.DeclStmt:
		v := &declVisitor{stk: nv.stk, main: nv.main}
		ast.Walk(v, node)
	case *ast.FuncDecl:
		fd := node.(*ast.FuncDecl)
		if fd.Recv == nil { // TODO(avd) - methods
			nv.addFunc(fd.Name.Name, fd)
		}
		ast.Walk(nv, fd.Body)
	case *ast.BlockStmt:
		nv.push()
		b := node.(*ast.BlockStmt)
		for _, stmt := range b.List {
			ast.Walk(nv, stmt)
		}
		nv.pop()
	case *ast.AssignStmt:
		a := node.(*ast.AssignStmt)
		for _, expr := range a.Rhs {
			ast.Walk(nv, expr)
		}
	case *ast.Ident:
		id := node.(*ast.Ident)
		nv.stk.mark(id.Name)
		ret = nv
	case *ast.KeyValueExpr:
		kv := node.(*ast.KeyValueExpr)
		ast.Walk(nv, kv.Value)
	case *ast.CompositeLit:
		t := node.(*ast.CompositeLit)
		fmt.Printf("comp %v %T\n", t, t)
		if t.Type != nil {
			fmt.Printf("  comp type %v %T\n", t.Type, t.Type)
			if id, ok := t.Type.(*ast.Ident); ok {
				println("mark", id.Name)
				nv.stk.mark(id.Name)
			}
		}
		for _, elt := range t.Elts {
			ast.Walk(nv, elt)
			fmt.Printf("visit comp %v %T\n", elt, elt)
		}
	default:
		ret = nv
	}
	return ret
}

func (nv *nodeVisitor) addFunc(name string, node ast.Node) {
	var used bool
	if nv.stk.top() {
		if name == "init" || name == "main" && nv.main || ast.IsExported(name) {
			used = true
		}
	}
	nv.stk.add(name, node, used)
}

type declVisitor struct {
	stk      stack
	main     bool
	forConst bool
}

func (d *declVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.GenDecl:
		d.forConst = n.Tok == token.CONST
	case *ast.ValueSpec:
		if d.forConst {
			for _, name := range n.Names {
				used := d.stk.top() && ast.IsExported(name.Name)
				d.stk.add(name.Name, name, used)
			}
		} else if n.Type != nil {
			if id, ok := n.Type.(*ast.Ident); ok {
				d.stk.mark(id.Name)
			}
		}
	case *ast.TypeSpec:
		d.stk.add(n.Name.Name, node, false)
	case *ast.StructType:
		for _, field := range n.Fields.List {
			if field.Type != nil {
				if id, ok := field.Type.(*ast.Ident); ok {
					d.stk.mark(id.Name)
				}
			}
		}
	}
	return d
}
