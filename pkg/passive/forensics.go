package passive

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/SaaSAILabs/attest-cli.git/pkg/active"
	"github.com/SaaSAILabs/attest-cli.git/pkg/util"
)

type TimelineMetrics struct {
	ConstructionSpread time.Duration
	DeliberationGap    time.Duration
	Events             []active.FlightEvent
	Forensics          ForensicProfile
}

type ForensicProfile struct {
	StaggerIndex    float64 `json:"mtime_stagger_ratio_0_to_1"`
	CtimeDriftFiles int     `json:"files_with_ctime_drift"`
	CtimeDriftMaxMs int64   `json:"max_ctime_mtime_drift_ms"`
	NewFileCount    int     `json:"newly_created_file_count"`
	TotalFiles      int     `json:"total_staged_files"`
}

func CalculateTimelineMetrics(files []string, commitTime time.Time) (*TimelineMetrics, error) {
	if len(files) == 0 {
		return &TimelineMetrics{}, nil
	}

	type fileData struct {
		path   string
		stamps util.FileTimestamps
	}

	entries := make([]fileData, 0, len(files))
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return nil, fmt.Errorf("unable to stat %s: %w", f, err)
		}
		entries = append(entries, fileData{path: f, stamps: util.GetFileTimestamps(info)})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].stamps.Mtime.Before(entries[j].stamps.Mtime)
	})

	first := entries[0].stamps.Mtime
	last := entries[len(entries)-1].stamps.Mtime
	spread := last.Sub(first)
	gap := commitTime.Sub(last)

	events := make([]active.FlightEvent, 0, len(entries))
	stamps := make([]util.FileTimestamps, 0, len(entries))
	for _, e := range entries {
		events = append(events, active.FlightEvent{
			Timestamp: e.stamps.Mtime.UnixMilli(),
			Type:      "file_modification",
			Meta: map[string]interface{}{
				"file":  e.path,
				"mtime": e.stamps.Mtime.UnixMilli(),
				"ctime": e.stamps.Ctime.UnixMilli(),
				"btime": e.stamps.Btime.UnixMilli(),
			},
		})
		stamps = append(stamps, e.stamps)
	}

	return &TimelineMetrics{
		ConstructionSpread: spread,
		DeliberationGap:    gap,
		Events:             events,
		Forensics:          computeForensicProfile(stamps),
	}, nil
}

func computeForensicProfile(stamps []util.FileTimestamps) ForensicProfile {
	n := len(stamps)
	if n == 0 {
		return ForensicProfile{}
	}

	mtimes := make([]time.Time, n)
	for i, s := range stamps {
		mtimes[i] = s.Mtime
	}

	profile := ForensicProfile{
		StaggerIndex: computeStaggerIndex(mtimes),
		TotalFiles:   n,
	}

	const driftThresholdMs = 100
	for _, s := range stamps {
		driftMs := s.Ctime.Sub(s.Mtime).Milliseconds()
		if driftMs > driftThresholdMs {
			profile.CtimeDriftFiles++
			if driftMs > profile.CtimeDriftMaxMs {
				profile.CtimeDriftMaxMs = driftMs
			}
		}

		if !s.Btime.IsZero() && s.Mtime.Sub(s.Btime) < 2*time.Second {
			profile.NewFileCount++
		}
	}

	return profile
}

// computeStaggerIndex measures how spread out file modification times are.
// 0.0 = all files modified in the same second (machine batch).
// 1.0 = every file modified in a distinct second (human sequential).
func computeStaggerIndex(mtimes []time.Time) float64 {
	if len(mtimes) <= 1 {
		return 0.0
	}
	buckets := make(map[int64]struct{})
	for _, t := range mtimes {
		buckets[t.Unix()] = struct{}{}
	}
	return float64(len(buckets)-1) / float64(len(mtimes)-1)
}
