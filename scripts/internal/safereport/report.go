// Package safereport writes private operator reports without following a target
// symlink or exposing new contents through an existing world-readable inode.
package safereport

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var ErrUnsafeDestination = errors.New("report destination must be a regular file in an existing directory")

// CheckDestination validates a file report before an operation starts. Write
// repeats this check using its pinned directory descriptor before replacement.
func CheckDestination(path string) error {
	directory, name, err := openDirectory(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return checkFile(int(directory.Fd()), name)
}

func openDirectory(path string) (*os.File, string, error) {
	if path == "" || path == "-" {
		return nil, "", ErrUnsafeDestination
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, "", ErrUnsafeDestination
	}
	name := filepath.Base(absolute)
	if name == "." || name == ".." || name == string(filepath.Separator) {
		return nil, "", ErrUnsafeDestination
	}
	directory, err := os.Open(filepath.Dir(absolute))
	if err != nil {
		return nil, "", ErrUnsafeDestination
	}
	info, err := directory.Stat()
	if err != nil || !info.IsDir() {
		directory.Close()
		return nil, "", ErrUnsafeDestination
	}
	return directory, name, nil
}

func checkFile(dirfd int, name string) error {
	var stat unix.Stat_t
	err := unix.Fstatat(dirfd, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return ErrUnsafeDestination
	}
	return nil
}

// Write pins the directory, creates an exclusive 0600 temporary inode, syncs
// it, then atomically replaces the target. It never opens the old target for
// writing: symlinks are rejected, hardlink peers keep their old contents, and
// replacing a 0644 report yields a new 0600 inode. Parent directory resolution
// happens once; later path changes cannot redirect the write to another parent.
func Write(path string, data []byte) error {
	directory, name, err := openDirectory(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	dirfd := int(directory.Fd())
	if err = checkFile(dirfd, name); err != nil {
		return err
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return errors.New("report temporary file unavailable")
	}
	temporary := ".miniblog-report-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(dirfd, temporary, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return errors.New("report temporary file unavailable")
	}
	defer unix.Unlinkat(dirfd, temporary, 0)
	file := os.NewFile(uintptr(fd), temporary)
	defer file.Close()
	if err = file.Chmod(0600); err != nil {
		return errors.New("report permissions unavailable")
	}
	if _, err = file.Write(data); err != nil {
		return errors.New("report file write failed")
	}
	if err = file.Sync(); err != nil {
		return errors.New("report file sync failed")
	}
	if err = file.Close(); err != nil {
		return errors.New("report file close failed")
	}
	if err = checkFile(dirfd, name); err != nil {
		return err
	}
	if err = unix.Renameat(dirfd, temporary, dirfd, name); err != nil {
		return errors.New("report atomic replacement failed")
	}
	if err = directory.Sync(); err != nil {
		return errors.New("report directory sync failed")
	}
	return nil
}
