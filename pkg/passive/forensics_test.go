package passive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SaaSAILabs/attest-cli.git/pkg/util"
)

// createTempFileWithMtime creates a temp file and sets its mtime to the given time.
func createTempFileWithMtime(t *testing.T, dir, name string, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create temp file %s: %v", name, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("failed to set mtime on %s: %v", name, err)
	}
	return path
}

func TestCalculateTimelineMetrics_SingleFile(t *testing.T) {
	dir := t.TempDir()
	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	f := createTempFileWithMtime(t, dir, "only.go", baseTime)

	commitTime := baseTime.Add(3 * time.Second)
	metrics, err := CalculateTimelineMetrics([]string{f}, commitTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.ConstructionSpread != 0 {
		t.Errorf("ConstructionSpread = %v, want 0", metrics.ConstructionSpread)
	}
	if metrics.DeliberationGap != 3*time.Second {
		t.Errorf("DeliberationGap = %v, want 3s", metrics.DeliberationGap)
	}
	if len(metrics.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(metrics.Events))
	}
	if metrics.Events[0].Type != "file_modification" {
		t.Errorf("event type = %q, want %q", metrics.Events[0].Type, "file_modification")
	}
}

func TestCalculateTimelineMetrics_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	// T1=1000ms, T2=1200ms, T3=1500ms offsets from a base time.
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	t1 := base.Add(1000 * time.Millisecond)
	t2 := base.Add(1200 * time.Millisecond)
	t3 := base.Add(1500 * time.Millisecond)

	f1 := createTempFileWithMtime(t, dir, "a.go", t1)
	f2 := createTempFileWithMtime(t, dir, "b.go", t2)
	f3 := createTempFileWithMtime(t, dir, "c.go", t3)

	// Feed files out of order to verify sorting.
	commitTime := base.Add(2000 * time.Millisecond)
	metrics, err := CalculateTimelineMetrics([]string{f3, f1, f2}, commitTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ConstructionSpread: T3 - T1 = 500ms
	if metrics.ConstructionSpread != 500*time.Millisecond {
		t.Errorf("ConstructionSpread = %v, want 500ms", metrics.ConstructionSpread)
	}

	// DeliberationGap: commitTime - T3 = 500ms
	if metrics.DeliberationGap != 500*time.Millisecond {
		t.Errorf("DeliberationGap = %v, want 500ms", metrics.DeliberationGap)
	}

	// Events should be sorted chronologically.
	if len(metrics.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(metrics.Events))
	}
	if metrics.Events[0].Timestamp > metrics.Events[1].Timestamp {
		t.Error("events are not sorted chronologically")
	}
	if metrics.Events[1].Timestamp > metrics.Events[2].Timestamp {
		t.Error("events are not sorted chronologically")
	}
}

func TestCalculateTimelineMetrics_Deliberation(t *testing.T) {
	dir := t.TempDir()

	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	t1 := base.Add(1000 * time.Millisecond)
	t2 := base.Add(1200 * time.Millisecond)
	t3 := base.Add(1500 * time.Millisecond)

	f1 := createTempFileWithMtime(t, dir, "x.go", t1)
	f2 := createTempFileWithMtime(t, dir, "y.go", t2)
	f3 := createTempFileWithMtime(t, dir, "z.go", t3)

	// Simulate a 12-minute review gap.
	commitTime := t3.Add(12 * time.Minute)
	metrics, err := CalculateTimelineMetrics([]string{f1, f2, f3}, commitTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.DeliberationGap != 12*time.Minute {
		t.Errorf("DeliberationGap = %v, want 12m", metrics.DeliberationGap)
	}
}

func TestCalculateTimelineMetrics_EmptyFiles(t *testing.T) {
	metrics, err := CalculateTimelineMetrics([]string{}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if metrics.ConstructionSpread != 0 {
		t.Errorf("ConstructionSpread = %v, want 0", metrics.ConstructionSpread)
	}
	if metrics.DeliberationGap != 0 {
		t.Errorf("DeliberationGap = %v, want 0", metrics.DeliberationGap)
	}
	if len(metrics.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(metrics.Events))
	}
}

func TestCalculateTimelineMetrics_MissingFile(t *testing.T) {
	_, err := CalculateTimelineMetrics([]string{"/nonexistent/file.go"}, time.Now())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestComputeForensicProfile_MachineBatch(t *testing.T) {
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	stamps := []util.FileTimestamps{
		{Mtime: base},
		{Mtime: base},
		{Mtime: base},
	}
	
	profile := computeForensicProfile(stamps)
	if profile.AvgModIntervalMs != 0 || profile.MaxModIntervalMs != 0 || profile.MinModIntervalMs != 0 {
		t.Errorf("expected 0 intervals for machine batch, got avg=%v max=%v min=%v", profile.AvgModIntervalMs, profile.MaxModIntervalMs, profile.MinModIntervalMs)
	}
}

func TestComputeForensicProfile_HumanSequential(t *testing.T) {
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	stamps := []util.FileTimestamps{
		{Mtime: base},
		{Mtime: base.Add(1 * time.Second)}, // 1s
		{Mtime: base.Add(4 * time.Second)}, // 3s
		{Mtime: base.Add(10 * time.Second)}, // 6s
	}
	
	profile := computeForensicProfile(stamps)
	if profile.MinModIntervalMs != 1000 {
		t.Errorf("expected MinModIntervalMs 1000, got %v", profile.MinModIntervalMs)
	}
	if profile.MaxModIntervalMs != 6000 {
		t.Errorf("expected MaxModIntervalMs 6000, got %v", profile.MaxModIntervalMs)
	}
	if profile.AvgModIntervalMs != (1000+3000+6000)/3 {
		t.Errorf("expected AvgModIntervalMs 3333, got %v", profile.AvgModIntervalMs)
	}
}

func TestComputeForensicProfile_Drift(t *testing.T) {
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	
	// Mock an architectural edit: metadata variables shift (recent ctime) 
	// while contents remain locked (old mtime)
	oldMtime := base.Add(-24 * time.Hour)
	recentCtime := base
	
	stamps := []util.FileTimestamps{
		{
			Mtime: oldMtime,
			Ctime: recentCtime,
			Btime: oldMtime,
		},
	}
	
	profile := computeForensicProfile(stamps)
	
	if profile.CtimeDriftFiles != 1 {
		t.Errorf("expected 1 drifted file, got %v", profile.CtimeDriftFiles)
	}
	
	expectedDrift := recentCtime.Sub(oldMtime).Milliseconds()
	if profile.CtimeDriftMaxMs != expectedDrift {
		t.Errorf("expected drift %v, got %v", expectedDrift, profile.CtimeDriftMaxMs)
	}
}
