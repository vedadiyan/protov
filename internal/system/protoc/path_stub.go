//go:build !windows

package protoc

func EnsureInUserPath(protoPath string) error {
	return nil
}
