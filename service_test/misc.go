package service

import (
	"einheit/boltkit/service"
	"path/filepath"
	"runtime"
	"strings"
)

// setup initialises the server modules, this is to be used when running
// tests.
func setup() error {
	if service.App == nil {
		// Get the root directory.
		_, filename, _, _ := runtime.Caller(0)
		testDir := filepath.Dir(filename)
		pathChunks := strings.Split(testDir, string(filepath.Separator))
		rootDir := strings.Join(pathChunks[0:len(pathChunks)-1],
			string(filepath.Separator))

		// Setup test environment.
		var err error
		service.App, err = service.NewService(filepath.Join(rootDir, "config.json"))
		if err != nil {
			return err
		}
	}
	return nil
}
