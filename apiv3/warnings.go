package apiv3

import "github.com/honeybadger-io/api-go/internal/gen"

// ResponseWarning is something about a result a caller should know before
// drawing conclusions from it, such as data a plan limit dropped during the
// period the result covers. The API leaves warnings out when there are none.
//
// Branch on Code, which is stable; Message is for people and may change. Code
// is a plain string rather than a closed set, so a warning this client doesn't
// know yet still arrives. StartAt and EndAt bound the period the warning is
// about, in UTC; EndAt is in the future while the limit is still in force.
type ResponseWarning = gen.ResponseWarning
