package compiler

import "testing"

func TestCompile(t *testing.T) {
	res, err := Compile("C:\\Users\\Pouya\\Desktop\\lab\\users\\model.proto")
	if err != nil {
		t.FailNow()
	}
	_ = res
}
