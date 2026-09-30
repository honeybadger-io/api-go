// Package honeybadgerapi is the module root. The clients live one level down.
//
// There are two, one per API version, and which you want depends on the
// credential you hold:
//
//	github.com/honeybadger-io/api-go/apiv2  — the v2 Data API
//	github.com/honeybadger-io/api-go/apiv3  — the v3 API
//
// v3 is where new work belongs. It accepts scoped API tokens (`hbt_` for a
// personal one, `hba_` for an account one) and OAuth access tokens, always as
// Bearer. v2's older personal auth tokens are rejected by v3 outright, so moving
// to it means a new credential rather than only new code.
//
// Both packages are generated from, or hand-written against, the same vendored
// OpenAPI bundle under openapi/. See openapi/README.md for how that is refreshed
// and openapi/GAPS.md for the v2 capabilities v3 cannot yet express.
//
// # Layout
//
// Until v0.9.0 the v2 services lived in this package. They moved to apiv2 so the
// two versions sit side by side under names that say which is which, rather than
// one being the unmarked default. A v0.8.0 caller updates its import path and the
// client constructor:
//
//	// before
//	client := honeybadgerapi.NewClient().WithAuthToken(token)
//
//	// after
//	client := apiv2.NewClient().WithAuthToken(token)
//
// Nothing else changed in v2's surface — the move was mechanical.
//
// The directories are apiv2 and apiv3 rather than v2 and v3 because Go reads a
// trailing /vN path element (N≥2) as a module major-version suffix, so
// api-go/v2 would be ambiguous with major version 2 of this module. The apivN
// convention follows Google's generated clients.
package honeybadgerapi
