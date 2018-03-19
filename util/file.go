package util

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ReadFileAsJSON reads the contents of a file as a JSON.
func ReadFileAsJSON(fullPath string) (map[string]interface{}, error) {
	fileBytes, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load file: %s", err)
	}

	contents := make(map[string]interface{})
	err = json.Unmarshal(fileBytes, &contents)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal file contents as JSON: %s", err)
	}
	return contents, nil
}

// CreateFile creates a file only it doesn't exist.
func CreateFile(fullPath string) error {
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("Failed to create file: %s", err)
			}

			defer file.Close()
			return nil
		}
	}
	return nil
}

// CreateJSONFile creates a file if it does not exist.
func CreateJSONFile(fullPath string) error {
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("Failed to create file: %s", err)
			}

			defer file.Close()
			_, err = file.WriteString("{}")
			return err
		}
	}
	return nil
}

// ReadFileAsBytes reads the contents of a file as JSON.
func ReadFileAsBytes(fullPath string) ([]byte, error) {
	fileBytes, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load file: %s", err)
	}
	return fileBytes, nil
}

// ReadFileAsCSV reads the contents of a file as CSV.
func ReadFileAsCSV(fullPath string) (*csv.Reader, error) {
	fileBytes, err := ioutil.ReadFile(fullPath)
	if err != nil {
		msg := fmt.Sprintf("Failed to load file: %s", err)
		return nil, errors.New(msg)
	}
	reader := csv.NewReader((strings.NewReader(string(fileBytes))))
	return reader, nil
}

// WriteFile writes data to the specified file path.
func WriteFile(data []byte, fullPath string) error {
	err := ioutil.WriteFile(fullPath, data, 0644)
	if err != nil {
		msg := fmt.Sprintf("Failed to write data to file: %s", err)
		return errors.New(msg)
	}
	return nil
}
