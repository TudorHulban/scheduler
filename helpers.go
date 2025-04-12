package scheduler

import (
	"fmt"
	"runtime"
)

type maxIntegerTypes interface {
	uint8 | uint16 | int64
}

func max[T maxIntegerTypes](a, b T) T {
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

func ternary[T any](condition bool, value1, value2 T) T {
	if condition {
		return value1
	}

	return value2
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

func traceExitWMarker(marker string) {
	pc, _, line, ok := runtime.Caller(1) // Get the caller of this function
	if ok {
		fmt.Printf(
			"exiting function %s at line %d (marker: %s).\n",

			runtime.FuncForPC(pc).Name(),
			line,
			marker,
		)
	}
}
