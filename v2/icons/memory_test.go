package icons

import (
	"runtime"
	"testing"
)

func TestMemoryUsage(t *testing.T) {
	var m runtime.MemStats

	// Force GC to get clean measurement
	runtime.GC()
	runtime.ReadMemStats(&m)
	before := m.Alloc

	// Access the package data to ensure it's loaded
	_ = awsShapes
	groups := AllAWSGroups()
	t.Logf("Number of groups: %d", len(groups))

	totalShapes := 0
	for _, g := range groups {
		shapes, _ := AWSShapesInGroup(g)
		totalShapes += len(shapes)
	}
	t.Logf("Total shapes: %d", totalShapes)

	runtime.GC()
	runtime.ReadMemStats(&m)
	after := m.Alloc

	// The package data is already loaded in init(), so this measures current heap
	t.Logf("Memory allocated for package data: ~%d KB", (after-before)/1024)
	t.Logf("Total heap allocation: ~%d KB", after/1024)

	// The embedded JSON is 225KB (220K), we expect parsed map to be around 750KB-1MB
	// This is acceptable per design requirements
	if after > 2*1024*1024 {
		t.Errorf("Memory usage exceeds expected range: %d KB (expected < 2 MB)", after/1024)
	}
}
