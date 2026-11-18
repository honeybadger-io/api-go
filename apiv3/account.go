package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// AccountUsage is how much of its plan's quotas the account is using.
type AccountUsage = gen.GetAccountUsage200JSONResponseBody_Data

// AccountQuota is what the account is allowed and how much of it is in use.
// Errors count over the current billing period, Insights in bytes over the
// current UTC day, and uptime sites and check-ins as the number in use now.
// UptimeLimit and CheckInsLimit are null when the plan doesn't limit them; an
// InsightsDailyLimit of 0 is unlimited unless InsightsLimited is true, which it
// is while Insights data is being dropped for exceeding the quota.
type AccountQuota = gen.GetAccountUsage200JSONResponseBody_Data_Quota

// AccountService reads the account the credential resolves to. No request
// names an account, so a credential covering more than one gets
// ErrAmbiguousAccount; use one bound to a single account.
type AccountService struct {
	client *Client
}

// Usage returns the account's usage against its quotas. It needs account:read.
// When the live Insights figures can't be read it fails with
// ErrServiceUnavailable, which is worth retrying shortly.
func (s *AccountService) Usage(ctx context.Context) (*AccountUsage, error) {
	return getOne[AccountUsage](ctx, s.client, "getAccountUsage", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetAccountUsage(ctx)
	})
}
