package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionFileRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := saveSessionFile("access-1", "refresh-1"); err != nil {
		t.Fatal(err)
	}
	a, r, err := loadSessionFile()
	if err != nil || a != "access-1" || r != "refresh-1" {
		t.Fatalf("loadSessionFile = %q, %q, %v", a, r, err)
	}

	// File and directory must be owner-only.
	info, err := os.Stat(filepath.Join(tmp, ".trufo", "session"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("session file perm = %o, want 600", perm)
	}
	dinfo, err := os.Stat(filepath.Join(tmp, ".trufo"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dinfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("session dir perm = %o, want 700", perm)
	}

	// Overwrite (rotation) works and stays atomic-readable.
	if err := saveSessionFile("access-2", "refresh-2"); err != nil {
		t.Fatal(err)
	}
	a, r, err = loadSessionFile()
	if err != nil || a != "access-2" || r != "refresh-2" {
		t.Fatalf("after rotation: %q, %q, %v", a, r, err)
	}

	// Corrupt file surfaces an error (Configure turns it into a warning).
	if err := os.WriteFile(filepath.Join(tmp, ".trufo", "session"), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadSessionFile(); err == nil {
		t.Error("expected error for corrupt session file")
	}

	// Missing file is ErrNotExist (silent skip path).
	if err := os.Remove(filepath.Join(tmp, ".trufo", "session")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadSessionFile(); !os.IsNotExist(err) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}
