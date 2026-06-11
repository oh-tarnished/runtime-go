// package config provides a thread-safe wrapper for performing common file system
// operations. It uses a mutex to serialize I/O operations, ensuring safe
// concurrent access to the local file system. All file operations are resolved
// relative to a configured base path, but absolute paths are also supported.
package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oh-tarnished/runtime-go/config/shared"
)

// IOInterface is the interface that wraps basic file system operations.
// It provides a contract for reading, writing, and managing files and directories.
type IOInterface interface {
	// ReadFile reads the file from the local file system.
	ReadFile(string) ([]byte, error)

	// WriteFile writes the file to the local file system.
	WriteFile(string, []byte) error

	// DeleteFile deletes the file from the local file system.
	DeleteFile(string) error

	// FileExists checks if the file exists in the local file system.
	FileExists(string) bool

	// GetFileList returns a list of files in the directory.
	GetFileList(string) ([]string, error)

	// CreateDirectory creates a directory in the local file system.
	CreateDirectory(string) error

	// DeleteDirectory deletes a directory from the local file system,
	// including all nested files and subdirectories.
	DeleteDirectory(string) error

	// DirectoryExists checks if the directory exists in the local file system.
	DirectoryExists(string) bool

	// CheckPathPermission checks if the path is accessible.
	// Note: This currently only verifies path existence, not specific read/write permissions.
	CheckPathPermission(string) bool
}

// IO provides a thread-safe implementation of the IOInterface.
// It manages file operations relative to a specified BasePath.
type IO struct {
	ioMutex  sync.RWMutex
	BasePath string
}

// expandPath expands a path beginning with a tilde (~) to the current
// user's home directory. If the path does not start with a tilde, it is
// returned unchanged.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		shared.Pulse.Logger.Errorf("expandPath failed to get current user: %v", err)
		return "", err
	}

	if path == "~" {
		return usr.HomeDir, nil
	}

	return filepath.Join(usr.HomeDir, path[2:]), nil
}

// NewIO creates and initializes a new IO instance for the given base path.
// It expands tilde (~) in the path, resolves it to an absolute path, and
// creates the directory if it does not already exist. It returns the new IO
// instance or an error if the path cannot be processed or created.
func newIO(path string) (*IO, error) {
	shared.Pulse.Logger.Debugf("NewIO called with base path: %s", path)

	expandedPath, err := expandPath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("NewIO expandPath error: %v", err)
		return nil, err
	}
	shared.Pulse.Logger.Debugf("Expanded base path: %s", expandedPath)

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		shared.Pulse.Logger.Errorf("NewIO Abs path error: %v", err)
		return nil, err
	}

	if err := os.MkdirAll(absPath, 0750); err != nil {
		shared.Pulse.Logger.Errorf("NewIO mkdir error: %v", err)
		return nil, err
	}

	shared.Pulse.Logger.Debugf("NewIO initialized with path: %s", absPath)
	return &IO{BasePath: absPath}, nil
}

// resolvePath converts a given path to an absolute path. If the path is
// already absolute, it is returned as is. If it is relative, it is joined
// with the IO's BasePath. It also handles tilde (~) expansion.
func (l *IO) resolvePath(path string) (string, error) {
	shared.Pulse.Logger.Debugf("resolvePath called with: %s", path)
	expanded, err := expandPath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("resolvePath expandPath error: %v", err)
		return "", err
	}

	if filepath.IsAbs(expanded) {
		shared.Pulse.Logger.Debugf("resolvePath returning absolute path: %s", expanded)
		return expanded, nil
	}

	resolved := filepath.Join(l.BasePath, expanded)
	shared.Pulse.Logger.Debugf("resolvePath joined with BasePath: %s", resolved)
	return resolved, nil
}

// ReadFile reads the entire file named by path and returns the contents.
// The operation is thread-safe. Paths are resolved relative to l.BasePath.
func (l *IO) ReadFile(path string) ([]byte, error) {
	shared.Pulse.Logger.Debugf("ReadFile called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("ReadFile resolvePath error: %v", err)
		return nil, err
	}

	resultChan := make(chan []byte)
	errChan := make(chan error)

	go func() {
		l.ioMutex.RLock()
		defer l.ioMutex.RUnlock()

		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			shared.Pulse.Logger.Errorf("ReadFile error reading file: %v", err)
			errChan <- err
			return
		}
		shared.Pulse.Logger.Debugf("ReadFile success: %s", resolvedPath)
		resultChan <- data
	}()

	select {
	case data := <-resultChan:
		return data, nil
	case err := <-errChan:
		return nil, err
	}
}

// WriteFile writes data to a file named by path. If the file does not exist,
// it is created with permissions 0644. Any necessary parent directories are
// also created. The operation is thread-safe.
func (l *IO) WriteFile(path string, data []byte) error {
	shared.Pulse.Logger.Debugf("WriteFile called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("WriteFile resolvePath error: %v", err)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0750); err != nil {
		shared.Pulse.Logger.Errorf("WriteFile mkdir error: %v", err)
		return err
	}

	errChan := make(chan error)

	go func() {
		l.ioMutex.Lock()
		defer l.ioMutex.Unlock()

		err := os.WriteFile(resolvedPath, data, 0600)
		if err != nil {
			shared.Pulse.Logger.Errorf("WriteFile error writing file: %v", err)
		} else {
			shared.Pulse.Logger.Debugf("WriteFile success: %s", resolvedPath)
		}
		errChan <- err
	}()

	return <-errChan
}

