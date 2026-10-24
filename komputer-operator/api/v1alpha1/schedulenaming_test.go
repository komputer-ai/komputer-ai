package v1alpha1

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRunAgentName(t *testing.T) {
	// 09:00 UTC is 12:00 in Asia/Jerusalem (UTC+3 in October).
	run := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	if got := RunAgentName("daily-report", "Asia/Jerusalem", ScheduleAgentNamingDate, run, "x"); got != "daily-report-20261024-1200" {
		t.Errorf("Date: %q, want the run time in the schedule's timezone", got)
	}
	if got := RunAgentName("daily-report", "", ScheduleAgentNamingDate, run, "x"); got != "daily-report-20261024-0900" {
		t.Errorf("Date, no timezone: %q, want UTC", got)
	}
	if got := RunAgentName("daily-report", "", ScheduleAgentNamingRandom, run, "k3x9q"); got != "daily-report-k3x9q" {
		t.Errorf("Random: %q", got)
	}
	long := strings.Repeat("a", 35) + "-" + strings.Repeat("b", 30)
	got := RunAgentName(long, "", ScheduleAgentNamingDate, run, "x")
	if len(got) > MaxRunAgentNameLen || !strings.HasSuffix(got, "-20261024-0900") || strings.Contains(got, "--") {
		t.Errorf("long name: %q (%d chars), want at most %d ending in the date", got, len(got), MaxRunAgentNameLen)
	}
}

func TestNameSuffixes(t *testing.T) {
	valid := regexp.MustCompile(`^[a-z0-9]{5}$`)
	if a, b := NameSuffix("uid/2026-10-24T09:00:00Z", 5), NameSuffix("uid/2026-10-24T09:00:00Z", 5); a != b || !valid.MatchString(a) {
		t.Errorf("NameSuffix must be stable for a seed: %q vs %q", a, b)
	}
	if NameSuffix("uid/2026-10-24T09:00:00Z", 5) == NameSuffix("uid/2026-10-24T10:00:00Z", 5) {
		t.Error("different runs should get different suffixes")
	}
	if s := RandomNameSuffix(5); !valid.MatchString(s) {
		t.Errorf("RandomNameSuffix = %q", s)
	}
}
