package common

import (
	"runtime"
)

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

func GetOS() OS {
	switch runtime.GOOS {
	case "windows":
		{
			return OS_WINDOWS
		}
	case "darwin":
		{
			return OS_DARWIN
		}
	case "linux":
		{
			return OS_LINUX
		}
	default:
		{
			return OS_OTHER
		}
	}
}

func GetArch() ARCH {
	switch runtime.GOARCH {
	case "amd64":
		{
			return ARCH_AMD64
		}
	case "arm64":
		{
			return ARCH_ARM64
		}
	case "arm":
		{
			return ARCH_ARM
		}
	default:
		{
			return ARCH_OTHER
		}
	}
}
