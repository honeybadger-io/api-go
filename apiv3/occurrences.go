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
// a month 31 daily. An unrecognised value is refused with 400 invalid_parameter.
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
}

// ProjectOccurrences is one project's occurrence series, with any warnings
// about the window it covers.
//
// It embeds the series so it serializes flat, as {buckets, project_id,
// warnings}: callers such as the MCP server pass it straight through.
type ProjectOccurrences struct {
	OccurrenceSeries

	// Warnings say what a caller should know before drawing conclusions from
	// the counts, such as errors dropped while the project was rate limited, so
	// a dip isn't read as fewer errors happening. Nil when the API sent none.
	Warnings []ResponseWarning `json:"warnings,omitempty"`
}

// Occurrences returns notice counts over time for one project.
func (s *ProjectsService) Occurrences(ctx context.Context, projectID string, o OccurrenceOptions) (*ProjectOccurrences, error) {
	params := &gen.GetProjectOccurrencesParams{}
	if o.Period != "" {
		period := gen.GetProjectOccurrencesParamsPeriod(o.Period)
		params.Period = &period
	}
	if o.Environment != "" {
		env := gen.OccurrenceEnvironment(o.Environment)
		params.Environment = &env
	}

	series, meta, err := getOneWithMeta[OccurrenceSeries](ctx, s.client, "getProjectOccurrences", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetProjectOccurrences(ctx, projectID, params)
	})
	if err != nil {
		return nil, err
	}
	return &ProjectOccurrences{OccurrenceSeries: *series, Warnings: meta.Warnings}, nil
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

		return listOffset[OccurrenceSeries](ctx, s.client, "listAccountOccurrences", func(ctx context.Context) (*http.Response, error) {
			return s.client.gen().ListAccountOccurrences(ctx, params)
		})
	})
}
