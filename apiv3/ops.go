package apiv3

import (
	"context"
	"net/http"
)

// operation is a generated call, ready to run.
type operation func() (*http.Response, error)

// The three helpers below exist because every service method is otherwise the
// same eleven lines: run the call, bail on error, decode the envelope. Keeping
// that in one place is what stops the service layer growing linearly with the
// number of resources — and means a fix to the response contract lands once
// rather than in a dozen near-identical copies.

// getOne runs an operation returning a single resource.
func getOne[T any](ctx context.Context, c *Client, op operation) (*T, error) {
	status, body, err := c.do(ctx, op)
	if err != nil {
		return nil, err
	}
	return decodeSingle[T](status, body)
}

// listOffset runs an operation returning an offset-paginated collection.
func listOffset[T any](ctx context.Context, c *Client, op operation) (*ListResponse[T], error) {
	status, body, err := c.do(ctx, op)
	if err != nil {
		return nil, err
	}
	return decodeOffsetList[T](status, body)
}

// listTimeSeries runs an operation returning a time-ordered collection.
func listTimeSeries[T any](ctx context.Context, c *Client, op operation) (*ListResponse[T], error) {
	status, body, err := c.do(ctx, op)
	if err != nil {
		return nil, err
	}
	return decodeTimeSeriesList[T](status, body)
}
