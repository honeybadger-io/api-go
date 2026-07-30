package apiv3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Stream is an Insights event stream.
//
// Every project has a `default` stream carrying events the application sends and
// an `internal` stream carrying events Honeybadger generates about the project —
// errors, deploys, uptime checks, check-ins.
type Stream = gen.Stream

// InsightsService runs BadgerQL queries and lists the streams they can target.
type InsightsService struct {
	client *Client
}

// InsightsQuery is a BadgerQL query.
type InsightsQuery struct {
	// Query is the BadgerQL query text, required.
	Query string

	// StreamIDs restricts the query to specific streams. Empty searches every
	// stream on the project. Every id must belong to the project — unlike v2,
	// which silently dropped ids that did not, v3 rejects them with a 422, so a
	// partial result can never be mistaken for a complete one.
	StreamIDs []string

	// Ts is the time range, such as "1h".
	Ts string

	// Timezone controls time bucketing and display.
	Timezone string
}

// InsightsResult is the outcome of a query.
//
// Data is deliberately untyped. The spec declines to pin the shape: it varies by
// query — an aggregation returns different fields from an event listing — and is
// passed through from the query service unchanged. Modelling it here would be an
// unverifiable claim about a contract the API does not make.
// Tagged because callers serialise this straight back out — the MCP server
// returns it as tool output — and Go-cased keys would leak into that payload.
type InsightsResult struct {
	Data      map[string]any `json:"data"`
	Meta      map[string]any `json:"meta,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// Query runs a BadgerQL query against the project's event streams.
//
// It is a POST because the query travels in the body, but it only reads.
//
// Selecting streams is part of the request rather than the query. An event class
// that lives on an unselected stream returns no rows, which looks exactly like
// the events not existing — so list the project's streams first when a query
// unexpectedly comes back empty.
func (s *InsightsService) Query(ctx context.Context, projectID string, q InsightsQuery, opts ...Option) (*InsightsResult, error) {
	ro := resolve(opts)

	body := gen.RunInsightsQueryJSONRequestBody{Query: q.Query}
	if len(q.StreamIDs) > 0 {
		body.StreamIds = &q.StreamIDs
	}
	if q.Ts != "" {
		body.Ts = &q.Ts
	}
	if q.Timezone != "" {
		body.Timezone = &q.Timezone
	}

	status, raw, err := s.client.do(ctx, func() (*http.Response, error) {
		return s.client.gen().RunInsightsQuery(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
	if err != nil {
		return nil, err
	}

	// Data is decoded in two steps because the API sends it two ways. The spec
	// says object, and that is what a corrected server sends. What it sends today
	// is a JSON *string* holding the object: the query service returns its result
	// already encoded, v2 passed that through with `render json: result` so it
	// stayed valid, and v3 wraps it in an envelope — `{data: result}.to_json`
	// escapes the string rather than embedding it, so the payload arrives
	// double-encoded.
	//
	// Decoding as RawMessage first accepts both, and unwrapping one level of
	// string leaves callers with the object the spec promises either way. Remove
	// the unwrapping once the server embeds the result properly; the object path
	// needs no change when that happens.
	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta map[string]any  `json:"meta"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, malformed(status, raw, err)
	}

	data, err := decodeInsightsData(envelope.Data)
	if err != nil {
		return nil, malformed(status, raw, err)
	}

	result := &InsightsResult{Data: data, Meta: envelope.Meta}
	if id, ok := envelope.Meta["request_id"].(string); ok {
		result.RequestID = id
	}
	return result, nil
}

// decodeInsightsData reads the query result, unwrapping the double-encoded form.
func decodeInsightsData(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err == nil {
		return data, nil
	}

	// Not an object. A JSON string is the double-encoded case; anything else is
	// genuinely unreadable and the original error is the honest one to report.
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, fmt.Errorf("query result was neither an object nor an encoded one: %w", err)
	}
	if err := json.Unmarshal([]byte(encoded), &data); err != nil {
		return nil, fmt.Errorf("query result was a string but not encoded JSON: %w", err)
	}
	return data, nil
}

// ListStreams returns one page of the project's event streams.
//
// Listing streams is a prerequisite for querying: a query against an unselected
// stream returns nothing, which is indistinguishable from the events not
// existing.
func (s *InsightsService) ListStreams(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Stream], error) {
	return s.listStreams(ctx, projectID, resolve(opts))
}

// ListAllStreams returns every stream for the project, walking pagination.
func (s *InsightsService) ListAllStreams(ctx context.Context, projectID string, opts ...ListAllOption) ([]Stream, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Stream], error) {
		ro.page = page
		return s.listStreams(ctx, projectID, ro)
	})
}

func (s *InsightsService) listStreams(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Stream], error) {
	params := &gen.ListStreamsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[Stream](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListStreams(ctx, s.client.accountID(ro.accountID), projectID, params)
	})
}
