// Copyright 2016 Aleksandr Demakin. All rights reserved.

package deadscanner

import (
	"go/ast"
	"go/token"
	"sort"
)

type identInfo struct {
	node ast.Node
	used bool
}

type context struct {
	decls map[string]identInfo
}

type stack struct {
	ctx []context
}

func (stk stack) current() context {
	return stk.ctx[len(stk.ctx)-1]
}

func (stk stack) top() bool {
	return len(stk.ctx) == 1
}

func (stk *stack) pop() {
	stk.ctx = stk.ctx[:len(stk.ctx)-1]
}

func (stk *stack) push() {
	stk.ctx = append(stk.ctx, context{
		decls: make(map[string]identInfo),
	})
}

func (stk stack) mark(name string) bool {
	if name == "_" {
		return true
	}
	for i := len(stk.ctx) - 1; i >= 0; i-- {
		if info, found := stk.ctx[i].decls[name]; found {
			stk.ctx[i].decls[name] = identInfo{node: info.node, used: true}
			return true
		}
	}
	return false
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
	var allFiles []string
	undeclarated := make(map[string]struct{})
	main := s.pkg.Name == "main"
	for name := range s.pkg.Files {
		allFiles = append(allFiles, name)
	}
	sort.Strings(allFiles)
	for _, name := range allFiles {
		file := s.pkg.Files[name]
		nv := &nodeVisitor{main: main, undeclarated: make(map[string]struct{})}
		nv.walk(file)
		for name := range nv.undeclarated {
			undeclarated[name] = struct{}{}
		}
		reports = append(reports, nv.reports...)
	}
	reports = s.checkGlobals(reports, undeclarated)
	return reports
}

func (s *Scanner) checkGlobals(reports Reports, undeclarated map[string]struct{}) Reports {
	tmp := reports[:0]
	for _, rep := range reports {
		if _, found := undeclarated[rep.Name]; !found {
			tmp = append(tmp, rep)
		}
	}
	return tmp
}

type nodeVisitor struct {
	stk          stack
	main         bool
	reports      Reports
	undeclarated map[string]struct{}
}

func (nv *nodeVisitor) walk(node ast.Node) {
	nv.stk.push()
	ast.Walk(nv, node)
	nv.pop()
}

func (nv *nodeVisitor) pop() {
	cur := nv.stk.current()
	for name, info := range cur.decls {
		if !info.used {
			nv.reports = append(nv.reports, Report{Name: name, Pos: info.node.Pos()})
		}
	}
	nv.stk.pop()
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
		v := &declVisitor{stk: nv.stk, main: nv.main, undeclarated: nv.undeclarated}
		ast.Walk(v, node)
	case *ast.FuncDecl:
		fd := node.(*ast.FuncDecl)
		if fd.Recv == nil { // TODO(avd) - methods
			nv.addFunc(fd.Name.Name, fd)
		}
		inspectFields(fd.Type.Params, &nv.stk, nv.undeclarated)
		inspectFields(fd.Type.Results, &nv.stk, nv.undeclarated)
		ast.Walk(nv, fd.Body)
	case *ast.BlockStmt:
		nv.stk.push()
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
		if !nv.stk.mark(id.Name) {
			nv.undeclarated[id.Name] = struct{}{}
		}
		ret = nv
	case *ast.KeyValueExpr:
		kv := node.(*ast.KeyValueExpr)
		ast.Walk(nv, kv.Value)
	case *ast.CompositeLit:
		t := node.(*ast.CompositeLit)
		if t.Type != nil {
			if id, ok := t.Type.(*ast.Ident); ok {
				if !nv.stk.mark(id.Name) {
					nv.undeclarated[id.Name] = struct{}{}
				}
			}
		}
		for _, elt := range t.Elts {
			ast.Walk(nv, elt)
		}
	default:
		ret = nv
	}
	return ret
}

func (nv *nodeVisitor) addFunc(name string, node ast.Node) {
	var used bool
	if nv.stk.top() {
		if name == "init" || name == "main" && nv.main || !nv.main && ast.IsExported(name) {
			used = true
		}
	}
	nv.stk.add(name, node, used)
}

type declVisitor struct {
	stk          stack
	main         bool
	forConst     bool
	undeclarated map[string]struct{}
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
				if !d.stk.mark(id.Name) {
					d.undeclarated[id.Name] = struct{}{}
				}
			}
		}
		for _, value := range n.Values {
			ast.Walk(&nodeVisitor{stk: d.stk, main: d.main, undeclarated: d.undeclarated}, value)
		}
		return nil
	case *ast.TypeSpec:
		ast.Walk(&typeVisitor{stk: d.stk, undeclarated: d.undeclarated}, node)
		return nil
	case *ast.StructType:
		for _, field := range n.Fields.List {
			if id, ok := field.Type.(*ast.Ident); ok {
				if !d.stk.mark(id.Name) {
					d.undeclarated[id.Name] = struct{}{}
				}
			}
		}
	case *ast.CallExpr:
		ast.Walk(&nodeVisitor{stk: d.stk, main: d.main, undeclarated: d.undeclarated}, node)
		return nil
	}
	return d
}

type typeVisitor struct {
	stk          stack
	undeclarated map[string]struct{}
}

func (t *typeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.TypeSpec:
		used := t.stk.top() && ast.IsExported(n.Name.Name)
		t.stk.add(n.Name.Name, node, used)
		if id, ok := n.Type.(*ast.Ident); ok {
			if !t.stk.mark(id.Name) {
				t.undeclarated[id.Name] = struct{}{}
			}
			return nil
		}
	case *ast.StructType:
		inspectFields(n.Fields, &t.stk, t.undeclarated)
		return nil
	case *ast.FuncType:
		inspectFields(n.Params, &t.stk, t.undeclarated)
		inspectFields(n.Results, &t.stk, t.undeclarated)
		return nil
	case *ast.ChanType:
		if id, ok := n.Value.(*ast.Ident); ok {
			if !t.stk.mark(id.Name) {
				t.undeclarated[id.Name] = struct{}{}
			}
			return nil
		}
	}
	return t
}

func inspectFields(fields *ast.FieldList, stk *stack, undeclarated map[string]struct{}) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		switch tField := field.Type.(type) {
		case *ast.Ident:
			if !stk.mark(tField.Name) {
				undeclarated[tField.Name] = struct{}{}
			}
		default:
			ast.Walk(&nodeVisitor{stk: *stk, main: false, undeclarated: undeclarated}, field.Type)
		}
	}
	return
}
