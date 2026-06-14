package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// freeLocalPort kills any process currently listening on the given local TCP
// port (except this app itself) so a forward can take it over. This mirrors
// the original dev-tunnel.sh, which killed stale tunnels before opening new
// ones. Relies on `lsof`, which ships with macOS.
func freeLocalPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		return // lsof exits non-zero when nothing is listening
	}

	self := os.Getpid()
	killed := false
	for _, field := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(field)
		if err != nil || pid == self {
			continue
		}
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Signal(syscall.SIGKILL)
			killed = true
		}
	}
	if killed {
		time.Sleep(250 * time.Millisecond) // let the OS release the socket
	}
}
