package apiv3

import (
	"bytes"
	"cmp"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// step is one scripted response. drop closes the connection without
// answering; truncate sends headers promising more body than it delivers.
type step struct {
	status   int
	header   map[string]string
	body     string
	drop     bool
	truncate bool
}

// script answers each request with the next step, repeating the last.
type script struct {
	mu     sync.Mutex
	steps  []step
	calls  int
	bodies [][]byte
}

func (s *script) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func newScripted(t *testing.T, p RetryPolicy, steps ...step) (*Client, *script) {
	t.Helper()
	s := &script{steps: steps}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		st := s.steps[min(s.calls, len(s.steps)-1)]
		s.calls++
		s.bodies = append(s.bodies, raw)
		s.mu.Unlock()

		if st.drop || st.truncate {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			if st.truncate {
				code := cmp.Or(st.status, http.StatusOK)
				_, _ = fmt.Fprintf(buf, "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: 100\r\n", code, http.StatusText(code))
				for k, v := range st.header {
					_, _ = fmt.Fprintf(buf, "%s: %s\r\n", k, v)
				}
				_, _ = buf.WriteString("\r\n{\"data\":")
				_ = buf.Flush()
			}
			_ = conn.Close()
			return
		}
		for k, v := range st.header {
			w.Header().Set(k, v)
		}
		if st.body == "" {
			w.WriteHeader(st.status)
			return
		}
		writeJSON(w, st.status, st.body)
	}))
	t.Cleanup(srv.Close)
	return NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").WithRetry(p), s
}

const projectBody = `{"data":{"id":"p1","account_id":"a1","name":"App","active":true}}`

var now = map[string]string{"Retry-After": "0"}

func TestRetryIsOffByDefault(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{},
		step{status: http.StatusServiceUnavailable, header: now},
		step{status: http.StatusOK, body: projectBody})

	if _, err := c.Projects.Get(context.Background(), "p1"); err == nil {
		t.Fatal("Get succeeded; the 503 should have been returned")
	}
	if s.count() != 1 {
		t.Errorf("calls = %d, want 1", s.count())
	}
}

// The rules, case by case, without a server.
func TestRetryableByMethodAndOutcome(t *testing.T) {
	status := func(code int) error { return &Error{StatusCode: code} }
	bodyCut := func(cause error) error { return &Error{StatusCode: 200, cause: cause} }
	wire := func(err error) error { return &url.Error{Op: "Get", URL: "http://x", Err: err} }
	all := []string{http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodPost}

	tests := []struct {
		name  string
		err   error
		retry []string // the methods that retry; the rest don't
	}{
		{"429", status(429), all},
		{"503", status(503), all[:4]},
		{"502", status(502), all[:4]},
		{"504", status(504), all[:4]},
		{"500", status(500), nil},
		{"404", status(404), nil},
		{"422", status(422), nil},
		{"429 with its body cut off", &Error{StatusCode: 429, cause: io.ErrUnexpectedEOF}, all},
		{"connection reset", wire(&net.OpError{Op: "read", Err: syscall.ECONNRESET}), all[:4]},
		{"broken pipe", wire(&net.OpError{Op: "write", Err: syscall.EPIPE}), all[:4]},
		// The shapes net/http returns on a reused connection, and over HTTP/2.
		{"idle connection closed", wire(errors.New("http: server closed idle connection")), all[:4]},
		{"HTTP/2 GOAWAY", wire(errors.New("http2: server sent GOAWAY and closed the connection")), all[:4]},
		{"unrecognised failure", wire(errors.New("something new went wrong")), all[:4]},
		{"EOF", wire(io.EOF), all[:4]},
		{"attempt deadline", wire(context.DeadlineExceeded), all[:4]},
		{"DNS timeout", wire(&net.DNSError{Err: "i/o timeout", Name: "x", IsTimeout: true}), all[:4]},
		{"body cut off", bodyCut(io.ErrUnexpectedEOF), all[:4]},
		{"body cut off by an HTTP/2 stream error", bodyCut(errors.New("stream error: stream ID 3; INTERNAL_ERROR")), all[:4]},
		{"body over the cap", bodyCut(fmt.Errorf("response body exceeds 1 bytes: %w", errBodyTooLarge)), nil},
		{"connection refused", wire(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}), nil},
		{"host not found", wire(&net.DNSError{Err: "no such host", Name: "x", IsNotFound: true}), nil},
		{"untrusted certificate", wire(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), nil},
		{"wrong hostname", wire(x509.HostnameError{Host: "x"}), nil},
		{"request never built", errors.New("parse \"::\": missing protocol scheme"), nil},
	}
	for _, tt := range tests {
		for _, m := range all {
			want := false
			for _, r := range tt.retry {
				want = want || r == m
			}
			if got := retryable(context.Background(), m, tt.err); got != want {
				t.Errorf("%s %s: retryable = %v, want %v", tt.name, m, got, want)
			}
		}
	}

	// The caller's own cancellation is never retried, even after a 429.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if retryable(ctx, http.MethodGet, status(429)) {
		t.Error("retried after the caller canceled")
	}
}

