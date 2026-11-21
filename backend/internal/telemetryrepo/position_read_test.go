package telemetryrepo

import (
	"strings"
	"testing"
)

func TestLatestPositionTimeFollowsCoordinates(t *testing.T) {
	if !strings.Contains(latestPositionsSQL, "l.lng, l.lat, l.be_time AS be_time") {
		t.Fatalf("position time must belong to the coordinates: %s", latestPositionsSQL)
	}
}
