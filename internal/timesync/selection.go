package timesync

import (
	"fmt"
	"sort"
	"time"
)

type TimeUpdate struct {
	Offset      float64
	SourceCount int
	BestTime    time.Time
}

func SelectBestTime(sources []*TimeSource, minSources int, maxSpread time.Duration) (*TimeUpdate, error) {
	if len(sources) < minSources {
		return nil, fmt.Errorf("insufficient sources: have %d, need %d", len(sources), minSources)
	}

	var validSources []*TimeSource
	for _, s := range sources {
		if s.ClockInfo.Stratum > 15 {
			continue
		}
		if s.ClockInfo.RootDispersion > 1.0 {
			continue
		}
		validSources = append(validSources, s)
	}

	if len(validSources) < minSources {
		return nil, fmt.Errorf("insufficient valid sources after filtering")
	}

	sort.Slice(validSources, func(i, j int) bool {
		return validSources[i].Timestamp < validSources[j].Timestamp
	})

	minTime := validSources[0].Timestamp
	maxTime := validSources[len(validSources)-1].Timestamp
	spread := time.Duration(maxTime - minTime)

	if spread > maxSpread {
		agreeing := findAgreeingSubset(validSources, maxSpread/2)
		if len(agreeing) < minSources {
			return nil, fmt.Errorf("sources disagree (spread: %v)", spread)
		}
		validSources = agreeing
	}

	var totalWeight float64
	var weightedTime float64

	for _, src := range validSources {
		weight := 1.0 / (src.ClockInfo.RootDispersion + 0.0001)
		weightedTime += float64(src.Timestamp) * weight
		totalWeight += weight
	}

	bestTimeNanos := int64(weightedTime / totalWeight)
	bestTime := time.Unix(0, bestTimeNanos)
	localTime := time.Now()
	offset := bestTime.Sub(localTime).Seconds()

	return &TimeUpdate{
		Offset:      offset,
		SourceCount: len(validSources),
		BestTime:    bestTime,
	}, nil
}

func findAgreeingSubset(sources []*TimeSource, tolerance time.Duration) []*TimeSource {
	if len(sources) == 0 {
		return nil
	}

	var bestSubset []*TimeSource
	bestSize := 0

	for i := 0; i < len(sources); i++ {
		var subset []*TimeSource
		for j := i; j < len(sources); j++ {
			if time.Duration(sources[j].Timestamp-sources[i].Timestamp) <= tolerance {
				subset = append(subset, sources[j])
			} else {
				break
			}
		}
		if len(subset) > bestSize {
			bestSize = len(subset)
			bestSubset = subset
		}
	}

	return bestSubset
}
