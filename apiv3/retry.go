package apiv3

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// RetryPolicy decides how many times a request is attempted and how long the
// client waits between attempts. Retries are off until a client opts in with
// WithRetry.
//
// v3 has no idempotency key, but it makes every GET, PATCH, PUT and DELETE
// converge when sent twice. A POST may create twice. So the method decides:
//
//   - 429: every method is retried. The rate limit is checked before the
//     request's action runs, so nothing happened.
//   - 503, 502, 504, or a transport failure whose outcome is unknown (a
//     timeout, a dropped or reset connection, an HTTP/2 GOAWAY, a response
//     cut off part way): GET, PATCH, PUT and DELETE are retried. POST never
//     is: a create can store its record and still fail to answer.
//   - A retried DELETE that answers 404 succeeds, when an earlier attempt may
//     have gone through: that attempt deleted it. The flip side is that a
//     delete of an id that never existed reports success after, say, a 503
//     and then a 404, so under WithRetry ErrNotFound from a delete isn't
//     reliable. After a 429, which ran nothing, a 404 is returned as it is.
//     A DELETE that returns the resource (Faults.Unassign) has nothing to
//     return, so it reports the 404.
//   - Anything else is returned at once: other statuses; failures that would
//     only repeat (a host that doesn't resolve, a refused connection, a TLS
//     certificate the client won't trust, a body over the size cap, a request
//     that couldn't be built); and the caller's own cancellation or deadline.
//     A TLS handshake that times out is a timeout, and is retried.
//
// Between attempts the client waits as long as Retry-After asks, in seconds or
// as an HTTP date. A Retry-After longer than MaxWait isn't waited out: the error
// is returned straight away. Without one it backs off from 1s, doubling, with
// up to a quarter either way of jitter so clients sharing a credential don't
// retry in step, and never waiting longer than MaxWait. Waiting stops when the
// caller's context ends, and the last attempt's error is returned.
//
// MaxAttempts counts this package's attempts. net/http resends a request
// itself when a reused connection closed before any of it was written, and a
// GET when it closed later, so the number of requests on the wire can be
// higher.
type RetryPolicy struct {
	// MaxAttempts is the most attempts made, the first included. Zero or one
	// means no retries.
	MaxAttempts int

	// MaxWait is the longest single wait between attempts. Zero means 60
	// seconds. The total time spent is bounded by the caller's context.
	MaxWait time.Duration
}

const (
	defaultMaxWait = 60 * time.Second
	initialBackoff = time.Second

	// Each attempt gets its own deadline, inside the caller's context. A delete
	// gets longer: deleting a project removes its data before it answers. So
	// does an Insights query: the server waits up to 60 seconds for one.
	attemptTimeout       = 30 * time.Second
	deleteAttemptTimeout = 5 * time.Minute
	queryAttemptTimeout  = 65 * time.Second
)

// WithRetry returns a client that retries failed requests under p. See
// RetryPolicy for which failures are retried.
func (c *Client) WithRetry(p RetryPolicy) *Client {
	next := c.clone()
	next.retry = p
	return next
}

// operation is a generated call, ready to run with an attempt's context.
type operation func(ctx context.Context) (*http.Response, error)

// opFollowLink names a request to a pagination link from a previous response.
// Those links are always GETs and have no operationId of their own.
const opFollowLink = "followLink"

// opRunInsightsQuery is the Insights query, which gets a longer deadline.
const opRunInsightsQuery = "runInsightsQuery"

// methodOf returns an operation's HTTP method.
func methodOf(opID string) (string, error) {
	if opID == opFollowLink {
		return http.MethodGet, nil
	}
	method, ok := operationMethods[opID]
	if !ok {
		return "", fmt.Errorf("apiv3: operation %q is not in the spec", opID)
	}
	return method, nil
}