// A 429 ran nothing, so even a create is sent again, with the same body.
func TestRetryResendsAPostAfterThrottling(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusTooManyRequests, header: now},
		step{status: http.StatusCreated, body: projectBody})

	if _, err := c.Projects.Create(context.Background(), ProjectCreateParams{Name: "App"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.count() != 2 {
		t.Fatalf("calls = %d, want 2", s.count())
	}
	if !bytes.Equal(s.bodies[0], s.bodies[1]) || len(s.bodies[0]) == 0 {
		t.Errorf("bodies differ: %q then %q", s.bodies[0], s.bodies[1])
	}
}

func TestRetryNeverResendsAPostAfterA503(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusServiceUnavailable, header: now},
		step{status: http.StatusCreated, body: projectBody})

	if _, err := c.Projects.Create(context.Background(), ProjectCreateParams{Name: "App"}); err == nil {
		t.Fatal("Create succeeded; the 503 should have been returned")
	}
	if s.count() != 1 {
		t.Errorf("calls = %d, want 1", s.count())
	}
}

func TestRetryGivesUpAfterMaxAttempts(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusServiceUnavailable, header: now})

	_, err := c.Projects.Get(context.Background(), "p1")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("err = %v, want the last 503", err)
	}
	if wait, ok := apiErr.RetryAfter(); !ok || wait != 0 {
		t.Errorf("RetryAfter = %v, %v; want 0, true", wait, ok)
	}
	if s.count() != 3 {
		t.Errorf("calls = %d, want 3", s.count())
	}
}

func TestRetryResendsAPatch(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusServiceUnavailable, header: now},
		step{status: http.StatusOK, body: projectBody})

	name := "App"
	if _, err := c.Projects.Update(context.Background(), "p1", ProjectParams{Name: &name}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.count() != 2 || !bytes.Equal(s.bodies[0], s.bodies[1]) {
		t.Errorf("calls = %d, bodies %q then %q; want the same body twice", s.count(), s.bodies[0], s.bodies[1])
	}
}

// A Retry-After beyond MaxWait returns at once, including one too large for a
// time.Duration.
func TestRetryReturnsAtOnceWhenRetryAfterExceedsMaxWait(t *testing.T) {
	for _, after := range []string{"120", "99999999999999", time.Now().Add(10 * time.Minute).UTC().Format(http.TimeFormat)} {
		c, s := newScripted(t, RetryPolicy{MaxAttempts: 3, MaxWait: 5 * time.Second},
			step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": after}})

		if _, err := c.Projects.Get(context.Background(), "p1"); err == nil {
			t.Fatalf("Retry-After %q: Get succeeded", after)
		}
		// Three attempts allowed, one made: the wait was refused, not shortened.
		if s.count() != 1 {
			t.Errorf("Retry-After %q: calls = %d, want 1", after, s.count())
		}
	}
}

func TestRetryBacksOffWithoutRetryAfter(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 2},
		step{status: http.StatusServiceUnavailable},
		step{status: http.StatusOK, body: projectBody})

	start := time.Now()
	if _, err := c.Projects.Get(context.Background(), "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if elapsed := time.Since(start); elapsed < initialBackoff*3/4 {
		t.Errorf("retried after %v, want at least %v less jitter", elapsed, initialBackoff)
	}
	if s.count() != 2 {
		t.Errorf("calls = %d, want 2", s.count())
	}
}

func TestRetryStopsWaitingWhenTheContextEnds(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": "30"}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Projects.Get(ctx, "p1")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("err = %v, want the 429", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %v; waiting should stop with the context, not sit out Retry-After: 30", elapsed)
	}
	if s.count() != 1 {
		t.Errorf("calls = %d, want 1", s.count())
	}
}

func TestRetryResendsAGetAfterADroppedConnection(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 2},
		step{drop: true},
		step{status: http.StatusOK, body: projectBody})

	start := time.Now()
	if _, err := c.Projects.Get(context.Background(), "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.count() != 2 {
		t.Errorf("calls = %d, want 2", s.count())
	}
	if time.Since(start) < initialBackoff*3/4 {
		t.Error("retried without backing off")
	}
}

