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
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("could not delete temp file %s", err)
	}
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Printf("could not close file %s", err)
	}
}
