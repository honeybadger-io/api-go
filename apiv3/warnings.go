package apiv3

import (
	"slices"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// ResponseWarning is something about a result a caller should know before
// drawing conclusions from it, such as data a plan limit dropped during the
// period the result covers. The API leaves warnings out when there are none.
//
// Branch on Code, which is stable; Message is for people and may change. Code
// is a plain string rather than a closed set, so a warning this client doesn't
// know yet still arrives. StartAt and EndAt bound the period the warning is
// about, in UTC; EndAt is in the future while the limit is still in force.
type ResponseWarning = gen.ResponseWarning

// warningSet collects the warnings of a pagination walk. Each page repeats the
// warnings in force when it was served, so it keeps each distinct warning once,
// in the order first seen. Warnings that differ in any field are all kept: if a
// limit changes mid-walk, both versions are true for some of the results.
type warningSet struct {
	list []ResponseWarning
}

func (s *warningSet) add(warnings []ResponseWarning) {
	for _, w := range warnings {
		if !slices.ContainsFunc(s.list, func(seen ResponseWarning) bool { return sameWarning(seen, w) }) {
			s.list = append(s.list, w)
		}
	}
}

func sameWarning(a, b ResponseWarning) bool {
	return a.Code == b.Code &&
		a.Message == b.Message &&
		a.StartAt.Equal(b.StartAt) &&
		a.EndAt.Equal(b.EndAt) &&
		(a.LimitPerMinute == nil) == (b.LimitPerMinute == nil) &&
		(a.LimitPerMinute == nil || *a.LimitPerMinute == *b.LimitPerMinute)
}
