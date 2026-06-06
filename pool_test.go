package email

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSMTPServer is a minimal SMTP server for testing the connection pool.
type fakeSMTPServer struct {
	listener   net.Listener
	addr       string
	connCount  atomic.Int64
	msgCount   atomic.Int64
	mu         sync.Mutex
	failAuth   bool
	failRSET   bool
	failAfterN int // fail DATA after N messages per connection
	closed     atomic.Bool
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := &fakeSMTPServer{
		listener: ln,
		addr:     ln.Addr().String(),
	}
	go s.serve(t)
	return s
}

func (s *fakeSMTPServer) serve(t *testing.T) {
	t.Helper()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			return
		}
		s.connCount.Add(1)
		go s.handleConn(t, conn)
	}
}

func (s *fakeSMTPServer) handleConn(t *testing.T, conn net.Conn) {
	t.Helper()
	defer conn.Close() //nolint:errcheck

	reader := bufio.NewReader(conn)
	write := func(msg string) {
		fmt.Fprintf(conn, "%s\r\n", msg)
	}

	write("220 localhost ESMTP fake")
	msgOnConn := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimRight(line, "\r\n"))

		action := s.dispatch(cmd, reader, write, &msgOnConn)
		if action == connClose {
			return
		}
	}
}

// connAction indicates what the connection loop should do after handling a command.
type connAction int

const (
	connContinue connAction = iota
	connClose
)

func (s *fakeSMTPServer) dispatch(cmd string, reader *bufio.Reader, write func(string), msgOnConn *int) connAction {
	switch {
	case strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO"):
		write("250-localhost")
		write("250-AUTH PLAIN LOGIN")
		write("250 OK")

	case strings.HasPrefix(cmd, "AUTH"):
		s.handleAuth(write)

	case strings.HasPrefix(cmd, "MAIL FROM:"):
		write("250 OK")

	case strings.HasPrefix(cmd, "RCPT TO:"):
		write("250 OK")

	case cmd == "DATA":
		if action := s.handleData(reader, write, msgOnConn); action != connContinue {
			return action
		}

	case cmd == "RSET":
		return s.handleReset(write)

	case cmd == "QUIT":
		write("221 Bye")
		return connClose

	case cmd == "NOOP":
		write("250 OK")

	case strings.HasPrefix(cmd, "STARTTLS"):
		write("502 Not implemented")

	default:
		write("500 Unrecognized command")
	}
	return connContinue
}

func (s *fakeSMTPServer) handleAuth(write func(string)) {
	s.mu.Lock()
	fail := s.failAuth
	s.mu.Unlock()
	if fail {
		write("535 Authentication failed")
	} else {
		write("235 Authentication successful")
	}
}

func (s *fakeSMTPServer) handleData(reader *bufio.Reader, write func(string), msgOnConn *int) connAction {
	s.mu.Lock()
	failN := s.failAfterN
	s.mu.Unlock()
	if failN > 0 && *msgOnConn >= failN {
		write("452 Too many messages")
		return connContinue
	}
	write("354 Start mail input")
	for {
		dataLine, derr := reader.ReadString('\n')
		if derr != nil {
			return connClose
		}
		if strings.TrimRight(dataLine, "\r\n") == "." {
			break
		}
	}
	*msgOnConn++
	s.msgCount.Add(1)
	write("250 OK")
	return connContinue
}

func (s *fakeSMTPServer) handleReset(write func(string)) connAction {
	s.mu.Lock()
	fail := s.failRSET
	s.mu.Unlock()
	if fail {
		write("421 Service not available")
		return connClose
	}
	write("250 OK")
	return connContinue
}

func (s *fakeSMTPServer) close() {
	s.closed.Store(true)
	s.listener.Close() //nolint:errcheck
}

func (s *fakeSMTPServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// helper to create a pool pointing at the fake server
func newTestPool(t *testing.T, srv *fakeSMTPServer, poolSize int) *smtpPool {
	t.Helper()
	cfg := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Timeout:  5 * time.Second,
		PoolSize: poolSize,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	t.Cleanup(func() { p.close() })
	return p
}

func TestPoolBasicReuse(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	p := newTestPool(t, srv, 2)

	ctx := context.Background()

	// Get a connection
	pc1, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// Return it
	p.put(pc1)

	// Get again — should be the same connection (LIFO reuse)
	pc2, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if pc1 != pc2 {
		t.Error("expected same connection to be reused")
	}
	p.put(pc2)
}

func TestPoolConnectionReuse(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Timeout:  5 * time.Second,
		PoolSize: 2,
	}
	sender, err := NewSMTPSender(cfg)
	if err != nil {
		t.Fatalf("NewSMTPSender: %v", err)
	}
	defer sender.Close() //nolint:errcheck

	ctx := context.Background()

	// Send two emails
	for i := range 2 {
		e := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: fmt.Sprintf("Test %d", i),
			Body:    "Hello",
			Headers: make(map[string]string),
		}
		if sendErr := sender.Send(ctx, e); sendErr != nil {
			t.Fatalf("send %d: %v", i, sendErr)
		}
	}

	// Should have used only 1 TCP connection
	if n := srv.connCount.Load(); n != 1 {
		t.Errorf("expected 1 TCP connection, got %d", n)
	}
	if n := srv.msgCount.Load(); n != 2 {
		t.Errorf("expected 2 messages, got %d", n)
	}
}

