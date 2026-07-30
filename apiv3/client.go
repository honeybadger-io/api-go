// Package apiv3 is a client for the Honeybadger v3 API.
//
// It wraps generated code (internal/gen) with a hand-written surface: auth,
// pagination, typed errors, and account handling.
//
// v3 rejects Honeybadger's older personal auth tokens. The accepted credentials
// are scoped API tokens (`hbt_` personal, `hba_` account) and OAuth access
// tokens, all presented as Bearer. There is no Basic-auth option here by
// design; see WithBearerToken.
//
// For the v2 API, use the root package.
package apiv3

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// DefaultBaseURL is the production v3 endpoint, including the version segment.
const DefaultBaseURL = "https://app.honeybadger.io/v3"

const versionSegment = "/v3"

// maxBodyBytes caps how much of a response body is read. The largest documented
// payloads are notice backtraces and Insights results, none of which approach
// this. A body beyond it is a malfunction, and reading it unbounded would let a
// single response exhaust memory.
const maxBodyBytes = 64 << 20 // 64 MiB

// AccountMe is the account_id sentinel v3 resolves from the credential. It is
// the default, which is why account ids do not appear in method signatures.
//
// It only resolves when the credential covers exactly one account. A credential
// covering several returns 422 with code ambiguous_account; recover by listing
// accounts and passing a concrete id through InAccount or WithAccountID.
const AccountMe = "me"

// RateLimit is a snapshot of the rate-limit headers from a response. v3 allows
// 360 requests per hour.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// RetryAfter reports how long to wait before the limit resets. It is zero once
// the reset time has passed.
func (r RateLimit) RetryAfter() time.Duration {
	d := time.Until(r.Reset)
	if d < 0 {
		return 0
	}
	return d
}

// RequestIDHook observes the request_id v3 returns in a response's meta block.
//
// It takes a context so callers can correlate the id with the work that issued
// the request. It is best-effort: request_id lives in the body rather than a
// header, so it is absent from 204s and from operations whose meta is empty,
// and the hook simply does not fire for those.
type RequestIDHook func(ctx context.Context, status int, requestID string)

// Client is a Honeybadger v3 API client.
//
// A Client is immutable once constructed. The With* methods each return a new
// Client rather than modifying the receiver, so a configured client is safe to
// share across goroutines and cannot have its credential changed underneath an
// in-flight request. Build per-credential clients by chaining from a base:
//
//	base := apiv3.NewClient().WithBaseURL(apiURL)
//	perRequest := base.WithBearerToken(tokenFromRequest)
//
// Both clients above are independent; configuring one never affects the other.
type Client struct {
	baseURL       string
	bearerToken   string
	httpClient    *http.Client
	requestID     RequestIDHook
	defaultAcctID string

	// rateLimit is the only mutable state, and it is observational: it records
	// the most recent response's headers for callers that want to check their
	// budget. Errors carry their own snapshot, taken from the exact response
	// that failed, so this is never used to explain a specific failure.
	mu        sync.RWMutex
	rateLimit *RateLimit

	// Projects handles the projects resource.
	Projects *ProjectsService

	// Faults handles the faults resource and its notices.
	Faults *FaultsService

	// Tokens describes the credential making the request.
	Tokens *TokensService

	// Insights runs BadgerQL queries and lists event streams.
	Insights *InsightsService
}

// NewClient returns a client pointing at the production API with a 30 second
// timeout, resolving the account from the credential.
func NewClient() *Client {
	return (&Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}).rebind()
}

// clone copies the configuration into a fresh Client with its own services and
// its own rate-limit state. Observational state is deliberately not carried
// over: a differently-configured client, often a different credential, has a
// different budget.
func (c *Client) clone() *Client {
	copied := &Client{
		baseURL:       c.baseURL,
		bearerToken:   c.bearerToken,
		httpClient:    c.httpClient,
		requestID:     c.requestID,
		defaultAcctID: c.defaultAcctID,
	}
	return copied.rebind()
}

// rebind points the service structs at this client. Every constructor and
// clone must call it, or a service would keep serving the client it came from.
func (c *Client) rebind() *Client {
	c.Projects = &ProjectsService{client: c}
	c.Faults = &FaultsService{client: c}
	c.Tokens = &TokensService{client: c}
	c.Insights = &InsightsService{client: c}
	return c
}

// WithBaseURL returns a client using the given API host. The version segment is
// optional: "https://app.honeybadger.io" and "https://app.honeybadger.io/v3"
// behave identically. Omitting it matches the v2 client and the MCP server's
// HONEYBADGER_API_URL.
func (c *Client) WithBaseURL(baseURL string) *Client {
	next := c.clone()
	next.baseURL = baseURL
	return next
}

// WithBearerToken returns a client using the given credential, sent as
// `Authorization: Bearer <token>`.
//
// Accepts a scoped API token (`hbt_` or `hba_`) or an OAuth access token. There
// is deliberately no Basic-auth equivalent: v3's documented challenge is
// `WWW-Authenticate: Bearer`, and a single credential path keeps authorization
// decisions in one place.
func (c *Client) WithBearerToken(token string) *Client {
	next := c.clone()
	next.bearerToken = token
	return next
}

