package common

type OS string
type ARCH string

const (
	OS_WINDOWS OS = "windows"
	OS_DARWIN  OS = "darwin"
	OS_LINUX   OS = "linux"
	OS_OTHER   OS = "other"
)

const (
	ARCH_AMD64 ARCH = "amd64"
	ARCH_ARM   ARCH = "arm"
	ARCH_ARM64 ARCH = "arm64"
	ARCH_OTHER ARCH = "other"
)