// DeleteFile deletes the file named by path. The operation is thread-safe.
func (l *IO) DeleteFile(path string) error {
	shared.Pulse.Logger.Debugf("DeleteFile called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("DeleteFile resolvePath error: %v", err)
		return err
	}

	errChan := make(chan error)

	go func() {
		l.ioMutex.Lock()
		defer l.ioMutex.Unlock()

		err := os.Remove(resolvedPath)
		if err != nil {
			shared.Pulse.Logger.Errorf("DeleteFile error: %v", err)
		} else {
			shared.Pulse.Logger.Debugf("DeleteFile success: %s", resolvedPath)
		}
		errChan <- err
	}()

	return <-errChan
}

// FileExists checks whether a file exists at the given path.
// The operation is thread-safe.
func (l *IO) FileExists(path string) bool {
	shared.Pulse.Logger.Debugf("FileExists called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("FileExists resolvePath error: %v", err)
		return false
	}

	resultChan := make(chan bool)

	go func() {
		l.ioMutex.RLock()
		defer l.ioMutex.RUnlock()

		_, err := os.Stat(resolvedPath)
		exists := !os.IsNotExist(err)
		shared.Pulse.Logger.Debugf("FileExists %s: %t", resolvedPath, exists)
		resultChan <- exists
	}()

	return <-resultChan
}

// GetFileList walks the directory tree rooted at dir and returns a slice of all
// file paths found. The returned paths are relative to the IO's BasePath.
// The operation is thread-safe.
func (l *IO) GetFileList(dir string) ([]string, error) {
	shared.Pulse.Logger.Debugf("GetFileList called with: %s", dir)
	resolvedPath, err := l.resolvePath(dir)
	if err != nil {
		shared.Pulse.Logger.Errorf("GetFileList resolvePath error: %v", err)
		return nil, err
	}

	resultChan := make(chan []string)
	errChan := make(chan error)

	go func() {
		l.ioMutex.RLock()
		defer l.ioMutex.RUnlock()

		var files []string
		err := filepath.Walk(resolvedPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(l.BasePath, path)
				if err != nil {
					return err
				}
				files = append(files, relPath)
			}
			return nil
		})

		if err != nil {
			shared.Pulse.Logger.Errorf("GetFileList error: %v", err)
			errChan <- err
			return
		}
		shared.Pulse.Logger.Debugf("GetFileList found %d files in %s", len(files), resolvedPath)
		resultChan <- files
	}()

	select {
	case files := <-resultChan:
		return files, nil
	case err := <-errChan:
		return nil, err
	}
}

// CreateDirectory creates a directory at the given path, along with any
// necessary parents. If the path already exists and is a directory, this
// function does nothing. The operation is thread-safe.
func (l *IO) CreateDirectory(path string) error {
	shared.Pulse.Logger.Debugf("CreateDirectory called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("CreateDirectory resolvePath error: %v", err)
		return err
	}

	errChan := make(chan error)

	go func() {
		l.ioMutex.Lock()
		defer l.ioMutex.Unlock()

		err := os.MkdirAll(resolvedPath, 0750)
		if err != nil {
			shared.Pulse.Logger.Errorf("CreateDirectory error: %v", err)
		} else {
			shared.Pulse.Logger.Debugf("CreateDirectory success: %s", resolvedPath)
		}
		errChan <- err
	}()

	return <-errChan
}

// DeleteDirectory removes the directory at path and any children it contains.
// It removes everything it can but returns the first error it encounters.
// The operation is thread-safe.
func (l *IO) DeleteDirectory(path string) error {
	shared.Pulse.Logger.Debugf("DeleteDirectory called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("DeleteDirectory resolvePath error: %v", err)
		return err
	}

	errChan := make(chan error)

	go func() {
		l.ioMutex.Lock()
		defer l.ioMutex.Unlock()

		err := os.RemoveAll(resolvedPath)
		if err != nil {
			shared.Pulse.Logger.Errorf("DeleteDirectory error: %v", err)
		} else {
			shared.Pulse.Logger.Debugf("DeleteDirectory success: %s", resolvedPath)
		}
		errChan <- err
	}()

	return <-errChan
}

// DirectoryExists checks whether a path exists and is a directory.
// The operation is thread-safe.
func (l *IO) DirectoryExists(path string) bool {
	shared.Pulse.Logger.Debugf("DirectoryExists called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("DirectoryExists resolvePath error: %v", err)
		return false
	}

	resultChan := make(chan bool)

	go func() {
		l.ioMutex.RLock()
		defer l.ioMutex.RUnlock()

		info, err := os.Stat(resolvedPath)
		if os.IsNotExist(err) {
			shared.Pulse.Logger.Debugf("DirectoryExists not found: %s", resolvedPath)
			resultChan <- false
			return
		}
		exists := info.IsDir()
		shared.Pulse.Logger.Debugf("DirectoryExists %s: %t", resolvedPath, exists)
		resultChan <- exists
	}()

	return <-resultChan
}

// CheckPathPermission reports whether a path is accessible by checking for
// its existence. It does not check for specific read or write permissions.
// Returns true if `os.Stat` on the path succeeds, false otherwise.
func (l *IO) CheckPathPermission(path string) bool {
	shared.Pulse.Logger.Debugf("CheckPathPermission called with: %s", path)
	resolvedPath, err := l.resolvePath(path)
	if err != nil {
		shared.Pulse.Logger.Errorf("CheckPathPermission resolvePath error: %v", err)
		return false
	}

	_, err = os.Stat(resolvedPath)
	if err != nil {
		shared.Pulse.Logger.Errorf("CheckPathPermission stat error: %v", err)
		return false
	}

	shared.Pulse.Logger.Debugf("CheckPathPermission success for: %s", resolvedPath)
	return true
}
