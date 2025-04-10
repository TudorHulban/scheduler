package scheduler

import (
	"fmt"
	"sort"
	"strings"
)

type ResourcesPerTimeInterval map[TimeInterval][]*Resource

func (rpt ResourcesPerTimeInterval) String() string {
	var sb strings.Builder
	sb.WriteString("_ResourcesPerTimeInterval{\n")

	// Sort keys for consistent output
	intervals := make([]TimeInterval, 0, len(rpt))
	for interval := range rpt {
		intervals = append(intervals, interval)
	}

	sort.Slice(
		intervals,
		func(i, j int) bool {
			if intervals[i].TimeStart != intervals[j].TimeStart {
				return intervals[i].TimeStart < intervals[j].TimeStart
			}
			return intervals[i].TimeEnd < intervals[j].TimeEnd
		},
	)

	for _, interval := range intervals {
		resources := rpt[interval]
		sb.WriteString(fmt.Sprintf("\t%v: []*Resource{\n", interval))

		for _, resource := range resources {
			if resource != nil {
				// Indent the resource string and add newlines after each line
				resourceStr := resource.String()
				resourceStr = strings.ReplaceAll(resourceStr, "\n", "\n\t\t")
				sb.WriteString("\t\t" + resourceStr + ",\n")
			} else {
				sb.WriteString("\t\tnil,\n")
			}
		}
		sb.WriteString("\t},\n")
	}

	sb.WriteString("}")

	return sb.String()
}

type ResourcesPerType map[uint8][]*Resource

func (rpt ResourcesPerType) String() string {
	var sb strings.Builder
	sb.WriteString("ResourcesPerType{\n")

	// Sort keys for consistent output
	types := make([]uint8, 0, len(rpt))

	for t := range rpt {
		types = append(types, t)
	}

	sort.Slice(
		types,
		func(i, j int) bool {
			return types[i] < types[j]
		},
	)

	for _, t := range types {
		resources := rpt[t]
		sb.WriteString(fmt.Sprintf("\t%d: []*Resource{\n", t))

		for _, resource := range resources {
			if resource != nil {
				// Indent the resource string and add newlines after each line
				resourceStr := resource.String()
				resourceStr = strings.ReplaceAll(resourceStr, "\n", "\n\t\t")
				sb.WriteString("\t\t" + resourceStr + ",\n")
			} else {
				sb.WriteString("\t\tnil,\n")
			}
		}

		sb.WriteString("\t},\n")
	}

	sb.WriteString("}")

	return sb.String()
}