func TestRetryResendsAGetAfterACutOffBody(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 2},
		step{truncate: true},
		step{status: http.StatusOK, body: projectBody})

	if _, err := c.Projects.Get(context.Background(), "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.count() != 2 {
		t.Errorf("calls = %d, want 2", s.count())
	}
}

// The first attempt may have deleted it, so the retry's 404 is success.
func TestRetryTreatsADeletesLater404AsDone(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{status: http.StatusServiceUnavailable, header: now},
		step{status: http.StatusNotFound, body: `{"error":{"code":"not_found","message":"Not found"}}`})

	if err := c.Projects.Delete(context.Background(), "p1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.count() != 2 {
		t.Errorf("calls = %d, want 2", s.count())
	}
}

// A 429 ran nothing, and a first-attempt 404 is just a 404.
func TestRetryKeepsADeletes404WhenNothingRan(t *testing.T) {
	notFound := step{status: http.StatusNotFound, body: `{"error":{"code":"not_found","message":"Not found"}}`}
	for name, steps := range map[string][]step{
		"first attempt": {notFound},
		"after a 429":   {{status: http.StatusTooManyRequests, header: now}, notFound},
	} {
		c, _ := newScripted(t, RetryPolicy{MaxAttempts: 3}, steps...)
		if err := c.Projects.Delete(context.Background(), "p1"); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestRetryDoesNotResendAfterAPermanentFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.Listener.Addr().String()
	srv.Close() // nothing listens there now: connection refused

	var dials atomic.Int32
	transport := &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		dials.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}}
	c := NewClient().WithBaseURL("http://" + addr).WithBearerToken("hbt_x").
		WithHTTPClient(&http.Client{Transport: transport}).WithRetry(RetryPolicy{MaxAttempts: 3})
	if _, err := c.Projects.Get(context.Background(), "p1"); err == nil {
		t.Fatal("Get succeeded against a closed port")
	}
	if n := dials.Load(); n != 1 {
		t.Errorf("dialed %d times, want 1: a refused connection is not retried", n)
	}
}

// A cut-off response keeps its Retry-After: one beyond MaxWait is refused rather
// than replaced by the backoff schedule.
func TestRetryKeepsRetryAfterFromACutOffResponse(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3, MaxWait: 5 * time.Second},
		step{truncate: true, status: http.StatusServiceUnavailable, header: map[string]string{"Retry-After": "120"}})

	if _, err := c.Projects.Get(context.Background(), "p1"); err == nil {
		t.Fatal("Get succeeded")
	}
	if s.count() != 1 {
		t.Errorf("calls = %d, want 1", s.count())
	}
}

