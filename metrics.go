package main

import "strings"

func (metrics Metrics) findGroups() {
	for i := range metrics {
		parts := strings.SplitN(metrics[i].Name, "_", 3)
		if len(parts) < 1 {
			continue
		}
		metrics[i].Group = parts[0]
		if len(parts) < 2 {
			continue
		}
		metrics[i].Subgroup = parts[1]
	}
}
