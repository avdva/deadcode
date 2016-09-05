package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/tsenart/deadcode/deadscanner"
)

var exitCode int

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		doDir(".")
	} else {
		for _, name := range flag.Args() {
			// Is it a directory?
			if fi, err := os.Stat(name); err == nil && fi.IsDir() {
				doDir(name)
			} else {
				errorf("not a directory: %s", name)
			}
		}
	}
	os.Exit(exitCode)
}

// error formats the error to standard error, adding program
// identification and a newline
func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "deadcode: "+format+"\n", args...)
	exitCode = 2
}

func doDir(name string) {
	notests := func(info os.FileInfo) bool {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") &&
			!strings.HasSuffix(info.Name(), "_test.go") {
			return true
		}
		return false
	}
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, name, notests, parser.Mode(0))
	if err != nil {
		errorf("%s", err)
		return
	}
	for _, pkg := range pkgs {
		doPackage(fs, pkg)
	}
}

func doPackage(fs *token.FileSet, pkg *ast.Package) {
	s := deadscanner.New(pkg)
	reports := s.Do()
	sort.Sort(reports)
	for _, report := range reports {
		errorf("%s: %s is unused", fs.Position(report.Pos), report.Name)
	}
}
