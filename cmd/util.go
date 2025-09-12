package main

import (
	"runtime"

	"github.com/vedadiyan/protov/internal/common"
)

func GetOS() common.OS {
	switch runtime.GOOS {
	case "windows":
		{
			return common.OS_WINDOWS
		}
	case "darwin":
		{
			return common.OS_DARWIN
		}
	case "linux":
		{
			return common.OS_LINUX
		}
	default:
		{
			return common.OS_OTHER
		}
	}
}

func GetArch() common.ARCH {
	switch runtime.GOARCH {
	case "amd64":
		{
			return common.ARCH_AMD64
		}
	case "arm64":
		{
			return common.ARCH_ARM64
		}
	case "arm":
		{
			return common.ARCH_ARM
		}
	default:
		{
			return common.ARCH_OTHER
		}
	}
}
