package options

import (
	"fmt"

	"github.com/vedadiyan/protov/internal/system/install"
)

type Install struct {
}

func (i *Install) Run() error {
	return install.Install(func(p string) { fmt.Printf("%s", p) })
}
