// Copyright 2016 Aleksandr Demakin. All rights reserved.

package deadscanner

import (
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

type rec struct {
	name string
	line int
}

var (
	names = []rec{
		rec{"t", 4},
		rec{"main", 18},
		rec{"f2", 34},
		rec{"const1", 37},
		rec{"const2", 38},
		rec{"main", 39},
		rec{"init", 40},
		rec{"f3", 46},
		rec{"ttt", 48},
		rec{"const2", 49},
		rec{"const1", 52},
		rec{"f", 73},
	}
)

func TestDeadScanner(t *testing.T) {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, "./testpkg/", func(os.FileInfo) bool {
		return true
	}, parser.Mode(0))
	if err != nil {
		t.Error(err)
		return
	}
	s := New(pkgs["testpkg"])
	reports := s.Do()
	sort.Sort(reports)
	for i, name := range names {
		if i >= len(reports) {
			t.Errorf("expected %d records, got %d", len(names), len(reports))
			return
		}
		report := reports[i]
		if name.line != fs.Position(report.Pos).Line || name.name != report.Name {
			t.Errorf("expected {%s %d}, got {%s %d}", name.name, name.line,
				report.Name, fs.Position(report.Pos).Line)
		}
	}
	if len(reports) > len(names) {
		t.Errorf("expected %d records, got %d", len(names), len(reports))
		for i := len(names); i < len(reports); i++ {
			report := reports[i]
			t.Errorf("unexpected rec {%s %d}", report.Name, fs.Position(report.Pos).Line)
		}
	}
}