// run makes an operation's attempts under the client's retry policy and returns
// the last one's result.
func (c *Client) run(ctx context.Context, opID string, op operation) (int, []byte, error) {
	method, err := methodOf(opID)
	if err != nil {
		return 0, nil, err
	}

	maxWait := c.retry.MaxWait
	if maxWait <= 0 {
		maxWait = defaultMaxWait
	}
	backoff := initialBackoff
	mayHaveLanded := false // an earlier attempt may have been carried out

	for attempt := 1; ; attempt++ {
		status, body, err := c.attempt(ctx, opID, method, op)
		if mayHaveLanded && method == http.MethodDelete && status == http.StatusNotFound {
			return status, body, &goneAfterRetry{err}
		}
		if err == nil || attempt >= c.retry.MaxAttempts || !retryable(ctx, method, err) {
			return status, body, err
		}
		if !throttled(err) {
			mayHaveLanded = true
		}

		wait, asked := retryAfterOf(err)
		if asked && wait > maxWait {
			return status, body, err
		}
		if !asked {
			wait = min(jitter(backoff), maxWait)
			if backoff < maxWait {
				backoff *= 2
			}
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return status, body, err
		case <-timer.C:
		}
	}
}

// attempt makes one request under its own deadline. The response body is read
// inside it, so the deadline covers the whole exchange.
func (c *Client) attempt(ctx context.Context, opID, method string, op operation) (int, []byte, error) {
	timeout := attemptTimeout
	switch {
	case method == http.MethodDelete:
		timeout = deleteAttemptTimeout
	case opID == opRunInsightsQuery:
		timeout = queryAttemptTimeout
	}
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.do(attemptCtx, op)
}

// goneAfterRetry is a DELETE's 404 after an earlier attempt may have been
// carried out: the earlier attempt probably deleted it. A delete that returns no
// content treats it as done; one that returns the resource can't, so it still
// sees the 404, which errors.Is matches as ErrNotFound.
type goneAfterRetry struct{ err error }

func (e *goneAfterRetry) Error() string { return e.err.Error() }
func (e *goneAfterRetry) Unwrap() error { return e.err }

// retryable reports whether a failed attempt may be made again.
func retryable(ctx context.Context, method string, err error) bool {
	// The caller gave up; their deadline or cancellation is not ours to retry.
	if ctx.Err() != nil {
		return false
	}

	if throttled(err) {
		return true
	}
	if method == http.MethodPost {
		return false
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.cause != nil:
			// The response started but its body didn't finish arriving.
			return transient(apiErr.cause)
		case apiErr.StatusCode == http.StatusServiceUnavailable,
			apiErr.StatusCode == http.StatusBadGateway,
			apiErr.StatusCode == http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	// No response at all. The HTTP client reports every failure to send as a
	// *url.Error; anything else means the request was never built.
	var urlErr *url.Error
	return errors.As(err, &urlErr) && transient(err)
}

// throttled reports whether the API refused the request for its rate limit,
// which it checks before doing anything else. The status settles it, even if
// the body was cut off.
func throttled(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests
}

// transient reports whether a failure to complete a request leaves its outcome
// unknown, so that a second try may succeed. It is a deny-list: a request may
// fail in many ways (a reset, a closed idle connection, a broken pipe, an
// HTTP/2 GOAWAY or stream error, a timeout), and the method has already decided
// whether resending is safe. Only failures that would repeat the same way are
// refused.
func transient(err error) bool {
	if errors.Is(err, errBodyTooLarge) || errors.Is(err, errConnRefused) || errors.Is(err, http.ErrSchemeMismatch) {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	var (
		verifyErr  *tls.CertificateVerificationError
		alertErr   tls.AlertError
		recordErr  tls.RecordHeaderError
		authority  x509.UnknownAuthorityError
		hostname   x509.HostnameError
		invalidErr x509.CertificateInvalidError
	)
	return !errors.As(err, &verifyErr) && !errors.As(err, &alertErr) && !errors.As(err, &recordErr) &&
		!errors.As(err, &authority) && !errors.As(err, &hostname) && !errors.As(err, &invalidErr)
}

// jitter spreads d over [0.75d, 1.25d).
func jitter(d time.Duration) time.Duration {
	return d*3/4 + rand.N(d/2)
}

// retryAfterOf returns the Retry-After of the response that failed, if any.
func retryAfterOf(err error) (time.Duration, bool) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return 0, false
	}
	return apiErr.RetryAfter()
}

// parseRetryAfter reads a Retry-After header, in seconds or as an HTTP date. A
// date already past means no wait.
func parseRetryAfter(h http.Header) (time.Duration, bool) {
	v := h.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(v, 10, 64); err == nil {
		if seconds < 0 {
			return 0, false
		}
		// A wait too long to represent is longer than anyone's MaxWait.
		if seconds > math.MaxInt64/int64(time.Second) {
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(seconds) * time.Second, true
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(time.Until(at), 0), true
	}
	return 0, false
}
