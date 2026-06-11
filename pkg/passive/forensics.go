package passive

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/SaaSAILabs/git-attest/pkg/active"
	"github.com/SaaSAILabs/git-attest/pkg/util"
)

type TimelineMetrics struct {
	ConstructionSpread time.Duration
	DeliberationGap    time.Duration
	Events             []active.FlightEvent
	Forensics          ForensicProfile
}

type ForensicProfile struct {
	AvgModIntervalMs int64 `json:"avg_mod_interval_ms"`
	MaxModIntervalMs int64 `json:"max_mod_interval_ms"`
	MinModIntervalMs int64 `json:"min_mod_interval_ms"`
	CtimeDriftFiles  int   `json:"files_with_ctime_drift"`
	CtimeDriftMaxMs  int64 `json:"max_ctime_mtime_drift_ms"`
	NewFileCount     int   `json:"newly_created_file_count"`
	TotalFiles       int   `json:"total_staged_files"`
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

	profile := ForensicProfile{
		TotalFiles: n,
	}

	if n > 1 {
		var totalInterval int64
		var maxInterval int64
		var minInterval int64 = -1

		for i := 1; i < n; i++ {
			interval := stamps[i].Mtime.Sub(stamps[i-1].Mtime).Milliseconds()
			totalInterval += interval
			if interval > maxInterval {
				maxInterval = interval
			}
			if minInterval == -1 || interval < minInterval {
				minInterval = interval
			}
		}

		if minInterval == -1 {
			minInterval = 0
		}

		profile.AvgModIntervalMs = totalInterval / int64(n-1)
		profile.MaxModIntervalMs = maxInterval
		profile.MinModIntervalMs = minInterval
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
