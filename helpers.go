package scheduler

import (
	"fmt"
	"runtime"
)

func max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

// Use as defer traceExit().
func traceExit() {
	pc, _, line, ok := runtime.Caller(1) // Get the caller of this function
	if ok {
		fmt.Printf(
			"exiting function %s at line %d.\n",

			runtime.FuncForPC(pc).Name(),
			line,
		)
	}
}
