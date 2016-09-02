package testpkg

import (
	"unsafe"
)

type used12 int

/// ---

type I1 interface {
	do(a used11)
}

type used11 int

/// ---

var (
	variable = unsafe.Sizeof(used13(0))
)

type used13 int

/// ---

func F4(param *used14) {

}

type used14 int

/// ---

func F5(param chan **used15) {

}

type used15 int

/// ---

var variable2 = used16{val: int(used17(0))}

type used16 struct {
	val int
}

type used17 int

/// ---

var _ = usedFunc()

func usedFunc() int {
	return 0
}
