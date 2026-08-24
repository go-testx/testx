package testx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const updateGoldenEnv = "TESTX_UPDATE_GOLDEN"

// Golden compares actual bytes with a golden file. Set TESTX_UPDATE_GOLDEN=1
// to create or update the file.
func Golden(t testing.TB, path string, actual []byte) bool {
	t.Helper()
	if path == "" {
		t.Errorf("testx.Golden: path is empty")
		return false
	}
	if os.Getenv(updateGoldenEnv) == "1" {
		if err := writeFileAtomic(path, actual, 0o644); err != nil {
			t.Errorf("testx.Golden: update %q: %v", path, err)
			return false
		}
		return true
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("testx.Golden: read %q: %v (set %s=1 to create it)", path, err, updateGoldenEnv)
		return false
	}
	return Assert(t, actual).Equal(want)
}

// GoldenString compares actual text with a golden file.
func GoldenString(t testing.TB, path, actual string) bool {
	t.Helper()
	return Golden(t, path, []byte(actual))
}

// Snapshot stores a named golden file under testdata/snapshots.
func Snapshot(t testing.TB, name string, actual []byte) bool {
	t.Helper()
	return Golden(t, snapshotPath(t.Name(), name), actual)
}

// SnapshotString stores a named text snapshot under testdata/snapshots.
func SnapshotString(t testing.TB, name, actual string) bool {
	t.Helper()
	return Snapshot(t, name, []byte(actual))
}

var unsafeSnapshotChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func snapshotPath(testName, name string) string {
	base := unsafeSnapshotChars.ReplaceAllString(testName, "_")
	label := unsafeSnapshotChars.ReplaceAllString(name, "_")
	if label == "" {
		label = "snapshot"
	}
	return filepath.Join("testdata", "snapshots", fmt.Sprintf("%s_%s.golden", base, label))
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".testx-golden-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(perm); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err == nil {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	backup, err := os.CreateTemp(dir, ".testx-golden-backup-*")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
			return fmt.Errorf("replace failed: %v; restore failed: %v", err, restoreErr)
		}
		return err
	}
	return os.Remove(backupPath)
}
