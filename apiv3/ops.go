package apiv3

import (
	"context"
	"errors"
)

// The helpers below exist because every service method is otherwise the
// same eleven lines: run the call, bail on error, decode the envelope. Keeping
// that in one place is what stops the service layer growing linearly with the
// number of resources — and means a fix to the response contract lands once
// rather than in a dozen near-identical copies. Each names its operationId, which
// tells the retry policy the operation's method and whether it may be resent.

// getOne runs an operation returning a single resource.
func getOne[T any](ctx context.Context, c *Client, opID string, op operation) (*T, error) {
	status, body, err := c.run(ctx, opID, op)
	if err != nil {
		return nil, err
	}
	return decodeSingle[T](status, body)
}

// listOffset runs an operation returning an offset-paginated collection.
func listOffset[T any](ctx context.Context, c *Client, opID string, op operation) (*ListResponse[T], error) {
	status, body, err := c.run(ctx, opID, op)
	if err != nil {
		return nil, err
	}
	return decodeOffsetList[T](status, body)
}

// noContent runs an operation that returns 204 with no body: deletes, and the
// pause/resume toggles. Decoding is skipped rather than attempted, since an
// empty body is the documented success case here and would otherwise look
// malformed.
func noContent(ctx context.Context, c *Client, opID string, op operation) error {
	_, _, err := c.run(ctx, opID, op)
	var gone *goneAfterRetry
	if errors.As(err, &gone) {
		return nil
	}
	return err
}

// listTimeSeries runs an operation returning a time-ordered collection.
func listTimeSeries[T any](ctx context.Context, c *Client, opID string, op operation) (*ListResponse[T], error) {
	status, body, err := c.run(ctx, opID, op)
	if err != nil {
		return nil, err
	}
	return decodeTimeSeriesList[T](status, body)
}
