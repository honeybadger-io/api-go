package apiv3

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInsightsQuerySendsBody(t *testing.T) {
	var got map[string]any
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		writeJSON(w, 0, `{"data":{"results":[]},"meta":{"request_id":"req_q"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{
		Query:     "@size > 0 | limit 10",
		StreamIDs: []string{"str_1", "str_2"},
		Ts:        "1h",
		Timezone:  "UTC",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if want := "/v3/projects/Xk9mZp/insights/queries"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if got["query"] != "@size > 0 | limit 10" {
		t.Errorf("query = %v", got["query"])
	}
	if got["ts"] != "1h" || got["timezone"] != "UTC" {
		t.Errorf("ts/timezone = %v/%v", got["ts"], got["timezone"])
	}
	ids, ok := got["stream_ids"].([]any)
	if !ok || len(ids) != 2 {
		t.Errorf("stream_ids = %v", got["stream_ids"])
	}
}

// Omitted optional fields must not appear at all, since an empty ts or timezone
// is not the same as a default one.
func TestInsightsQueryOmitsEmptyFields(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		writeJSON(w, 0, `{"data":{}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Insights.Query(context.Background(), "Xk9mZp",
		InsightsQuery{Query: "count()"}); err != nil {
		t.Fatalf("Query: %v", err)
	}

	for _, key := range []string{"ts", "timezone", "stream_ids"} {
		if _, present := got[key]; present {
			t.Errorf("%q was sent despite being empty", key)
		}
	}
}

// The result shape is defined by the query service, not this API, so it is passed
// through rather than modelled.
func TestInsightsQueryPassesResultThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{
		  "results":[{"count":42,"bucket":"2026-07-29T00:00:00Z"}],
		  "fields":["count","bucket"],
		  "total_rows":1
		},"meta":{"request_id":"req_q"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	res, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{Query: "count()"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if res.RequestID != "req_q" {
		t.Errorf("RequestID = %q", res.RequestID)
	}
	rows, ok := res.Data["results"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("Data[results] = %v", res.Data["results"])
	}
	first, ok := rows[0].(map[string]any)
	if !ok || first["count"] != float64(42) {
		t.Errorf("first row = %v", rows[0])
	}
	if res.Data["total_rows"] != float64(1) {
		t.Errorf("total_rows = %v", res.Data["total_rows"])
	}
}

// v3 rejects a bad query with 422 rather than v2's inline error on a 200, so the
// failure must arrive as an error and not as an empty result.
func TestInsightsQueryRejectedQueryIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity,
			`{"error":{"code":"validation_error","message":"syntax error at ' | limit'"},
			  "meta":{"request_id":"req_bad"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{Query: "| limit"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if apiErr.RequestID != "req_bad" {
		t.Errorf("RequestID = %q, want req_bad", apiErr.RequestID)
	}
	// The query service's own message is more specific than anything this client
	// could substitute, so it must survive.
	if apiErr.Message != "syntax error at ' | limit'" {
		t.Errorf("Message = %q", apiErr.Message)
	}
}

// A 503 from the query service is documented as retryable, so it must be
// distinguishable from a permanent failure.
func TestInsightsQueryServiceUnavailableIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable,
			`{"error":{"code":"service_unavailable","message":"The query service was unreachable"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{Query: "count()"})
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("err = %v, want ErrServiceUnavailable", err)
	}
}

func TestListStreams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/streams"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":[
		  {"id":"01h7vm19r5","name":"Default","slug":"default","internal":false},
		  {"id":"01h7vm19r6","name":"Internal","slug":"internal","internal":true}
		],"pagination":{"page":1,"per_page":25,"total_count":2,"total_pages":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Insights.ListStreams(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("ListStreams: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("Data = %v, want 2 streams", resp.Data)
	}
	if resp.Data[0].Slug == nil || *resp.Data[0].Slug != "default" {
		t.Errorf("first slug = %v", resp.Data[0].Slug)
	}
	// internal marks the stream carrying Honeybadger-generated event classes,
	// which are only queryable when it is selected.
	if resp.Data[1].Internal == nil || !*resp.Data[1].Internal {
		t.Errorf("second stream internal = %v, want true", resp.Data[1].Internal)
	}
}

// The API sends the query result two ways: the object the spec documents, and a
// JSON string holding that object, which is what the v3 envelope produces today.
// Both must land on the same result — a caller should not have to know which
// server it reached.
func TestInsightsQueryAcceptsBothEncodings(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"object, as the spec documents", `{"data":{"results":[{"count":7}]},"meta":{"request_id":"r1"}}`},
		{"string, as the envelope sends today", `{"data":"{\"results\":[{\"count\":7}]}","meta":{"request_id":"r1"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := captureWrite(t, http.StatusOK, tc.body)

			got, err := c.Insights.Query(context.Background(), "Xk9mZp",
				InsightsQuery{Query: "stats count()"})
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			results, ok := got.Data["results"].([]any)
			if !ok || len(results) != 1 {
				t.Fatalf("Data = %#v, want one result", got.Data)
			}
			if got.RequestID != "r1" {
				t.Errorf("RequestID = %q", got.RequestID)
			}
		})
	}
}

// A string that is not encoded JSON is a real failure and must not be swallowed.
func TestInsightsQueryRejectsUnreadableData(t *testing.T) {
	c, _ := captureWrite(t, http.StatusOK, `{"data":"not json at all"}`)

	if _, err := c.Insights.Query(context.Background(), "Xk9mZp",
		InsightsQuery{Query: "stats count()"}); err == nil {
		t.Fatal("err = nil, want a decode failure")
	}
}
