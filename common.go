package mp3joiner

import (
	"bytes"
	"log"
	"os"
	"os/exec"
)

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func deleteFile(filePath string) {
	if err := os.Remove(filePath); err != nil {
		log.Printf("could not delete file %s: %v", filePath, err)
	}
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("could not close file %s: %v", file.Name(), err)
	}
}
