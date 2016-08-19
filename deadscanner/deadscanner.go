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
			println("mark", name)
			stk[i].decls[name] = identInfo{node: info.node, used: true}
			return
		}
	}
	//panic("undeclarated identifier " + name)
}

type Report struct {
	Pos  token.Pos
	Name string
}

type Reports []Report

func (l Reports) Len() int           { return len(l) }
func (l Reports) Less(i, j int) bool { return l[i].Pos < l[j].Pos }
func (l Reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

type Scanner struct {
	pkg *ast.Package
}

func New(pkg *ast.Package) *Scanner {
	return &Scanner{pkg: pkg}
}

func (s *Scanner) Do() Reports {
	var reports Reports
	main := s.pkg.Name == "main"
	for _, file := range s.pkg.Files {
		// walk file looking for used nodes.
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
			fmt.Printf("unused: %s\n", name)
			nv.reports = append(nv.reports, Report{Name: name, Pos: info.node.Pos()})
		}
	}
	nv.stk = nv.stk[:len(nv.stk)-1]
}

func (nv *nodeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	fmt.Printf("node %v %T\n", node, node)
	switch node.(type) {
	case *ast.File:
		f := node.(*ast.File)
		for _, decl := range f.Decls {
			ast.Walk(nv, decl)
		}
		return nil
	case *ast.ValueSpec, *ast.TypeSpec, *ast.GenDecl, *ast.DeclStmt:
		v := &declVisitor{stk: nv.stk, main: nv.main}
		ast.Walk(v, node)
		return nil
	case *ast.FuncDecl:
		println("func decl")
		fd := node.(*ast.FuncDecl)
		nv.addFunc(fd.Name.Name, fd)
		ast.Walk(nv, fd.Body)
		return nil
	case *ast.BlockStmt:
		nv.push()
		println("PUSH")
		b := node.(*ast.BlockStmt)
		for _, stmt := range b.List {
			ast.Walk(nv, stmt)
		}
		println("POP")
		nv.pop()
		return nil
	case *ast.AssignStmt:
		a := node.(*ast.AssignStmt)
		for _, expr := range a.Rhs {
			ast.Walk(nv, expr)
		}
		return nil
	case *ast.Ident:
		id := node.(*ast.Ident)
		nv.stk.mark(id.Name)
	}
	return nv
}

func (nv *nodeVisitor) addFunc(name string, node ast.Node) {
	var used bool
	if nv.stk.top() {
		if ast.IsExported(name) {
			used = true
		} else if name == "init" {
			used = true
		} else if name == "main" && nv.main {
			used = true
		}
	}
	nv.stk.current().decls[name] = identInfo{node: node, used: used}
}

type declVisitor struct {
	main     bool
	stk      stack
	forConst bool
}

func (d *declVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	fmt.Printf("  decl %v %T\n", node, node)
	switch n := node.(type) {
	case *ast.GenDecl:
		d.forConst = n.Tok == token.CONST
	case *ast.DeclStmt:
		println("  decl2")
	case *ast.ValueSpec:
		fmt.Println("  value spec")
		if d.forConst {
			for _, name := range n.Names {
				println("  new", name.Name)
				used := d.stk.top() && ast.IsExported(name.Name)
				d.stk.current().decls[name.Name] = identInfo{node: name, used: used}
			}
		} else if n.Type != nil {
			if id, ok := n.Type.(*ast.Ident); ok {
				d.stk.mark(id.Name)
			}
		}
	case *ast.TypeSpec:
		d.stk.current().decls[n.Name.Name] = identInfo{node: node}
	}
	return d
}
