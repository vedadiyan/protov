package compiler

import (
	"bytes"
	"os"
	"testing"
	"text/template"

	"google.golang.org/protobuf/proto"
)

func TestCompile(t *testing.T) {
	res, err := Compile("C:\\Users\\Pouya\\Desktop\\lab\\users\\model.proto")
	if err != nil {
		t.FailNow()
	}
	tmpl, err := template.ParseFiles(`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\message.go.tmpl`, `C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\test.go.tmpl`)
	if err != nil {
		t.FailNow()
	}
	var out bytes.Buffer
	err = tmpl.ExecuteTemplate(&out, "Main", res.Files[0])
	if err != nil {
		t.FailNow()
	}
	_r := out.String()
	os.WriteFile("test.go", []byte(_r), os.ModePerm)
}

func TestT(t *testing.T) {
	id := int64(1)
	x := User{
		id:         &id,
		first_name: "test",
	}

	z, err := proto.Marshal(&x)
	if err != nil {
		t.FailNow()
	}

	zz := User{}
	err = proto.Unmarshal(z, &zz)
	if err != nil {
		t.FailNow()
	}

}
