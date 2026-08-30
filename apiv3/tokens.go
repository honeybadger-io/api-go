package apiv3

import (
	"context"
	"net/http"
	"slices"
)

// TokensService describes the credential making the request.
type TokensService struct {
	client *Client
}

// TokenKind is what sort of credential is in use.
type TokenKind string

const (
	// TokenKindUser is a scoped API token belonging to a person (`hbt_`).
	TokenKindUser TokenKind = "user"
	// TokenKindAccount is a scoped API token belonging to an account (`hba_`).
	TokenKindAccount TokenKind = "account"
	// TokenKindOAuth is an access token issued to an application acting for a
	// user.
	TokenKindOAuth TokenKind = "oauth"
)

// TokenInfo describes a credential and what it can reach.
type TokenInfo struct {
	Kind TokenKind
	Name string

	// Scopes are the credential's granular permissions, such as "faults:read".
	// For an OAuth grant these are its read/write aliases already expanded to
	// the API surface as it stood when the grant was consented to.
	Scopes []string

	// AccountID is the account the credential is bound to. Passing it as an
	// explicit account is how a caller recovers from ambiguous_account.
	AccountID string

	// ProjectIDs are the projects the credential can reach. Empty means an
	// account with no visible projects; an unrestricted credential lists every
	// project in the account.
	ProjectIDs []string

	// ExpiresAt and LastUsedAt are nil when the API reports them as null.
	ExpiresAt  *string
	LastUsedAt *string
}

// HasScope reports whether the credential holds the given scope.
func (t *TokenInfo) HasScope(scope string) bool {
	return slices.Contains(t.Scopes, scope)
}

// Get describes the credential making the request.
//
// It requires no scope by design, so a client holding a narrow token can always
// discover its own limits. Two uses worth knowing:
//
//   - Recovering from ambiguous_account: AccountID names the account the
//     credential is bound to. v3 resolves the account from the credential,
//     so the fix is to use a credential scoped to one account.
//   - Gating features ahead of a 403: Scopes lets a caller check what is
//     permitted rather than discovering it from a refusal.
func (s *TokensService) Get(ctx context.Context) (*TokenInfo, error) {
	// Decoded into a local type rather than the generated one: the generated model
	// for this response is an anonymous struct, so there is no named type to
	// decode into.
	type payload struct {
		Kind       string   `json:"kind"`
		Name       *string  `json:"name"`
		Scopes     []string `json:"scopes"`
		AccountID  string   `json:"account_id"`
		ProjectIDs []string `json:"project_ids"`
		ExpiresAt  *string  `json:"expires_at"`
		LastUsedAt *string  `json:"last_used_at"`
	}
	data, err := getOne[payload](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetToken(ctx)
	})
	if err != nil {
		return nil, err
	}

	info := &TokenInfo{
		Kind:       TokenKind(data.Kind),
		Scopes:     data.Scopes,
		AccountID:  data.AccountID,
		ProjectIDs: data.ProjectIDs,
		ExpiresAt:  data.ExpiresAt,
		LastUsedAt: data.LastUsedAt,
	}
	if data.Name != nil {
		info.Name = *data.Name
	}
	return info, nil
}
