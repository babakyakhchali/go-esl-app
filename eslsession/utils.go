package eslsession

import (
	"fmt"
	"runtime"
)

func getMemStats() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("Alloc:%d bytes HeapAlloc:%d bytes", m.Alloc, m.HeapAlloc)
}

func dumpAllRoutines() string {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	return fmt.Sprintf("%s", buf)
}
