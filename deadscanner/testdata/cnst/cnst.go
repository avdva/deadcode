package cnst

///

const c1 = 1
const c2 = 1
const c3 = 1
const c4 = 1
const c5 = 1

var _ = []int{c3: 1}

type Arr [c5]int

type T1 struct {
	F1 [c1]int
}

func init() {
	_ = []int{c2: 1}
	var _ [c4]int
}
