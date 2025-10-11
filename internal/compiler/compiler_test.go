package compiler

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
	"text/template"
)

func TestCompile(t *testing.T) {
	res, err := Compile(`C:\Users\Pouya\Desktop\lab\users\service.proto`)
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.ParseFiles(
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\service.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\enum.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\message.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\iszero.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\encoderepeated.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\encodemap.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\encode.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\decoderepeated.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\decodemap.go.tmpl`,
		`C:\Users\Pouya\Desktop\lab\protov\internal\compiler\microservice\templates\decode.go.tmpl`,
	)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tmpl.ExecuteTemplate(&out, "Main", res.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	_r := out.String()
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