// WithHTTPClient returns a client using the given HTTP client, for callers that
// need their own transport, timeout, or instrumentation. A nil argument is
// ignored, keeping the existing client rather than panicking on first use.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	if hc == nil {
		return c
	}
	next := c.clone()
	next.httpClient = hc
	return next
}

// WithRequestIDHook returns a client that reports the request_id from response
// bodies. Useful for logging an identifier Honeybadger support can correlate.
func (c *Client) WithRequestIDHook(hook RequestIDHook) *Client {
	next := c.clone()
	next.requestID = hook
	return next
}

// WithAccountID returns a client that uses the given account for every request,
// replacing the `me` sentinel. Needed only for a credential covering more than
// one account; a per-call InAccount option still takes precedence.
func (c *Client) WithAccountID(id string) *Client {
	next := c.clone()
	next.defaultAcctID = id
	return next
}

// accountID resolves which account a call should use: the per-call value, then
// the client default, then the `me` sentinel.
func (c *Client) accountID(perCall string) string {
	if perCall != "" {
		return perCall
	}
	if c.defaultAcctID != "" {
		return c.defaultAcctID
	}
	return AccountMe
}

// LastRateLimit returns a snapshot of the rate-limit headers from the most
// recent response, or nil if no response has carried them.
//
// This is a budget indicator, not an explanation of any particular call. To
// learn why one request was throttled, read Error.RateLimit, which is taken
// from that request's own response.
func (c *Client) LastRateLimit() *RateLimit {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rateLimit == nil {
		return nil
	}
	snapshot := *c.rateLimit
	return &snapshot
}

// serverURL is the base URL with exactly one version segment.
func (c *Client) serverURL() string {
	base := strings.TrimSuffix(c.baseURL, "/")
	if strings.HasSuffix(base, versionSegment) {
		return base
	}
	return base + versionSegment
}

// gen builds a generated client bound to this client's transport and auth.
//
// Constructed per call rather than cached: construction only assembles structs
// and performs no I/O, and because the Client is immutable, every call sees a
// coherent configuration.
func (c *Client) gen() *gen.ClientWithResponses {
	client, err := gen.NewClientWithResponses(
		c.serverURL(),
		gen.WithHTTPClient(c.httpClient),
		gen.WithRequestEditorFn(c.authorize),
	)
	if err != nil {
		// NewClientWithResponses only fails if a ClientOption fails. Neither of
		// the options above can, so this is unreachable.
		panic("apiv3: constructing generated client: " + err.Error())
	}
	return client
}

// authorize attaches the Bearer credential. A client with no token sends no
// Authorization header, which surfaces as a 401 from the API rather than a
// local error — the same shape as an invalid token, and easier to diagnose from
// a response than from a client-side panic.
func (c *Client) authorize(ctx context.Context, req *http.Request) error {
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	return nil
}

// do runs a generated operation and returns its body.
//
// The facade deliberately calls the raw generated operations rather than their
// *WithResponse wrappers. Those wrappers decode eagerly and, when a body does
// not parse, return a bare json.SyntaxError with the response discarded —
// losing the status, the body, the request id, and this package's typed errors.
// Reading the response here keeps all of it, buffers the body exactly once, and
// takes the rate-limit snapshot from the very response being reported on.
func (c *Client) do(ctx context.Context, op func() (*http.Response, error)) ([]byte, error) {
	resp, err := op()
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	rateLimit := readRateLimit(resp)
	if rateLimit != nil {
		c.mu.Lock()
		c.rateLimit = rateLimit
		c.mu.Unlock()
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if readErr != nil {
		// A truncated body must not be mistaken for a short one. Report the read
		// failure with the status attached, rather than decoding what arrived.
		apiErr := parseError(resp.StatusCode, body)
		apiErr.Message = "reading response body: " + readErr.Error()
		apiErr.RateLimit = rateLimit
		return nil, apiErr
	}

	c.reportRequestID(ctx, resp.StatusCode, body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := parseError(resp.StatusCode, body)
		apiErr.RateLimit = rateLimit
		return nil, apiErr
	}
	return body, nil
}

// reportRequestID hands the body's request_id to the hook, if there is one and
// the body carries one.
func (c *Client) reportRequestID(ctx context.Context, status int, body []byte) {
	if c.requestID == nil || len(body) == 0 {
		return
	}
	var envelope struct {
		Meta struct {
			RequestID string `json:"request_id"`
		} `json:"meta"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Meta.RequestID == "" {
		return
	}
	c.requestID(ctx, status, envelope.Meta.RequestID)
}

func readRateLimit(resp *http.Response) *RateLimit {
	limit, okLimit := atoiHeader(resp, "X-RateLimit-Limit")
	remaining, okRemaining := atoiHeader(resp, "X-RateLimit-Remaining")
	reset, okReset := atoiHeader(resp, "X-RateLimit-Reset")
	if !okLimit && !okRemaining && !okReset {
		return nil
	}
	return &RateLimit{
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Unix(int64(reset), 0),
	}
}

func atoiHeader(resp *http.Response, name string) (int, bool) {
	raw := resp.Header.Get(name)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseBasic reports whether an Authorization header carries Basic credentials.
// Used by tests to assert apiv3 never sends them.
func parseBasic(header string) (user, pass string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return "", "", false
	}
	user, pass, found := strings.Cut(string(decoded), ":")
	return user, pass, found
}
