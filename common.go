package mp3joiner

import (
	"log"
	"os"
)

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
