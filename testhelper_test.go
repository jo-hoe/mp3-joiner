package mp3joiner

import (
	"os"
	"path/filepath"
	"testing"
)

const testFilename = "edgar-allen-poe-the-telltale-heart-original.mp3"

func getMP3TestFolder(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "test", "mp3")
}

func generateMP3FileName(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "*.mp3")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	storeFileForCleanUp(t, name)
	return name
}

func storeFileForCleanUp(t *testing.T, filename string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := os.Stat(filename); err == nil {
			if err := os.Remove(filename); err != nil {
				t.Errorf("could not delete file %s: %v", filename, err)
			}
		}
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func checkErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func getFileSizeInBytes(t *testing.T, filePath string) int64 {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("could not stat %s: %v", filePath, err)
	}
	return info.Size()
}

func setupTestFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "testFile")
	if err != nil {
		t.Fatal("could not create test file:", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal("could not close test file:", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("could not remove test file %s: %v", f.Name(), err)
		}
	})
	return f.Name()
}
