package controller

import "time"

// copyOutTimeout bounds one copy-out (exec + upload).
var copyOutTimeout = 3 * time.Minute
