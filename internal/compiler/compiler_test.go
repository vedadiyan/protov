package compiler

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"
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
	fmt.Println(_r)
}
