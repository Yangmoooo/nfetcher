package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type ExtraFile struct {
	Name string
	Data []byte
}

func WriteCBZ(srcDir, dstPath string, extraFiles []ExtraFile) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	zipWriter := zip.NewWriter(dst)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		src, err := os.Open(srcPath)
		if err != nil {
			_ = zipWriter.Close()
			return err
		}

		writer, err := zipWriter.Create(entry.Name())
		if err != nil {
			src.Close()
			_ = zipWriter.Close()
			return err
		}

		if _, err := io.Copy(writer, src); err != nil {
			src.Close()
			_ = zipWriter.Close()
			return err
		}

		if err := src.Close(); err != nil {
			_ = zipWriter.Close()
			return err
		}
	}

	sort.Slice(extraFiles, func(left, right int) bool {
		return extraFiles[left].Name < extraFiles[right].Name
	})

	for _, extraFile := range extraFiles {
		writer, err := zipWriter.Create(extraFile.Name)
		if err != nil {
			_ = zipWriter.Close()
			return err
		}

		if _, err := io.Copy(writer, bytes.NewReader(extraFile.Data)); err != nil {
			_ = zipWriter.Close()
			return err
		}
	}

	return zipWriter.Close()
}