// A 429 is a 429 even if its body is cut off, so a create is still resent.
func TestRetryResendsAPostAfterACutOff429(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3},
		step{truncate: true, status: http.StatusTooManyRequests, header: now},
		step{status: http.StatusCreated, body: projectBody})

	if _, err := c.Projects.Create(context.Background(), ProjectCreateParams{Name: "App"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.count() != 2 {
		t.Errorf("calls = %d, want 2", s.count())
	}
}

// The client's own backoff is capped at MaxWait rather than abandoned, so a
// short MaxWait still allows every attempt.
func TestRetryCapsItsBackoffAtMaxWait(t *testing.T) {
	c, s := newScripted(t, RetryPolicy{MaxAttempts: 3, MaxWait: 50 * time.Millisecond},
		step{status: http.StatusServiceUnavailable},
		step{status: http.StatusServiceUnavailable},
		step{status: http.StatusOK, body: projectBody})

	if _, err := c.Projects.Get(context.Background(), "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.count() != 3 {
		t.Errorf("calls = %d, want 3", s.count())
	}
}

func TestJitterStaysWithinAQuarter(t *testing.T) {
	for range 1000 {
		if got := jitter(time.Second); got < 750*time.Millisecond || got >= 1250*time.Millisecond {
			t.Fatalf("jitter(1s) = %v, want within [750ms, 1250ms)", got)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
		ok     bool
	}{
		{"", 0, false},
		{"0", 0, true},
		{"30", 30 * time.Second, true},
		{"-1", 0, false},
		{"soon", 0, false},
		{"99999999999999", time.Duration(math.MaxInt64), true},
		{time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0, true},
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(http.Header{"Retry-After": {tt.header}})
		if got != tt.want || ok != tt.ok {
			t.Errorf("Retry-After %q = %v, %v; want %v, %v", tt.header, got, ok, tt.want, tt.ok)
		}
	}
}

// deadlineRecorder answers every request and notes the deadline its context
// carried, which is the attempt's own.
type deadlineRecorder struct {
	mu        sync.Mutex
	deadlines []time.Duration
}

func (d *deadlineRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	if !ok {
		return nil, fmt.Errorf("%s %s carried no deadline", req.Method, req.URL.Path)
	}
	d.mu.Lock()
	d.deadlines = append(d.deadlines, time.Until(deadline))
	d.mu.Unlock()
	status, body := http.StatusOK, projectBody
	if req.Method == http.MethodDelete {
		status, body = http.StatusNoContent, ""
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestEachAttemptHasItsOwnDeadline(t *testing.T) {
	rec := &deadlineRecorder{}
	c := NewClient().WithBaseURL("https://api.example.com").WithBearerToken("hbt_x").
		WithHTTPClient(&http.Client{Transport: rec})

	if _, err := c.Projects.Get(context.Background(), "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := c.Projects.Delete(context.Background(), "p1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	short, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := c.Projects.Get(short, "p1"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := []time.Duration{attemptTimeout, deleteAttemptTimeout, 2 * time.Second}
	for i, w := range want {
		if got := rec.deadlines[i]; got > w || got < w-time.Second {
			t.Errorf("attempt %d deadline in %v, want about %v", i+1, got, w)
		}
	}
}

func TestOperationMethodsCoverTheSpec(t *testing.T) {
	if len(operationMethods) != len(OperationScopes)+1 { // +1: getToken needs no scope
		t.Errorf("%d methods for %d scoped operations plus getToken", len(operationMethods), len(OperationScopes))
	}
	for op := range OperationScopes {
		if _, ok := operationMethods[op]; !ok {
			t.Errorf("%s has a scope but no method", op)
		}
	}
	for op, want := range map[string]string{
		"getToken": "GET", "getProject": "GET", "createAlarm": "POST",
		"updateCheckIn": "PATCH", "replaceCheckIns": "PUT", "deleteProject": "DELETE",
	} {
		if got := operationMethods[op]; got != want {
			t.Errorf("%s = %q, want %q", op, got, want)
		}
	}
}

// Every request helper names an operation that exists, and the one its closure
// actually calls, so the retry rules apply the right method.
func TestCallSitesNameTheOperationTheyCall(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	helpers := map[string]int{"getOne": 2, "listOffset": 2, "listTimeSeries": 2, "noContent": 2, "run": 1}
	fset := token.NewFileSet()
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			idx, isHelper := helpers[calleeName(call.Fun)]
			if !isHelper {
				return true
			}
			pos := fset.Position(call.Pos())
			if len(call.Args) != idx+2 {
				t.Errorf("%s: %s takes the operation and a closure; this call can't be checked", pos, calleeName(call.Fun))
				return true
			}
			// A helper handing its own parameters on (getOne → run) names nothing.
			opArg, opIsParam := call.Args[idx].(*ast.Ident)
			closureArg, closureIsParam := call.Args[idx+1].(*ast.Ident)
			if opIsParam && closureIsParam && opArg.Name == "opID" && closureArg.Name == "op" {
				return true
			}
			if _, isClosure := call.Args[idx+1].(*ast.FuncLit); !isClosure {
				t.Errorf("%s: pass the operation as a closure literal so this test can check it", pos)
				return true
			}
			if ident, ok := call.Args[idx].(*ast.Ident); ok && ident.Name == "opFollowLink" {
				checked++
				return true
			}
			lit, ok := call.Args[idx].(*ast.BasicLit)
			if !ok {
				t.Errorf("%s: operation is not a literal", pos)
				return true
			}
			opID := strings.Trim(lit.Value, `"`)
			if _, ok := operationMethods[opID]; !ok {
				t.Errorf("%s: %q is not an operation in the spec", pos, opID)
			}
			if called := generatedCall(call.Args[idx+1].(*ast.FuncLit)); called != strings.ToUpper(opID[:1])+opID[1:] {
				t.Errorf("%s: names %q but calls gen().%s", pos, opID, called)
			}
			checked++
			return true
		})
	}
	if checked < 50 {
		t.Errorf("checked %d call sites; the scan is missing them", checked)
	}
}

func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.IndexExpr:
		return calleeName(f.X)
	case *ast.IndexListExpr:
		return calleeName(f.X)
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// generatedCall returns the method a closure calls on gen().
func generatedCall(fn *ast.FuncLit) string {
	name := ""
	ast.Inspect(fn, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			if g, ok := inner.Fun.(*ast.SelectorExpr); ok && g.Sel.Name == "gen" {
				name = sel.Sel.Name
			}
		}
		return true
	})
	return name
}
