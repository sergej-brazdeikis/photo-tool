package domain

import (
	"fmt"
	"strings"
)

// CollectionGrouping selects how collection detail partitions assets (Story 2.8 FR-23/24).
type CollectionGrouping int

const (
	CollectionGroupStars CollectionGrouping = iota
	CollectionGroupDay
	CollectionGroupCamera
)

// ParseCollectionGrouping parses a persisted UI string (case-insensitive).
func ParseCollectionGrouping(s string) (CollectionGrouping, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "stars", "star":
		return CollectionGroupStars, nil
	case "day", "by_day", "calendar":
		return CollectionGroupDay, nil
	case "camera", "by_camera":
		return CollectionGroupCamera, nil
	default:
		return 0, fmt.Errorf("unknown collection grouping %q", s)
	}
}

func (g CollectionGrouping) String() string {
	switch g {
	case CollectionGroupStars:
		return "stars"
	case CollectionGroupDay:
		return "day"
	case CollectionGroupCamera:
		return "camera"
	default:
		return "stars"
	}
}
