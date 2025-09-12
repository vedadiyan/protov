package protoc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vedadiyan/protov/internal/system/common"
)

func protoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	protocDir := filepath.Join(homeDir, "protoc")
	if err := os.MkdirAll(protocDir, 0755); err != nil {
		return "", err
	}

	return protocDir, nil
}

func exportEnv(protoPath string) error {
	binPath := filepath.Join(protoPath, "bin")
	currentPath := os.Getenv("PATH")
	if common.GetOS() == common.OS_WINDOWS {
		cmd := exec.Command("setx", "PATH", fmt.Sprintf("%s;%s", currentPath, binPath)).Run()
		if cmd.Error() != "" {
			return errors.New(cmd.Error())
		}
		return nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	profile := filepath.Join(homeDir, ".bashrc")
	file, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "\nexport PATH=$PATH:%s\n", binPath); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
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
					return "win64", nil
				}
			}
		}
	case common.OS_DARWIN:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "osx-x86_64", nil
				}
			case common.ARCH_ARM64:
				{
					return "osx-aarch_64", nil
				}
			}
		}
	case common.OS_LINUX:
		{
			switch arch {
			case common.ARCH_AMD64:
				{
					return "linux-x86_32", nil
				}
			case common.ARCH_ARM64:
				{
					return "linux-aarch_64", nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s-%s is not supported", os, arch)
}

func Install(progressChannel chan<- int) error {
	repos, err := common.LatestTag("", "")
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

	protoPath, err := protoPath()
	if err != nil {
		return err
	}

	if err := exportEnv(protoPath); err != nil {
		return err
	}
	if err := common.UnZipDump(protoPath, bytes.NewReader(buffer.Bytes()), l); err != nil {
		return err
	}
	return nil
}
