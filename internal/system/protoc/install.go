package protoc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vedadiyan/protov/internal/system/common"
	"golang.org/x/sys/windows/registry"
)

func ProtoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	protocDir := filepath.Join(homeDir, "protoc2")
	if err := os.MkdirAll(protocDir, 0755); err != nil {
		return "", err
	}

	return protocDir, nil
}

func exportEnv(protoPath string) error {
	if common.GetOS() == common.OS_WINDOWS {
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := os.Getenv("PATH")
	if strings.Contains(path, protoPath) {
		return nil
	}
	profile := filepath.Join(homeDir, ".bashrc")
	file, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "\nexport PATH=$PATH:%s\n", protoPath); err != nil {
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

func Install(feedback func(string)) error {
	repos, err := common.LatestTag("protocolbuffers", "protobuf")
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
		provideFeedback(feedback, fmt.Sprintf("\rDownloading Dependencies: %d%%", int((float64(buffer.Len())/float64(l)*100))))
		if wl < bufferSize {
			break
		}
	}
	provideFeedback(feedback, "\r\nSetting Environment Variables...")
	protoPath, err := ProtoPath()
	if err != nil {
		return err
	}

	protovFile, err := os.ReadFile(os.Args[0])
	if err != nil {
		return err
	}
	_, fileName := filepath.Split(os.Args[0])
	if err := os.WriteFile(filepath.Join(protoPath, fileName), protovFile, os.ModePerm); err != nil {
		return err
	}
	if err := exportEnv(protoPath); err != nil {
		return err
	}
	provideFeedback(feedback, "\r\nDecompressing Archives...")
	if err := common.UnZipDump(protoPath, bytes.NewReader(buffer.Bytes()), l); err != nil {
		return err
	}
	cmd := exec.Command("go", "install", "golang.org/x/tools/cmd/goimports@latest")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func provideFeedback(fn func(string), str string) {
	if fn != nil {
		fn(str)
	}
}
