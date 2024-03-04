package utils

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort(ctx context.Context) (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			port := l.Addr().(*net.TCPAddr).Port
			slog.DebugContext(ctx, fmt.Sprintf("reserving port %d", port))

			defer l.Close()
			return port, nil
		}
	}
	return
}
