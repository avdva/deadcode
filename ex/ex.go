package ex

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
