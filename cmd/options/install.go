package options

import (
	"fmt"

	"github.com/vedadiyan/protov/internal/system/protoc"
)

type Install struct {
}

func (i *Install) Run() error {
	return protoc.Install(func(p string) { fmt.Printf("%s", p) })
}