func TestPoolMaxConnections(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	poolSize := 3
	p := newTestPool(t, srv, poolSize)
	ctx := context.Background()

	// Check out poolSize connections
	conns := make([]*pooledConn, poolSize)
	for i := range poolSize {
		pc, err := p.get(ctx)
		if err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
		conns[i] = pc
	}

	// Pool should be exhausted — verify numOpen
	p.mu.Lock()
	numOpen := p.numOpen
	p.mu.Unlock()
	if numOpen != poolSize {
		t.Errorf("expected numOpen=%d, got %d", poolSize, numOpen)
	}

	// Return them
	for _, pc := range conns {
		p.put(pc)
	}
}

func TestPoolWaitTimeout(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        1,
		PoolWaitTimeout: 50 * time.Millisecond,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	ctx := context.Background()

	// Check out the only connection
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	// Second get should timeout
	_, err = p.get(ctx)
	if !errors.Is(err, ErrPoolTimeout) {
		t.Errorf("expected ErrPoolTimeout, got %v", err)
	}

	p.put(pc)
}

func TestPoolContextCancellation(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        1,
		PoolWaitTimeout: 5 * time.Second,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	ctx := context.Background()

	// Exhaust the pool
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	// Cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = p.get(cancelCtx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	p.put(pc)
}

func TestPoolHealthCheckFailure(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	p := newTestPool(t, srv, 2)
	ctx := context.Background()

	// Get and return a connection
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	p.put(pc)

	// Now make RSET fail
	srv.mu.Lock()
	srv.failRSET = true
	srv.mu.Unlock()

	connsBefore := srv.connCount.Load()

	// Get should discard the stale conn and dial a fresh one.
	// But the fresh one will also fail RSET on health check if we don't
	// restore RSET. Restore it so the fresh dial's first use works.
	go func() {
		time.Sleep(20 * time.Millisecond)
		srv.mu.Lock()
		srv.failRSET = false
		srv.mu.Unlock()
	}()

	pc2, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get after RSET failure: %v", err)
	}

	// Should have dialed at least one new connection
	if srv.connCount.Load() <= connsBefore {
		t.Error("expected a new connection to be dialed after health check failure")
	}

	p.put(pc2)
}

func TestPoolMaxMessages(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:        "127.0.0.1",
		Port:        srv.port(),
		Timeout:     5 * time.Second,
		PoolSize:    2,
		MaxMessages: 2,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	ctx := context.Background()

	// Get, increment message count to the limit, return
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	pc.msgCount = 2 // at the limit
	p.put(pc)

	connsBefore := srv.connCount.Load()

	// Next get should discard the expired conn and dial fresh
	pc2, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if srv.connCount.Load() <= connsBefore {
		t.Error("expected new connection after maxMessages exceeded")
	}
	p.put(pc2)
}

func TestPoolMaxLifetime(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        2,
		PoolMaxLifetime: 50 * time.Millisecond,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	ctx := context.Background()

	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	p.put(pc)

	// Wait for the connection to expire
	time.Sleep(60 * time.Millisecond)

	connsBefore := srv.connCount.Load()

	pc2, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if srv.connCount.Load() <= connsBefore {
		t.Error("expected new connection after lifetime expired")
	}
	p.put(pc2)
}

func TestPoolIdleEviction(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        2,
		PoolMaxIdleTime: 30 * time.Millisecond,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	ctx := context.Background()

	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	p.put(pc)

	// Verify there's one idle conn
	p.mu.Lock()
	idleBefore := len(p.idle)
	p.mu.Unlock()
	if idleBefore != 1 {
		t.Fatalf("expected 1 idle conn, got %d", idleBefore)
	}

	// Wait for cleaner to evict (cleaner runs every 100ms minimum interval)
	// After 300ms the 30ms idle limit is well exceeded.
	time.Sleep(300 * time.Millisecond)

	p.mu.Lock()
	idleAfter := len(p.idle)
	p.mu.Unlock()
	if idleAfter != 0 {
		t.Errorf("expected 0 idle conns after eviction, got %d", idleAfter)
	}
}

func TestPoolClose(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Timeout:  5 * time.Second,
		PoolSize: 2,
	}
	p := newSMTPPool(cfg, NoOpLogger{})

	ctx := context.Background()

	// Get and return a connection
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	p.put(pc)

	// Close the pool
	closeErr := p.close()
	if closeErr != nil {
		t.Fatalf("close: %v", closeErr)
	}

	// Subsequent get should fail
	_, err = p.get(ctx)
	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}

	// Double close should be safe
	closeErr = p.close()
	if closeErr != nil {
		t.Errorf("double close: %v", closeErr)
	}
}

