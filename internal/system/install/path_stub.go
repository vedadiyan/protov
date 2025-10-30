//go:build !windows

package install

func EnsureInUserPath(protoPath string) error {
	return nil
}
