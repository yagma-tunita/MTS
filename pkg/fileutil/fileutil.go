package fileutil

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// EnsureDir creates a directory if it does not exist.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Exists checks if a file or directory exists.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// WriteFile writes data to a file atomically (rename after write).
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir, 0755); err != nil {
		return err
	}
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}

// CopyDir recursively copies a directory.
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return CopyFile(path, dstPath)
	})
}

// FileLock is a simple exclusive file lock using flock.
type FileLock struct {
	path  string
	flock *flock.Flock
}

// NewFileLock creates a new file lock.
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path:  path,
		flock: flock.New(path),
	}
}

// Lock acquires the lock.
func (l *FileLock) Lock() error {
	return l.flock.Lock()
}

// Unlock releases the lock.
func (l *FileLock) Unlock() error {
	return l.flock.Unlock()
}

// TryLock attempts to acquire the lock without blocking.
func (l *FileLock) TryLock() (bool, error) {
	return l.flock.TryLock()
}
