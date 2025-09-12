package protoc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/klauspost/compress/zip"
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
			return fmt.Errorf(cmd.Error())
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

func unzip(path string, r io.ReaderAt, l int64) error {
	reader, err := zip.NewReader(r, l)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		destPath := filepath.Join(path, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(destPath, file.FileInfo().Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(destPath), 0755)
		rc, err := file.Open()
		if err != nil {
			return err
		}
		outFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(outFile, rc); err != nil {
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
		if err := outFile.Close(); err != nil {
			return err
		}
	}
	return nil
}

func Install(progressChannel chan<- int) error {
	repos, err := common.LatestTag("", "")
	if err != nil {
		return err
	}
	release, err := repos.Assets.GetReleaseFor(common.GetOS(), common.GetArch())
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
	if err := unzip(protoPath, bytes.NewReader(buffer.Bytes()), l); err != nil {
		return err
	}
	return nil
}
