package apiv3

import (
	"encoding/json"
	"fmt"
)

// offsetEnvelope is the documented shape of an offset-paginated collection.
type offsetEnvelope[T any] struct {
	Data       []T            `json:"data"`
	Pagination *Pagination    `json:"pagination"`
	Links      map[string]any `json:"links"`
	Meta       struct {
		RequestID string `json:"request_id"`
	} `json:"meta"`
}

// cursorEnvelope is the same shape for cursor-paginated collections. v3 uses the
// same "pagination" key for both schemes, with a different object under it.
type cursorEnvelope[T any] struct {
	Data       []T               `json:"data"`
	Pagination *CursorPagination `json:"pagination"`
	Links      map[string]any    `json:"links"`
	Meta       struct {
		RequestID string `json:"request_id"`
	} `json:"meta"`
}

// singleEnvelope is the documented shape of a single-resource response.
type singleEnvelope[T any] struct {
	Data *T `json:"data"`
	Meta struct {
		RequestID string `json:"request_id"`
	} `json:"meta"`
}

// decodeOffsetList parses an offset-paginated collection response.
func decodeOffsetList[T any](status int, body []byte) (*ListResponse[T], error) {
	var envelope offsetEnvelope[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, malformed(status, body, err)
	}
	return &ListResponse[T]{
		Data:       envelope.Data,
		Pagination: envelope.Pagination,
		Links:      envelope.Links,
		RequestID:  envelope.Meta.RequestID,
	}, nil
}

// decodeCursorList parses a cursor-paginated collection response.
func decodeCursorList[T any](status int, body []byte) (*ListResponse[T], error) {
	var envelope cursorEnvelope[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, malformed(status, body, err)
	}
	return &ListResponse[T]{
		Data:      envelope.Data,
		Cursor:    envelope.Pagination,
		Links:     envelope.Links,
		RequestID: envelope.Meta.RequestID,
	}, nil
}

// decodeSingle parses a single-resource response.
//
// A missing data member is an error rather than a zero value. The spec marks
// data optional on these responses, so `{}` decodes cleanly — but a caller
// asking for one project should never receive a Project with an empty id and no
// indication anything went wrong.
func decodeSingle[T any](status int, body []byte) (*T, error) {
	var envelope singleEnvelope[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, malformed(status, body, err)
	}
	if envelope.Data == nil {
		return nil, malformed(status, body, fmt.Errorf("response has no data member"))
	}
	return envelope.Data, nil
}

// malformed reports a response whose status said success but whose body did not
// match the documented envelope. Returning an error beats handing back a zero
// value that looks like real data.
func malformed(status int, body []byte, cause error) error {
	apiErr := parseError(status, body)
	apiErr.Message = "response did not match the documented envelope: " + cause.Error()
	return apiErr
}
