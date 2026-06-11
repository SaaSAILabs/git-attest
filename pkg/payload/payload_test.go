package payload

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SaaSAILabs/git-attest/pkg/active"
	"github.com/SaaSAILabs/git-attest/pkg/passive"
)

func TestBuild_MergesAndSortsEvents(t *testing.T) {
	promptEvents := []active.FlightEvent{
		{Timestamp: 3000, Type: "agent_prompt", Meta: map[string]interface{}{"prompt": "fix auth"}},
		{Timestamp: 1000, Type: "agent_prompt", Meta: map[string]interface{}{"prompt": "add login"}},
	}

	metrics := &passive.TimelineMetrics{
		ConstructionSpread: 500 * time.Millisecond,
		DeliberationGap:    12 * time.Minute,
		Events: []active.FlightEvent{
			{Timestamp: 2000, Type: "file_modification", Meta: map[string]interface{}{"file": "auth.go"}},
		},
	}

	commitTime := time.UnixMilli(5000)
	p := Build(promptEvents, metrics, commitTime)

	// Verify metadata.
	if p.Version != payloadVersion {
		t.Errorf("Version = %q, want %q", p.Version, payloadVersion)
	}
	if p.CommitTimestamp != 5000 {
		t.Errorf("CommitTimestamp = %d, want 5000", p.CommitTimestamp)
	}

	// Verify summary metrics.
	if p.Summary.ConstructionSpreadMs != 500 {
		t.Errorf("ConstructionSpreadMs = %d, want 500", p.Summary.ConstructionSpreadMs)
	}
	if p.Summary.DeliberationGapMs != 720000 {
		t.Errorf("DeliberationGapMs = %d, want 720000", p.Summary.DeliberationGapMs)
	}

	// Verify merged event count.
	if p.Summary.TotalEvents != 3 {
		t.Fatalf("TotalEvents = %d, want 3", p.Summary.TotalEvents)
	}
	if len(p.FlightRecorder) != 3 {
		t.Fatalf("FlightRecorder len = %d, want 3", len(p.FlightRecorder))
	}

	// Verify chronological sort: 1000 → 2000 → 3000.
	for i := 1; i < len(p.FlightRecorder); i++ {
		if p.FlightRecorder[i].Timestamp < p.FlightRecorder[i-1].Timestamp {
			t.Errorf("events not sorted: [%d].Timestamp=%d < [%d].Timestamp=%d",
				i, p.FlightRecorder[i].Timestamp, i-1, p.FlightRecorder[i-1].Timestamp)
		}
	}
}

func TestSerialize_ProducesValidJSON(t *testing.T) {
	p := &GitNotePayload{
		Version:         "0.1.0",
		Profile:         "claude_code",
		CommitTimestamp: 5000,
		Summary: SummaryMetrics{
			ConstructionSpreadMs: 100,
			DeliberationGapMs:    200,
			TotalEvents:          1,
		},
		FlightRecorder: []active.FlightEvent{
			{Timestamp: 1000, Type: "agent_prompt", Meta: map[string]interface{}{"prompt": "test"}},
		},
	}

	jsonStr, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	// Verify it's valid JSON by round-tripping.
	var decoded GitNotePayload
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Fatalf("produced invalid JSON: %v", err)
	}
	if decoded.Version != "0.1.0" {
		t.Errorf("round-trip Version = %q, want %q", decoded.Version, "0.1.0")
	}
	if decoded.Summary.TotalEvents != 1 {
		t.Errorf("round-trip TotalEvents = %d, want 1", decoded.Summary.TotalEvents)
	}
}

func TestBuild_EmptyInputs(t *testing.T) {
	metrics := &passive.TimelineMetrics{}
	p := Build(nil, metrics, time.UnixMilli(1000))

	if len(p.FlightRecorder) != 0 {
		t.Errorf("expected 0 events, got %d", len(p.FlightRecorder))
	}
	if p.Summary.TotalEvents != 0 {
		t.Errorf("TotalEvents = %d, want 0", p.Summary.TotalEvents)
	}
}

func TestSortEvents_Stability(t *testing.T) {
	events := []active.FlightEvent{
		{Timestamp: 5000, Type: "c"},
		{Timestamp: 1000, Type: "a"},
		{Timestamp: 3000, Type: "b"},
	}
	SortEvents(events)

	expected := []int64{1000, 3000, 5000}
	for i, want := range expected {
		if events[i].Timestamp != want {
			t.Errorf("events[%d].Timestamp = %d, want %d", i, events[i].Timestamp, want)
		}
	}
}
