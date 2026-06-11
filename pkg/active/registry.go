package active

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/SaaSAILabs/git-attest/pkg/privacy"
)

// TimeWindow represents the period between the previous commit and the current commit.
type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// Overlaps checks if a session's active period (birth to mtime) overlaps with the window.
func (w TimeWindow) Overlaps(sessionBirth, sessionMtime time.Time) bool {
	return !sessionMtime.Before(w.Start) && !sessionBirth.After(w.End)
}

// Extractor defines the interface for harvesting FlightEvents from an AI tool's logs.
type Extractor interface {
	Name() string
	Extract(window TimeWindow) ([]FlightEvent, error)
}

// HarvestAll runs every registered extractor, merges successful results,
// and returns a chronologically sorted event array.
func HarvestAll(window TimeWindow) []FlightEvent {
	extractors := []Extractor{
		&ClaudeExtractor{},
		&AntigravityExtractor{},
		&CopilotExtractor{},
		&CursorExtractor{},
	}

	var merged []FlightEvent
	for _, ext := range extractors {
		events, err := ext.Extract(window)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[attest] %s: %v\n", ext.Name(), err)
			continue
		}
		merged = append(merged, events...)
	}

	redactor := privacy.NewRedactor()
	_ = redactor.LoadTraceFilter(".tracefilter")

	for i := range merged {
		if prompt, ok := merged[i].Meta["prompt"].(string); ok {
			merged[i].Meta["prompt"] = redactor.Redact(prompt)
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp < merged[j].Timestamp
	})

	return merged
}

// --- Claude Code Extractor ---

type ClaudeExtractor struct{}

func (c *ClaudeExtractor) Name() string { return "claude_code" }

func (c *ClaudeExtractor) Extract(window TimeWindow) ([]FlightEvent, error) {
	paths, err := FindRelevantSessions(window)
	if err != nil {
		return nil, err
	}
	
	var allEvents []FlightEvent
	for _, path := range paths {
		events, err := ParseSessionFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[attest] warning: failed to parse %s: %v\n", path, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}
	return allEvents, nil
}


