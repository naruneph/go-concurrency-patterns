package contextio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"orchestration_contextio_net",
		"Minimal TCP echo with context-aware Read/Write via deadlines; shows success then a timed-out read.",
		ContextIONetExample,
	)
}

// serveEcho accepts one connection and echoes lines.
func serveEcho(ln net.Listener) (retErr error) {
	c, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accept: %w", err)
	}
	defer func() {
		if cerr := c.Close(); cerr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("conn close: %w", cerr))
		}
	}()

	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			// client closed or timed out on its side
			return err
		}
		if _, err := c.Write([]byte("echo: " + line)); err != nil {
			return err
		}
	}
}

// readContext ties the connection's read deadline to ctx.
func readContext(ctx context.Context, c net.Conn, p []byte) (int, error) {
	// Clear any previous deadline.
	_ = c.SetReadDeadline(time.Time{})

	// If ctx has a deadline, use it; otherwise bump the deadline when ctx cancels.
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetReadDeadline(dl)
	} else {
		go func() {
			<-ctx.Done()
			_ = c.SetReadDeadline(time.Now())
		}()
	}
	return c.Read(p)
}

// writeContext ties the connection's write deadline to ctx.
func writeContext(ctx context.Context, c net.Conn, p []byte) (int, error) {
	_ = c.SetWriteDeadline(time.Time{})
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetWriteDeadline(dl)
	} else {
		go func() {
			<-ctx.Done()
			_ = c.SetWriteDeadline(time.Now())
		}()
	}
	return c.Write(p)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// ContextIONetExample spins a tiny echo server, then a client that uses ctx-bound
// deadlines to avoid hanging reads/writes.
func ContextIONetExample() {
	var runErr error

	// Start a tiny echo server on a random local port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(err)

	defer func() {
		if cerr := ln.Close(); cerr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("listener close: %w", cerr))
		}
		if runErr != nil {
			fmt.Printf("contextio-net: run completed with error: %v\n", runErr)
		}
	}()
	fmt.Println("listening on", ln.Addr())

	// accept one client and echo lines
	go func() {
		if err := serveEcho(ln); err != nil {
			fmt.Printf("serveEcho error: %v\n", err)
		}
	}()

	// Client dials the server.
	conn, err := net.Dial("tcp", ln.Addr().String())
	must(err)
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("client conn close: %w", cerr))
		}
		if runErr != nil {
			fmt.Printf("contextio-net: run completed with error: %v\n", runErr)
		}
	}()

	// 1) Happy path round-trip under 500 ms.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		if _, err := writeContext(ctx, conn, []byte("ping\n")); err != nil {
			fmt.Println("client write error:", err)
			return
		}
		buf := make([]byte, 64)
		n, err := readContext(ctx, conn, buf)
		fmt.Printf("client read1: n=%d err=%v data=%q\n", n, err, string(buf[:n]))
	}

	// 2) No server reply now; a 300 ms read should time out via ctx deadline.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		buf := make([]byte, 64)
		start := time.Now()
		n, err := readContext(ctx, conn, buf)
		fmt.Printf("client read2: n=%d err=%v elapsed=%v\n", n, err, time.Since(start).Round(10*time.Millisecond))
	}

	fmt.Println("contextio-net: done")
}
