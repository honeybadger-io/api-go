package apiv3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
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
//   - 503, 502, 504, a timeout, or a dropped connection: GET, PATCH, PUT and
//     DELETE are retried. POST never is: a create can store its record and
//     still fail to answer.
//   - A retried DELETE that answers 404 succeeds, when an earlier attempt may
//     have gone through: that attempt deleted it. After a 429, which ran
//     nothing, a 404 is returned as it is.
//   - Anything else is returned at once: other statuses, permanent transport
//     failures (DNS, TLS, a refused connection), and the caller's own
//     cancellation or deadline.
//
// Between attempts the client waits as long as Retry-After asks, in seconds or
// as an HTTP date; without one it waits 1s, then 2s, then 4s and so on. A wait
// longer than MaxWait isn't attempted: the error is returned straight away.
// Waiting stops when the caller's context ends, and the last attempt's error is
// returned.
//
// MaxAttempts counts this package's attempts. net/http may itself resend a
// GET, PUT or DELETE on a reused connection that closed before the request was
// written, so the number of requests on the wire can be higher.
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
	// gets longer: deleting a project removes its data before it answers.
	attemptTimeout       = 30 * time.Second
	deleteAttemptTimeout = 5 * time.Minute
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
		status, body, err := c.attempt(ctx, method, op)
		if mayHaveLanded && method == http.MethodDelete && status == http.StatusNotFound {
			return http.StatusNoContent, nil, nil
		}
		if err == nil || attempt >= c.retry.MaxAttempts || !retryable(ctx, method, err) {
			return status, body, err
		}
		if !throttled(err) {
			mayHaveLanded = true
		}

		wait, asked := retryAfterOf(err)
		if !asked {
			wait = backoff
			backoff *= 2
		}
		if wait > maxWait {
			return status, body, err
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
func (c *Client) attempt(ctx context.Context, method string, op operation) (int, []byte, error) {
	timeout := attemptTimeout
	if method == http.MethodDelete {
		timeout = deleteAttemptTimeout
	}
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.do(attemptCtx, op)
}

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
	// No response at all.
	return transient(err)
}

// throttled reports whether the API refused the request for its rate limit,
// which it checks before doing anything else.
func throttled(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests && apiErr.cause == nil
}

// transient reports whether a transport failure leaves the outcome unknown: the
// request may or may not have reached the server, and a second try may succeed.
// Failures that will happen again the same way (DNS, TLS, a refused
// connection) are not transient.
func transient(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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
	if seconds, err := strconv.Atoi(v); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(time.Until(at), 0), true
	}
	return 0, false
}
