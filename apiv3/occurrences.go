package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// OccurrencePeriod is the window an occurrence report covers.
//
// Both ends are inclusive, so each period returns one bucket more than its name
// suggests: an hour gives 61 one-minute buckets, a day 25 hourly, a week 8 daily,
// a month 31 daily. An unrecognised value falls back to hour, and the response's
// meta says so rather than failing.
type OccurrencePeriod string

const (
	PeriodHour  OccurrencePeriod = "hour"
	PeriodDay   OccurrencePeriod = "day"
	PeriodWeek  OccurrencePeriod = "week"
	PeriodMonth OccurrencePeriod = "month"
)

// OccurrenceOptions narrow an occurrence report.
type OccurrenceOptions struct {
	// Period defaults to hour when empty.
	Period OccurrencePeriod

	// Environment counts only notices from faults in that environment.
	Environment string

	// AccountID addresses a specific account rather than resolving one from the
	// credential.
	AccountID string
}

// Occurrences returns notice counts over time for one project.
//
// Untyped: the payload is a bucket series the spec does not pin.
func (s *ProjectsService) Occurrences(ctx context.Context, projectID string, o OccurrenceOptions) (map[string]any, error) {
	params := &gen.GetProjectOccurrencesParams{}
	if o.Period != "" {
		period := gen.GetProjectOccurrencesParamsPeriod(o.Period)
		params.Period = &period
	}
	if o.Environment != "" {
		env := gen.OccurrenceEnvironment(o.Environment)
		params.Environment = &env
	}

	data, err := getOne[map[string]any](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetProjectOccurrences(ctx, projectID, params)
	})
	if err != nil {
		return nil, err
	}
	return *data, nil
}

// OccurrenceSeries is one project's notice counts, bucketed over the requested
// window. Both ends of the window are inclusive, so a series carries one bucket
// more than the period name suggests.
type OccurrenceSeries = gen.OccurrenceSeries

// AccountOccurrences returns notice counts over time for every project the
// credential can reach, walking pagination.
//
// This is the all-projects report v2 offered. Note two differences: it is
// account-scoped rather than global, so a credential covering several accounts
// reports on one of them; and it returns a series per project rather than a
// single object, which is why it is a collection here.
func (s *ProjectsService) AccountOccurrences(ctx context.Context, o OccurrenceOptions) ([]OccurrenceSeries, error) {
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[OccurrenceSeries], error) {
		params := &gen.ListAccountOccurrencesParams{}
		if page > 0 {
			p := gen.Page(page)
			params.Page = &p
		}
		if o.Period != "" {
			period := gen.ListAccountOccurrencesParamsPeriod(o.Period)
			params.Period = &period
		}
		if o.Environment != "" {
			env := gen.OccurrenceEnvironment(o.Environment)
			params.Environment = &env
		}

		return listOffset[OccurrenceSeries](ctx, s.client, func() (*http.Response, error) {
			return s.client.gen().ListAccountOccurrences(ctx, params)
		})
	})
}
