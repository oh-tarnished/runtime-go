package config

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/oh-tarnished/runtime-go/config/shared"
)

// Compression provides methods for creating and extracting archives.
type Compression struct {
	io *IO
}

// CompressionType defines supported archive formats.
type CompressionType string

const (
	TarGz CompressionType = "tar.gz"
)

// newCompression creates a new Compression instance.
func newCompression(io *IO) *Compression {
	shared.Pulse.Logger.Debugf("Initializing new Compression handler")
	return &Compression{io: io}
}

// CreateTarGz creates a gzipped tar archive from a list of source files.
// The archive is saved to the destinationPath.
func (c *Compression) CreateTarGz(destinationPath string, sourceFiles []string) error {
	shared.Pulse.Logger.Infof("Creating archive: %s", destinationPath)

	// Use a buffer to create the archive in memory before writing to disk.
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// Add each source file to the archive.
	for _, path := range sourceFiles {
		fileBytes, err := c.io.ReadFile(path)
		if err != nil {
			shared.Pulse.Logger.Errorf("Failed to read source file %s for archiving: %v", path, err)
			return err
		}

		header := &tar.Header{
			Name:    path,
			Size:    int64(len(fileBytes)),
			Mode:    int64(0644),
			ModTime: time.Now(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write(fileBytes); err != nil {
			return err
		}
		shared.Pulse.Logger.Debugf("Added file to archive: %s", path)
	}

	// It's crucial to close the writers in order to flush all data.
	tarWriter.Close()
	gzipWriter.Close()

	// Write the final archive from the buffer to the destination file.
	return c.io.WriteFile(destinationPath, buf.Bytes())
}

// ExtractTarGz extracts a gzipped tar archive to the specified destination directory.
func (c *Compression) ExtractTarGz(archivePath string, dest string, strip int) error {
	shared.Pulse.Logger.Infof("Extracting archive '%s' to '%s'", archivePath, dest)
	if err := c.io.CreateDirectory(dest); err != nil {
		return err
	}

	// Read the entire archive into memory using the IO module.
	archiveBytes, err := c.io.ReadFile(archivePath)
	if err != nil {
		shared.Pulse.Logger.Errorf("Failed to read archive file %s: %v", archivePath, err)
		return err
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(archiveBytes))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Strip leading path components if requested.
		pathParts := strings.Split(header.Name, string(filepath.Separator))
		if len(pathParts) <= strip {
			continue
		}
		strippedPath := filepath.Join(pathParts[strip:]...)
		target := filepath.Join(dest, strippedPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := c.io.CreateDirectory(target); err != nil {
				return err
			}
		case tar.TypeReg:
			// Read the content of the file within the archive.
			fileContent, err := io.ReadAll(tarReader)
			if err != nil {
				return err
			}
			// Write the extracted file to disk.
			if err := c.io.WriteFile(target, fileContent); err != nil {
				return err
			}
			shared.Pulse.Logger.Debugf("Extracted file: %s", target)
		default:
			shared.Pulse.Logger.Warnf("Unsupported file type in archive: %s (type %d)", header.Name, header.Typeflag)
		}
	}
	shared.Pulse.Logger.Infof("Successfully extracted archive: %s", archivePath)
	return nil
}
