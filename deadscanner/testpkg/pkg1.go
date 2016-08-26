package testpkg

// unused unexported const
const t = 10

// exported consts are always used
const ExportedConst = 0

// unexported used type
type ttt string

// init is always used
func init() {

}

// main is not used in non-main packages
func main() {

}

// used func
func f1() {
	const used = 234
	var a int = used
	var c ttt
	b := used
	_ = a
	_ = b
	_ = c
}

// unused func
func f2() {
	// unused consts
	const (
		const1 = 20
		const2 = 30
		main   = 0
		init   = 0
	)
	f1()
}

// unused func
func f3(a, b int) float32 {
	// unused local type
	type ttt string
	const const1, const2 = 2, 3
	{
		// unused const in a separate scope
		const const1 = 2
	}
	_ = const1
	return 0
}

// exported funcs are always used.
func Used() {
	v := func(a, b int) {

	}
	_ = v
}

type innertype struct {
}

type outertype struct {
	i innertype
}

func f() {
	c := outertype{}
	_ = c
}

type outer2 struct {
	f  interface{}
	f2 interface{}
	a  int
}

type used2 struct {
}

func V() {
	v := outer2{f: used2{}}
	u := outer2{used3{}, used3{}, 0}
	_ = v
	_ = u
}

type used3 struct {
}

func U() {
	type usedType int
	{
		var c usedType
		_ = c
	}
	_ = usedConst1
}

type Outer3 struct {
	u used4
}

type used4 int

type Outer5 used5

type used5 int

const usedConst1 = 0

func F1(a, b used6) {

}

type used6 int

type FT func(a, b used7)

type used7 int

type used8 int

func F2(a used8) {

}

func F3() {
	ch := make(chan used9)
	_ = ch
}

type used9 int

type Ex chan used10

type used10 int

type Ex2 used12
