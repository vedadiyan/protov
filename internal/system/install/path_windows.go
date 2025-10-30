//go:build windows

package install

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func EnsureInUserPath(protoPath string) error {
	val, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QWORD)
	if err != nil {
		return err
	}
	v, _, err := val.GetStringValue("Path")
	if err != nil {
		return err
	}
	if strings.Contains(v, protoPath) {
		return nil
	}
	if err := val.SetStringValue("Path", fmt.Sprintf("%s;%s", v, protoPath)); err != nil {
		return err
	}
	return nil
}
