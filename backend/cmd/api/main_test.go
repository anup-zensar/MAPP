package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
)

// The API listens on 127.0.0.1 only, so other computers cannot reach it (constitution: local only).
func TestListen_OnlyOnThisComputer(t *testing.T) {
	listener, err := listen(t.Context(), "127.0.0.1", 0) // port 0: a free port chosen by the system
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	if ip := listener.Addr().(*net.TCPAddr).IP; !ip.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("listening on %v, want 127.0.0.1", ip)
	}
}

// A port that another program uses stops the API with a message that says what to do, on Windows too
// (research R21).
func TestListen_BusyPortSaysWhatToDo(t *testing.T) {
	other, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer other.Close()
	port := other.Addr().(*net.TCPAddr).Port

	_, err = listen(t.Context(), "127.0.0.1", port)

	want := fmt.Sprintf("Port %d is already in use — stop the other program or set APP_PORT.", port)
	if err == nil || err.Error() != want {
		t.Errorf("listen = %v, want %q", err, want)
	}
}

// On Windows, a port that another program holds on all addresses, or that Windows has reserved, fails
// with "access denied" (WSAEACCES, 10013) rather than "in use"; it gets the same message (research R21).
// The error is built here: listening on all addresses in a test would open a Windows Firewall prompt.
func TestIsPortInUse_WindowsAccessDenied(t *testing.T) {
	err := &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", syscall.Errno(10013))}

	if !isPortInUse(err) {
		t.Errorf("isPortInUse(%v) = false, want true", err)
	}
}
