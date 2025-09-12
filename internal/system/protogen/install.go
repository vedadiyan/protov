package protogen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vedadiyan/protov/internal/system/common"
)

func goPath() string {
	homeDir := os.Getenv("GOPATH")
	protocDir := filepath.Join(homeDir, "bin")
	return protocDir
}

func getReleaseFor(assets common.Assets, os common.OS, arch common.ARCH) (common.Release, error) {
	osAndArch, err := combineOSandArch(os, arch)
	if err != nil {
		return "", err
	}
	for _, i := range assets {
		if strings.Contains(string(i.BrowserDownloadURL), osAndArch) {
			return i.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("not found")
}

func combineOSandArch(os common.OS, arch common.ARCH) (string, error) {
	switch os {
	case common.OS_WINDOWS:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "windows.amd64", nil
				}
			}
		}
	case common.OS_DARWIN:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "darwin.amd64", nil
				}
			case common.ARCH_ARM64:
				{
					return "darwin.arm64", nil
				}
			}
		}
	case common.OS_LINUX:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "linux.amd64", nil
				}
			case common.ARCH_ARM64:
				{
					return "linux.arm64", nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s-%s is not supported", os, arch)
}

func Install(progressChannel chan<- int) error {
	repos, err := common.LatestTag("protocolbuffers", "protobuf-go")
	if err != nil {
		return err
	}
	release, err := getReleaseFor(repos.Assets, common.GetOS(), common.GetArch())
	if err != nil {
		return err
	}
	l, data, err := release.Download()
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	const bufferSize = 1024

	for {
		wl, err := io.CopyN(&buffer, data, bufferSize)
		if err != nil && err != io.EOF {
			return err
		}
		if progressChannel != nil {
			progressChannel <- int((float64(buffer.Len()) / float64(l) * 100))
		}
		if wl < bufferSize {
			break
		}
	}

	protoPath := goPath()

	if err := common.UnZipDump(protoPath, bytes.NewReader(buffer.Bytes()), l); err != nil {
		return err
	}
	return nil
}
