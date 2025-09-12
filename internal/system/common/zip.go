package common

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func UnZipDump(path string, r io.ReaderAt, l int64) error {
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
