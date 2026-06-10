package active

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestCopilotExtractor_Integration(t *testing.T) {
	// Load .env.test if it exists
	godotenv.Load("../../.env.test")

	dbPath := os.Getenv("ATTEST_TEST_COPILOT_DB_PATH")
	if dbPath == "" {
		t.Skip("Skipping test because ATTEST_TEST_COPILOT_DB_PATH is not set")
	}

	ext := &CopilotExtractor{}
	window := TimeWindow{
		Start: time.Time{},
		End:   time.Now().Add(24 * time.Hour),
	}

	events, err := ext.Extract(window)
	if err != nil {
		t.Fatalf("Copilot Extract() failed: %v", err)
	}

	t.Logf("Found %d Copilot events", len(events))
}

func TestCursorExtractor_Integration(t *testing.T) {
	// Load .env.test if it exists
	godotenv.Load("../../.env.test")

	dbPath := os.Getenv("ATTEST_TEST_CURSOR_DB_PATH")
	if dbPath == "" {
		t.Skip("Skipping test because ATTEST_TEST_CURSOR_DB_PATH is not set")
	}

	ext := &CursorExtractor{}
	window := TimeWindow{
		Start: time.Time{},
		End:   time.Now().Add(24 * time.Hour),
	}

	events, err := ext.Extract(window)
	if err != nil {
		t.Fatalf("Cursor Extract() failed: %v", err)
	}

	t.Logf("Found %d Cursor events", len(events))
}
