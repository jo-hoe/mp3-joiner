package internal

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
)

func calculateFileHash(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return make([]byte, 0), err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return make([]byte, 0), err
	}
	return hasher.Sum(nil), err
}

func areFileEqual(leftFilePath string, rightFilePath string) (bool, error) {
	leftFileHash, err := calculateFileHash(leftFilePath)
	if err != nil {
		return false, err
	}
	rightFileHash, err := calculateFileHash(rightFilePath)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftFileHash, rightFileHash), nil
}

func doesFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func MoveFile(sourcePath, targetPath string) (err error) {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	if doesFileExist(targetPath) {
		// check if this is the same file
		filesEqual, err := areFileEqual(sourcePath, targetPath)
		if err != nil {
			return err
		}
		if filesEqual {
			// same file already exists and can be removed from source
			err = inputFile.Close()
			if err != nil {
				return err
			}
			err = os.Remove(sourcePath)
			if err != nil {
				return err
			}
			// stop process
			return nil
		} else {
			// remove destination file and continue
			err = os.Remove(targetPath)
			if err != nil {
				return err
			}
		}
	}

	outputFile, err := os.Create(targetPath)
	if err != nil {
		inputFile.Close()
		return err
	}
	defer func() {
		fileClosingError := outputFile.Close()
		if fileClosingError != nil {
			return
		}

		// check if copying was successfull
		if err != nil {
			return
		}

		// currently double-check of file hash
		// target/source file has been omitted

		// The copy was successful, so now delete the original file
		err = os.Remove(sourcePath)
	}()

	// actual file copy
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return err
	}

	return nil
}
