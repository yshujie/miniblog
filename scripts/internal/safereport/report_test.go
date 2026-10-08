package safereport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicReportReplacesPublicInodeWithoutChangingHardlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	peer := filepath.Join(dir, "old-copy.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, peer); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("new private result")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("report is not private")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new private result" {
		t.Fatal("report replacement incomplete")
	}
	data, err = os.ReadFile(peer)
	if err != nil || string(data) != "old" {
		t.Fatal("existing hardlink was mutated")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatal("temporary report was left behind")
	}
}

func TestReportRejectsSymlinkAndDirectoryWithoutTouchingTarget(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.json")
	link := filepath.Join(dir, "report.json")
	if err := os.WriteFile(victim, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	if CheckDestination(link) == nil || Write(link, []byte("private")) == nil {
		t.Fatal("symlink report was accepted")
	}
	data, err := os.ReadFile(victim)
	if err != nil || string(data) != "keep" {
		t.Fatal("symlink target was changed")
	}
	if Write(dir, nil) == nil || Write(filepath.Join(dir, "missing", "report"), nil) == nil {
		t.Fatal("non-file destination was accepted")
	}
}

func TestFreshReportIsPrivateAndComplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := CheckDestination(path); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("new report is not private")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{}\n" {
		t.Fatal("new report incomplete")
	}
}
