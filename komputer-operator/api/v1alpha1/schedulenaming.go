package v1alpha1

import (
	"crypto/sha256"
	"math/rand/v2"
	"strings"
	"time"
)

// MaxRunAgentNameLen keeps a per-run agent name, plus a collision suffix and the
// suffixes derived from it (pod, PVC), within the 63-char k8s name and label limit.
const MaxRunAgentNameLen = 50

const nameChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// RunAgentName names the agent for one run of a Date or Random schedule. The
// operator (cron runs) and the API (manual triggers) both name agents with it.
// Date uses runTime in the schedule's timezone (UTC if unset or invalid);
// Random uses randomPart.
func RunAgentName(scheduleName, timezone string, naming ScheduleAgentNaming, runTime time.Time, randomPart string) string {
	suffix := "-" + randomPart
	if naming == ScheduleAgentNamingDate {
		loc := time.UTC
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
		suffix = runTime.In(loc).Format("-20060102-1504")
	}
	base := scheduleName
	if len(base)+len(suffix) > MaxRunAgentNameLen {
		base = strings.TrimRight(base[:MaxRunAgentNameLen-len(suffix)], "-")
	}
	return base + suffix
}

// RandomNameSuffix returns n random lowercase letters and digits.
func RandomNameSuffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = nameChars[rand.IntN(len(nameChars))]
	}
	return string(b)
}

// NameSuffix returns n lowercase letters and digits derived from seed, so a
// retried run computes the same agent name instead of creating a second agent.
func NameSuffix(seed string, n int) string {
	sum := sha256.Sum256([]byte(seed))
	b := make([]byte, n)
	for i := range b {
		b[i] = nameChars[int(sum[i])%len(nameChars)]
	}
	return string(b)
}
