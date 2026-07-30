package apiv3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// followTimeSeries fetches a page from a navigation link supplied by a previous
// response.
//
// It is a package-level function rather than a method because Go methods cannot
// take type parameters.
//
// The link comes from the server, and this request carries the caller's
// credential, so the target is checked against the configured base URL first. A
// link pointing elsewhere would send the token to another host; refusing is the
// only safe response, and a legitimate API never emits one.
func followTimeSeries[T any](ctx context.Context, c *Client, link string) (*ListResponse[T], error) {
	target, err := c.validateLink(link)
	if err != nil {
		return nil, err
	}

	return listTimeSeries[T](ctx, c, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if err := c.authorize(ctx, req); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
}

// validateLink resolves a navigation link and confirms it addresses the same
// origin as the client's base URL.
//
// Relative links are resolved against the base URL. Absolute links must match
// scheme, host, and port exactly.
func (c *Client) validateLink(link string) (string, error) {
	base, err := url.Parse(c.serverURL())
	if err != nil {
		return "", fmt.Errorf("apiv3: client base URL %q is not a valid URL: %w", c.serverURL(), err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("apiv3: pagination link %q is not a valid URL: %w", link, err)
	}

	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != base.Scheme || resolved.Host != base.Host {
		return "", fmt.Errorf(
			"%w: pagination link %q points at %s://%s, not the configured host %s://%s",
			ErrUntrustedLink, link, resolved.Scheme, resolved.Host, base.Scheme, base.Host)
	}
	return resolved.String(), nil
}
