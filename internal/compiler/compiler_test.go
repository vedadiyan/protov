package compiler

import (
	"os"
	"os/exec"
	"testing"
)

func TestCompile(t *testing.T) {
	res, err := Parse("C:\\Users\\Pouya\\Desktop\\New folder\\users\\service.proto")
	if err != nil {
		t.Fatal(err)
	}
	_r, err := Compile(res.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile("test.go", []byte(_r), os.ModePerm)
	cmd := exec.Command("gofmt", "-w", "test.go")
	cmd.Run()
}

// func TestT(t *testing.T) {
// 	id := int64(1)
// 	x := User{
// 		id:         &id,
// 		first_name: "test",
// 	}

// 	z, err := proto.Marshal(&x)
// 	if err != nil {
// 		t.FailNow()
// 	}

// 	zz := User{}
// 	err = proto.Unmarshal(z, &zz)
// 	if err != nil {
// 		t.FailNow()
// 	}

// }
