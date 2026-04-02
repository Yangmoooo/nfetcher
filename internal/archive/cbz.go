package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func WriteCBZ(srcDir, dstPath string) error {
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

	return zipWriter.Close()
}
