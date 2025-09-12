package options

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/vedadiyan/protov/internal/system/common"
)

func protoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	protocDir := filepath.Join(homeDir, "protoc", "include", "options")
	if err := os.MkdirAll(protocDir, 0755); err != nil {
		return "", err
	}

	return protocDir, nil
}

func Install(progressChannel chan<- int) error {
	repos, err := common.LatestTag("vedadiyan", "options")
	if err != nil {
		return err
	}
	l, data, err := repos.Assets[0].BrowserDownloadURL.Download()
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

	if err := common.UnZipDump(protoPath, bytes.NewReader(buffer.Bytes()), int64(buffer.Len())); err != nil {
		return err
	}
	return nil
}
