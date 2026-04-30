package controller

// mergeLabels merges user-supplied labels with system-managed labels.
// System labels are written last and overwrite any user value, so users
// cannot override internal labels like komputer.ai/agent-name.
//
// Always returns a non-nil map (possibly empty) so callers can safely assign
// to ObjectMeta.Labels without checking for nil.
func mergeLabels(user, system map[string]string) map[string]string {
	out := make(map[string]string, len(user)+len(system))
	for k, v := range user {
		out[k] = v
	}
	for k, v := range system {
		out[k] = v
	}
	return out
}
