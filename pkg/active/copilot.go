package active

import (
	"os"
	"path/filepath"
)

type CopilotExtractor struct{}

func (c *CopilotExtractor) Name() string { return "copilot" }

func (c *CopilotExtractor) Extract(window TimeWindow) ([]FlightEvent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	basePath := filepath.Join(home, "Library", "Application Support", "Code", "User")
	
	// Test override
	if testPath := os.Getenv("ATTEST_TEST_COPILOT_DB_PATH"); testPath != "" {
		// Just use the provided path for testing
		basePath = testPath
	}

	return ParseVSCodeChatFiles(basePath, window, "copilot")
}
