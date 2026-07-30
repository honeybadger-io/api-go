// Package apiv3 is a client for the Honeybadger v3 API.
//
// It wraps generated code (internal/gen) with a hand-written surface: auth,
// pagination, typed errors, and account handling. The generated layer is
// private on purpose — its shape follows the OpenAPI bundle and is not a stable
// API.
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

// RateLimit is a snapshot of the rate-limit headers from the most recent
// response. v3 allows 360 requests per hour.
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

// AccountMe is the account_id sentinel v3 resolves from the credential. It is
// the default, which is why account ids do not appear in method signatures.
//
// It only resolves when the credential covers exactly one account. A credential
// covering several returns 422 with code ambiguous_account; recover by listing
// accounts and passing a concrete id through the AccountID option or
// WithAccountID.
const AccountMe = "me"

// Client is a Honeybadger v3 API client. Construct it with NewClient and
// configure it with the With* methods, which return the same client for
// chaining.
type Client struct {
	baseURL       string
	bearerToken   string
	httpClient    *http.Client
	requestID     RequestIDHook
	defaultAcctID string

	mu        sync.RWMutex
	rateLimit *RateLimit

	// Projects handles the projects resource.
	Projects *ProjectsService
}

// NewClient returns a client pointing at the production API with a 30 second
// timeout, resolving the account from the credential.
func NewClient() *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	c.Projects = &ProjectsService{client: c}
	return c
}

// WithAccountID sets the account every request uses, replacing the `me`
// sentinel. Needed only for a credential covering more than one account; a
// per-call AccountID option still takes precedence.
func (c *Client) WithAccountID(id string) *Client {
	c.defaultAcctID = id
	return c
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

// WithBaseURL sets the API host. The version segment is optional: pass
// "https://app.honeybadger.io" or "https://app.honeybadger.io/v3" and the
// result is the same. Omitting it matches the v2 client and the MCP server's
// HONEYBADGER_API_URL.
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithBearerToken sets the credential, sent as `Authorization: Bearer <token>`.
//
// Accepts a scoped API token (`hbt_` or `hba_`) or an OAuth access token. There
// is deliberately no Basic-auth equivalent: v3's documented challenge is
// `WWW-Authenticate: Bearer`, and a single credential path keeps authorization
// decisions in one place.
func (c *Client) WithBearerToken(token string) *Client {
	c.bearerToken = token
	return c
}

// WithHTTPClient replaces the underlying HTTP client, for callers that need
// their own transport, timeout, or instrumentation.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	c.httpClient = hc
	return c
}

// WithRequestIDHook registers a callback for the request_id in response bodies.
// Useful for logging an identifier that Honeybadger support can correlate.
func (c *Client) WithRequestIDHook(hook RequestIDHook) *Client {
	c.requestID = hook
	return c
}

// LastRateLimit returns a snapshot of the rate-limit headers from the most
// recent response, or nil if no response has carried them yet.
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
// It is constructed per call rather than cached because With* methods may run
// in any order, and a cached client would capture a stale base URL or token.
// Construction only assembles structs; it performs no I/O.
func (c *Client) gen() *gen.ClientWithResponses {
	client, err := gen.NewClientWithResponses(
		c.serverURL(),
		gen.WithHTTPClient(&observer{client: c}),
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

// observer wraps the HTTP client so responses can be inspected for rate-limit
// headers and request ids. The generated code only exposes a request editor
// hook, which cannot see responses.
type observer struct {
	client *Client
}

func (o *observer) Do(req *http.Request) (*http.Response, error) {
	resp, err := o.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	o.client.recordRateLimit(resp)
	o.client.reportRequestID(req.Context(), resp)
	return resp, nil
}

func (c *Client) recordRateLimit(resp *http.Response) {
	limit, okLimit := atoiHeader(resp, "X-RateLimit-Limit")
	remaining, okRemaining := atoiHeader(resp, "X-RateLimit-Remaining")
	reset, okReset := atoiHeader(resp, "X-RateLimit-Reset")
	if !okLimit && !okRemaining && !okReset {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.rateLimit = &RateLimit{
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Unix(int64(reset), 0),
	}
}

// reportRequestID reads request_id out of the response body and hands it to the
// hook. The body is fully buffered and replaced, so the generated decoder still
// sees it — generated code reads the body itself, and consuming it here without
// restoring it would break every response.
func (c *Client) reportRequestID(ctx context.Context, resp *http.Response) {
	if c.requestID == nil || resp.Body == nil {
		return
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(strings.NewReader(string(body)))
	if err != nil || len(body) == 0 {
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
	c.requestID(ctx, resp.StatusCode, envelope.Meta.RequestID)
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