func TestPoolCloseWithActive(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Timeout:  5 * time.Second,
		PoolSize: 2,
	}
	p := newSMTPPool(cfg, NoOpLogger{})

	ctx := context.Background()

	// Check out a connection
	pc, err := p.get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	// Close the pool while connection is checked out
	if err := p.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Returning the connection should clean it up without panic
	p.put(pc)

	p.mu.Lock()
	numOpen := p.numOpen
	p.mu.Unlock()
	if numOpen != 0 {
		t.Errorf("expected numOpen=0 after put on closed pool, got %d", numOpen)
	}
}

func TestPoolConcurrentBatch(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	poolSize := 5
	numEmails := 20

	cfg := SMTPConfig{
		Host:         "127.0.0.1",
		Port:         srv.port(),
		Timeout:      5 * time.Second,
		PoolSize:     poolSize,
		MaxIdleConns: poolSize, // match pool size to prevent idle eviction under -race
		RateLimit:    -1,       // disable rate limiter
	}
	sender, err := NewSMTPSender(cfg)
	if err != nil {
		t.Fatalf("NewSMTPSender: %v", err)
	}
	defer sender.Close() //nolint:errcheck

	ctx := context.Background()
	var wg sync.WaitGroup
	errCh := make(chan error, numEmails)

	for i := range numEmails {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := &Email{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: fmt.Sprintf("Batch %d", i),
				Body:    "Hello",
				Headers: make(map[string]string),
			}
			if sendErr := sender.Send(ctx, e); sendErr != nil {
				errCh <- sendErr
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for sendErr := range errCh {
		t.Errorf("send error: %v", sendErr)
	}

	if n := srv.msgCount.Load(); n != int64(numEmails) {
		t.Errorf("expected %d messages, got %d", numEmails, n)
	}

	// Should not exceed pool size connections
	if n := srv.connCount.Load(); n > int64(poolSize) {
		t.Errorf("expected at most %d connections, got %d", poolSize, n)
	}
}

func TestPoolDialFailure(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        2,
		PoolWaitTimeout: 100 * time.Millisecond,
	}
	p := newSMTPPool(cfg, NoOpLogger{})
	defer p.close()

	// Override dial to fail
	dialErr := errors.New("simulated dial failure")
	p.dialFn = func(_ context.Context) (*pooledConn, error) {
		return nil, dialErr
	}

	ctx := context.Background()
	_, err := p.get(ctx)
	if !errors.Is(err, dialErr) {
		t.Errorf("expected dial error, got %v", err)
	}

	// numOpen should be back to 0
	p.mu.Lock()
	numOpen := p.numOpen
	p.mu.Unlock()
	if numOpen != 0 {
		t.Errorf("expected numOpen=0 after dial failure, got %d", numOpen)
	}
}

func TestPoolNoPoolBackwardCompat(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Timeout:  5 * time.Second,
		PoolSize: 0, // pooling disabled
	}
	sender, err := NewSMTPSender(cfg)
	if err != nil {
		t.Fatalf("NewSMTPSender: %v", err)
	}
	defer sender.Close() //nolint:errcheck

	if sender.pool != nil {
		t.Error("expected pool to be nil when PoolSize=0")
	}

	ctx := context.Background()
	// Send two emails — should use separate connections (no reuse)
	for i := range 2 {
		e := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: fmt.Sprintf("Test %d", i),
			Body:    "Hello",
			Headers: make(map[string]string),
		}
		if sendErr := sender.Send(ctx, e); sendErr != nil {
			t.Fatalf("send %d: %v", i, sendErr)
		}
	}

	if n := srv.connCount.Load(); n != 2 {
		t.Errorf("expected 2 separate connections (no pool), got %d", n)
	}
}

func TestPoolCleanerStops(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.close()

	cfg := SMTPConfig{
		Host:            "127.0.0.1",
		Port:            srv.port(),
		Timeout:         5 * time.Second,
		PoolSize:        2,
		PoolMaxIdleTime: 100 * time.Millisecond,
	}
	p := newSMTPPool(cfg, NoOpLogger{})

	// Close the pool and verify cleaner channel is closed
	p.close()

	// cleanerCh should be closed — reading should not block
	select {
	case <-p.cleanerCh:
		// ok — channel is closed
	case <-time.After(time.Second):
		t.Error("cleanerCh should be closed after pool.close()")
	}
}

func TestPoolConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  SMTPConfig
		wantErr bool
	}{
		{
			name: "negative pool size",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				PoolSize: -1,
			},
			wantErr: true,
		},
		{
			name: "negative max idle",
			config: SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				PoolSize:     5,
				MaxIdleConns: -1,
			},
			wantErr: true,
		},
		{
			name: "max idle exceeds pool size",
			config: SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				PoolSize:     2,
				MaxIdleConns: 5,
			},
			wantErr: true,
		},
		{
			name: "negative max messages",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				PoolSize:    5,
				MaxMessages: -1,
			},
			wantErr: true,
		},
		{
			name: "valid pool config",
			config: SMTPConfig{
				Host:         "smtp.example.com",
				Port:         587,
				PoolSize:     5,
				MaxIdleConns: 3,
				MaxMessages:  50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
