package active

import (
	"os"
	"path/filepath"
)

type CursorExtractor struct{}

func (c *CursorExtractor) Name() string { return "cursor" }

func (c *CursorExtractor) Extract(window TimeWindow) ([]FlightEvent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	basePath := filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	
	// Test override
	if testPath := os.Getenv("ATTEST_TEST_CURSOR_DB_PATH"); testPath != "" {
		// Just use the provided path for testing
		basePath = testPath
	}

	return ParseVSCodeChatFiles(basePath, window, "cursor")
}
