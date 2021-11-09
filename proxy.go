package tcpproxy

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func ToUnix(addr string) (net.Listener, error) {
	socketPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().UnixNano(), 10))
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to create unix listening socket: %w", err)
	}
	command := exec.Command("tcpproxy", "-i", addr+":"+socketPath)
	if os.Getenv("PROXY_DEBUG") == "1" {
		command.Stderr = os.Stderr
	}
	stdoutPipe, _ := command.StdoutPipe()
	command.StdinPipe() //use stdin to signal parent termination

	if err := command.Start(); err != nil {
		l.Close()
		return nil, fmt.Errorf("Failed to start tcpproxy: %w", err)
	}
	io.ReadAll(stdoutPipe) // signal listeners are up and running
	go func() {
		command.Wait() //when the proxy command terminates, close unix socket
		l.Close()
	}()
	return &listener{Listener: l, cmd: command}, nil
}

type listener struct {
	net.Listener
	cmd *exec.Cmd
}

func (l *listener) Close() error {
	l.cmd.Process.Kill()
	return l.Listener.Close()
}
