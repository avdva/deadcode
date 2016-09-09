// Copyright 2016 Aleksandr Demakin. All rights reserved.

package deadscanner

import (
	"go/parser"
	"go/token"
	"os"
	"testing"
)

type rec struct {
	name string
	line int
}

func check(t *testing.T, fs *token.FileSet, records []rec, reports []Report) {
	var i int
outer:
	for i < len(records) {
		name := records[i]
		for j := 0; j < len(reports); j++ {
			report := reports[j]
			if name.line == fs.Position(report.Pos).Line && name.name == report.Name {
				newRecords := records[:i]
				if i < len(records)-1 {
					newRecords = append(newRecords, records[i+1:]...)
				}
				newReports := reports[:j]
				if j < len(reports)-1 {
					newReports = append(newReports, reports[j+1:]...)
				}
				records, reports = newRecords, newReports
				continue outer
			}
		}
		i++
	}
	for _, rec := range records {
		t.Errorf("not marked as unused: %v", rec)
	}
	for _, rep := range reports {
		t.Errorf("must not marked be as unused: %s at %s", rep.Name, fs.Position(rep.Pos))
	}
}

func checkDir(t *testing.T, records []rec, dir, pkg string) {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, dir, func(os.FileInfo) bool {
		return true
	}, parser.Mode(0))
	if err != nil {
		t.Error(err)
		return
	}
	s := New(pkgs[pkg])
	reports := s.Do()

	check(t, fs, records, reports)
}

func TestDeadScannerNonMain(t *testing.T) {
	var (
		records = []rec{
			rec{"t", 4},
			rec{"main", 18},
			rec{"variable", 20},
			rec{"f2", 34},
			rec{"const1", 37},
			rec{"const2", 38},
			rec{"main", 39},
			rec{"init", 40},
			rec{"variable2", 43},
			rec{"f3", 46},
			rec{"ttt", 48},
			rec{"const2", 49},
			rec{"const1", 52},
			rec{"f", 73},
		}
	)

	checkDir(t, records, "./testdata/pkg/", "testpkg")
}

func TestDeadScannerMain(t *testing.T) {
	var (
		records = []rec{
			rec{"Unused1", 3},
			rec{"UnusedConst1", 5},
			rec{"unusedConst2", 6},
			rec{"unusedfunc1", 8},
			rec{"UnusedFunc2", 15},
			rec{"UnusedType", 20},
		}
	)
	checkDir(t, records, "./testdata/main/", "main")
}

func TestDeadScannerConst(t *testing.T) {
	var (
		records = []rec{}
	)
	checkDir(t, records, "./testdata/cnst/", "cnst")
}

func TestDeadScannerArray(t *testing.T) {
	var (
		records = []rec{}
	)
	checkDir(t, records, "./testdata/array/", "array")
}
