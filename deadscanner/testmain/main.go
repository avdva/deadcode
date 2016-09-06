package main

var Unused1 = "asd"

const UnusedConst1 = 0
const unusedConst2 = 0

func unusedfunc1() {

}

const UsedConst1 = 0
const usedConst2 = 0

func UnusedFunc2() {
	a, b := UsedConst1, usedConst2
	_, _ = a, b
}

func main() {

}
