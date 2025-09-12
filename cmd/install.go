package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/klauspost/compress/zip"
	githubrepo "github.com/vedadiyan/protov/internal/clients/github/repos"
	"github.com/vedadiyan/protov/internal/common"
)

func PreparePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	protocDir := filepath.Join(homeDir, "protoc")
	if err := os.MkdirAll(protocDir, 0755); err != nil {
		return "", err
	}

	binPath := filepath.Join(protocDir, "bin")
	currentPath := os.Getenv("PATH")
	if GetOS() == common.OS_WINDOWS {
		exec.Command("setx", "PATH", fmt.Sprintf("%s;%s", currentPath, binPath)).Run()
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		profile := filepath.Join(homeDir, ".bashrc")
		file, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(file, "\nexport PATH=$PATH:%s\n", binPath); err != nil {
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return protocDir, nil
}

func InstallProtoc(progressChannel chan<- int) error {

	repos, err := githubrepo.LatestTag()
	if err != nil {
		return err
	}
	release, err := repos.Assets.GetReleaseFor(GetOS(), GetArch())
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

	protocDir, err := PreparePath()
	if err != nil {
		return err
	}
	if err := UnZip(protocDir, bytes.NewReader(buffer.Bytes()), l); err != nil {
		return err
	}
	return nil
}

func UnZip(path string, r io.ReaderAt, l int64) error {
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
