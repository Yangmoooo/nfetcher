package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"nfetcher/internal/metadata"
)

func RewriteCBZ(srcPath, dstPath, storyArc string, rank int) error {
	src, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	zipWriter := zip.NewWriter(dst)
	foundComicInfo := false

	for _, file := range src.File {
		if file.Name == "ComicInfo.xml" {
			foundComicInfo = true
			if err := copyPatchedComicInfo(zipWriter, file, storyArc, rank); err != nil {
				_ = zipWriter.Close()
				return err
			}
			continue
		}

		if err := copyEntry(zipWriter, file); err != nil {
			_ = zipWriter.Close()
			return err
		}
	}

	if !foundComicInfo {
		_ = zipWriter.Close()
		return fmt.Errorf("ComicInfo.xml not found in %s", srcPath)
	}

	return zipWriter.Close()
}

func copyPatchedComicInfo(zipWriter *zip.Writer, file *zip.File, storyArc string, rank int) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	patched, err := metadata.PatchComicInfo(data, storyArc, rank)
	if err != nil {
		return err
	}

	header := file.FileHeader
	header.CRC32 = 0
	header.CompressedSize = 0
	header.CompressedSize64 = 0
	header.UncompressedSize = 0
	header.UncompressedSize64 = 0

	writer, err := zipWriter.CreateHeader(&header)
	if err != nil {
		return err
	}
	_, err = writer.Write(patched)
	return err
}

func copyEntry(zipWriter *zip.Writer, file *zip.File) error {
	writer, err := zipWriter.CreateHeader(&file.FileHeader)
	if err != nil {
		return err
	}

	if file.FileInfo().IsDir() {
		return nil
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	return err
}
