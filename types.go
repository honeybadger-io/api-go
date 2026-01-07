package honeybadgerapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Number is a custom type that can unmarshal from either a string or integer JSON value
// and stores it as an integer
type Number int

// UnmarshalJSON implements json.Unmarshaler interface to handle both string and integer values
func (n *Number) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as integer first
	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		*n = Number(num)
		return nil
	}

	// If that fails, try as string and parse to int
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// Try to parse the string as an integer
		parsed, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("Number: cannot parse string %q as integer: %v", str, err)
		}
		*n = Number(parsed)
		return nil
	}

	return fmt.Errorf("Number: cannot unmarshal value into integer or string")
}

// User represents a Honeybadger user
type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Account represents a Honeybadger account
type Account struct {
	ID             string                 `json:"id"`
	Email          string                 `json:"email"`
	Name           string                 `json:"name"`
	Active         *bool                  `json:"active,omitempty"`
	Parked         *bool                  `json:"parked,omitempty"`
	QuotaConsumed  *float64               `json:"quota_consumed,omitempty"` // Percentage
	APIStats       map[string]interface{} `json:"api_stats,omitempty"`
}

// AccountUser represents a user associated with an account
type AccountUser struct {
	ID    int    `json:"id"`
	Role  string `json:"role"` // Member, Billing, Admin, Owner
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AccountUserListResponse represents the response from listing account users
type AccountUserListResponse struct {
	Results []AccountUser `json:"results"`
}

// AccountUserUpdateRequest represents the request body for updating an account user
type AccountUserUpdateRequest struct {
	User struct {
		Role string `json:"role"` // Member, Billing, Admin, Owner
	} `json:"user"`
}

// AccountInvitation represents an invitation to join an account
type AccountInvitation struct {
	ID        int        `json:"id"`
	Token     string     `json:"token"`
	Email     string     `json:"email"`
	Role      string     `json:"role"` // Member, Billing, Admin, Owner
	TeamIDs   []int      `json:"team_ids"`
	CreatedAt time.Time  `json:"created_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
}

// AccountInvitationListResponse represents the response from listing account invitations
type AccountInvitationListResponse struct {
	Results []AccountInvitation `json:"results"`
}

// AccountInvitationRequest represents the request body for creating/updating an invitation
type AccountInvitationRequest struct {
	Invitation AccountInvitationParams `json:"invitation"`
}

// AccountInvitationParams represents parameters for creating/updating an account invitation
type AccountInvitationParams struct {
	Email   string `json:"email,omitempty"`
	Role    string `json:"role,omitempty"` // Member, Billing, Admin, Owner
	TeamIDs []int  `json:"team_ids,omitempty"`
}

// AccountListResponse represents the response for listing accounts
type AccountListResponse struct {
	Results []Account `json:"results"`
}

// StatusPage represents a status page
type StatusPage struct {
	ID               int                    `json:"id"`
	Name             string                 `json:"name"`
	AccountID        int                    `json:"account_id"`
	Domain           *string                `json:"domain"`
	URL              string                 `json:"url"`
	CreatedAt        time.Time              `json:"created_at"`
	DomainVerifiedAt *time.Time             `json:"domain_verified_at"`
	Sites            []string               `json:"sites"`     // Array of site IDs
	CheckIns         []string               `json:"check_ins"` // Array of check-in slugs
	HideBranding     *bool                  `json:"hide_branding,omitempty"`
	Features         map[string]interface{} `json:"features,omitempty"`
}

// StatusPageListResponse represents the response for listing status pages
type StatusPageListResponse struct {
	Results []StatusPage `json:"results"`
}

// StatusPageRequest represents the request body for creating/updating a status page
type StatusPageRequest struct {
	StatusPage StatusPageParams `json:"status_page"`
}

// StatusPageParams represents parameters for creating/updating a status page
type StatusPageParams struct {
	Name         string                 `json:"name,omitempty"`
	Domain       *string                `json:"domain,omitempty"`
	Sites        []string               `json:"sites,omitempty"`
	CheckIns     []string               `json:"check_ins,omitempty"`
	HideBranding *bool                  `json:"hide_branding,omitempty"`
	Features     map[string]interface{} `json:"features,omitempty"`
}

// Team represents a Honeybadger team
type Team struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	AccountID int       `json:"account_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// TeamListResponse represents the response from listing teams
type TeamListResponse struct {
	Results []Team `json:"results"`
}

// TeamRequest represents the request body for creating/updating a team
type TeamRequest struct {
	Team struct {
		Name string `json:"name"`
	} `json:"team"`
}

// TeamMember represents a member of a team
type TeamMember struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}

// TeamMemberListResponse represents the response from listing team members
type TeamMemberListResponse struct {
	Results []TeamMember `json:"results"`
}

// TeamMemberUpdateRequest represents the request body for updating a team member
type TeamMemberUpdateRequest struct {
	TeamMember struct {
		Admin bool `json:"admin"`
	} `json:"team_member"`
}

// TeamInvitation represents an invitation to join a team
type TeamInvitation struct {
	ID        int       `json:"id"`
	Token     string    `json:"token"`
	Email     string    `json:"email"`
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"created_at"`
	AcceptedAt *time.Time `json:"accepted_at"`
	Message   *string   `json:"message"`
}

// TeamInvitationRequest represents the request body for creating/updating a team invitation
type TeamInvitationRequest struct {
	TeamInvitation TeamInvitationParams `json:"team_invitation"`
}

// TeamInvitationParams represents parameters for creating/updating a team invitation
type TeamInvitationParams struct {
	Email   string  `json:"email,omitempty"`
	Admin   *bool   `json:"admin,omitempty"`
	Message *string `json:"message,omitempty"`
}

// TeamInvitationListResponse represents the response from listing team invitations
type TeamInvitationListResponse struct {
	Results []TeamInvitation `json:"results"`
}

// Environment represents a project environment
type Environment struct {
	ID            int       `json:"id"`
	ProjectID     int       `json:"project_id"`
	Name          string    `json:"name"`
	Notifications bool      `json:"notifications"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EnvironmentRequest represents the request body for creating/updating an environment
type EnvironmentRequest struct {
	Environment EnvironmentParams `json:"environment"`
}

// EnvironmentParams represents parameters for creating/updating an environment
type EnvironmentParams struct {
	Name          string `json:"name,omitempty"`
	Notifications *bool  `json:"notifications,omitempty"` // Defaults to true
}

// Site represents an uptime monitoring site
type Site struct {
	ID            string     `json:"id"`
	Active        bool       `json:"active"`
	Frequency     int        `json:"frequency"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	Match         *string    `json:"match"`
	MatchType     string     `json:"match_type"`
	Name          string     `json:"name"`
	State         string     `json:"state"`
	URL           string     `json:"url"`
}

// SiteListResponse represents the response from listing sites
type SiteListResponse struct {
	Results []Site `json:"results"`
}

// SiteCreateRequest represents the request for creating a site
type SiteCreateRequest struct {
	Site SiteParams `json:"site"`
}

// SiteParams represents parameters for creating/updating a site
type SiteParams struct {
	Name            string              `json:"name,omitempty"`
	URL             string              `json:"url,omitempty"`
	Frequency       *int                `json:"frequency,omitempty"`        // 1, 5, or 15 minutes
	Match           *string             `json:"match,omitempty"`            // Status code or text pattern
	MatchType       *string             `json:"match_type,omitempty"`       // "success", "exact", "include", "exclude"
	RequestMethod   *string             `json:"request_method,omitempty"`   // GET, POST, PUT, PATCH, DELETE
	RequestBody     *string             `json:"request_body,omitempty"`     // Request payload
	RequestHeaders  []map[string]string `json:"request_headers,omitempty"`  // [{"key": "...", "value": "..."}]
	Locations       []string            `json:"locations,omitempty"`        // Virginia, Oregon, Frankfurt, Singapore, London
	ValidateSSL     *bool               `json:"validate_ssl,omitempty"`     // Default true
	Timeout         *int                `json:"timeout,omitempty"`          // 30-120 seconds (Business/Enterprise)
	OutageThreshold *int                `json:"outage_threshold,omitempty"` // Failed checks to trigger alert
	Active          *bool               `json:"active,omitempty"`           // Enable/disable checks
}

// Outage represents a site outage
type Outage struct {
	DownAt    time.Time              `json:"down_at"`
	UpAt      *time.Time             `json:"up_at"`
	CreatedAt time.Time              `json:"created_at"`
	Status    int                    `json:"status"`
	Reason    string                 `json:"reason"`
	Headers   map[string]interface{} `json:"headers"`
}

// OutageListResponse represents the response from listing outages
type OutageListResponse struct {
	Results []Outage `json:"results"`
}

// OutageListOptions represents query parameters for listing outages
type OutageListOptions struct {
	CreatedAfter  int64 `url:"created_after,omitempty"`  // Unix timestamp
	CreatedBefore int64 `url:"created_before,omitempty"` // Unix timestamp
	Limit         int   `url:"limit,omitempty"`          // 1-25, default 25
}

// UptimeCheck represents a single uptime check
type UptimeCheck struct {
	CreatedAt time.Time `json:"created_at"`
	Duration  int       `json:"duration"` // milliseconds
	Location  string    `json:"location"`
	Up        bool      `json:"up"`
}

// UptimeCheckListResponse represents the response from listing uptime checks
type UptimeCheckListResponse struct {
	Results []UptimeCheck `json:"results"`
}

// UptimeCheckListOptions represents query parameters for listing uptime checks
type UptimeCheckListOptions struct {
	CreatedAfter  int64 `url:"created_after,omitempty"`  // Unix timestamp
	CreatedBefore int64 `url:"created_before,omitempty"` // Unix timestamp
	Limit         int   `url:"limit,omitempty"`          // 1-25, default 25
}

// Project represents a Honeybadger project
type Project struct {
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Active               bool       `json:"active"`
	CreatedAt            time.Time  `json:"created_at"`
	EarliestNoticeAt     *time.Time `json:"earliest_notice_at"`
	LastNoticeAt         *time.Time `json:"last_notice_at"`
	Environments         []string   `json:"environments"`
	FaultCount           int        `json:"fault_count"`
	UnresolvedFaultCount int        `json:"unresolved_fault_count"`
	Token                string     `json:"token"`
	Sites                []Site     `json:"sites"`
	Teams                []Team     `json:"teams"`
	Users                []User     `json:"users"`
}

// Fault represents a Honeybadger fault
type Fault struct {
	ID                  int        `json:"id"`
	Action              string     `json:"action"`
	Assignee            *User      `json:"assignee"`
	CommentsCount       int        `json:"comments_count"`
	Component           string     `json:"component"`
	CreatedAt           time.Time  `json:"created_at"`
	Environment         string     `json:"environment"`
	Ignored             bool       `json:"ignored"`
	Klass               string     `json:"klass"`
	LastNoticeAt        *time.Time `json:"last_notice_at"`
	Message             string     `json:"message"`
	NoticesCount        int        `json:"notices_count"`
	NoticesCountInRange *int       `json:"notices_count_in_range,omitempty"` // Added when fault list search query affects notice count
	ProjectID           int        `json:"project_id"`
	Resolved            bool       `json:"resolved"`
	Tags                []string   `json:"tags"`
	URL                 string     `json:"url"`
}

// NoticeEnvironment represents the environment information for a notice
type NoticeEnvironment struct {
	EnvironmentName string                 `json:"environment_name"`
	Hostname        string                 `json:"hostname"`
	ProjectRoot     interface{}            `json:"project_root"` // Can be string or object
	Revision        *string                `json:"revision"`
	Stats           map[string]interface{} `json:"stats"`
	Time            string                 `json:"time"`
	PID             int                    `json:"pid"`
}

// NoticeRequest represents the HTTP request information for a notice
type NoticeRequest struct {
	Action    *string                `json:"action"`
	Component *string                `json:"component"`
	Context   map[string]interface{} `json:"context"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	URL       *string                `json:"url"`
	User      map[string]interface{} `json:"user"`
}

// BacktraceEntry represents a single entry in the error backtrace
type BacktraceEntry struct {
	Number  Number                 `json:"number"`
	Column  *Number                `json:"column,omitempty"`
	File    string                 `json:"file"`
	Method  string                 `json:"method"`
	Class   string                 `json:"class,omitempty"`
	Type    string                 `json:"type,omitempty"`
	Args    []interface{}          `json:"args,omitempty"`
	Source  map[string]interface{} `json:"source,omitempty"`
	Context string                 `json:"context,omitempty"`
}

// Notice represents a Honeybadger notice (individual error occurrence)
type Notice struct {
	ID               string                 `json:"id"`
	CreatedAt        time.Time              `json:"created_at"`
	Environment      NoticeEnvironment      `json:"environment"`
	EnvironmentName  string                 `json:"environment_name"`
	Cookies          map[string]interface{} `json:"cookies"`
	FaultID          int                    `json:"fault_id"`
	URL              string                 `json:"url"`
	Message          string                 `json:"message"`
	WebEnvironment   map[string]interface{} `json:"web_environment"`
	Request          NoticeRequest          `json:"request"`
	Backtrace        []BacktraceEntry       `json:"backtrace"`
	ApplicationTrace []BacktraceEntry       `json:"application_trace"`
	Deploy           interface{}            `json:"deploy"` // Can be null or object
}

// PaginationLinks represents the pagination links in API responses
type PaginationLinks struct {
	Next string `json:"next"`
	Prev string `json:"prev"`
	Self string `json:"self"`
}

// ListResponse represents a generic list response with results and pagination links
type ListResponse[T any] struct {
	Results []T             `json:"results"`
	Links   PaginationLinks `json:"links"`
}

// Comment represents a comment on a fault
type Comment struct {
	ID           int       `json:"id"`
	FaultID      int       `json:"fault_id"`
	Event        string    `json:"event"`
	Source       string    `json:"source"`
	NoticesCount int       `json:"notices_count"`
	CreatedAt    time.Time `json:"created_at"`
	Author       *User     `json:"author"`
	Body         string    `json:"body"`
}

// CommentRequest represents the request body for creating/updating a comment
type CommentRequest struct {
	Comment struct {
		Body string `json:"body"`
	} `json:"comment"`
}

// Deployment represents a deployment in Honeybadger
type Deployment struct {
	ID            int       `json:"id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Environment   string    `json:"environment"`
	LocalUsername string    `json:"local_username"`
	ProjectID     int       `json:"project_id"`
	Repository    string    `json:"repository"`
	Revision      string    `json:"revision"`
}

// DeploymentListResponse represents the response from listing deployments
type DeploymentListResponse struct {
	Results []Deployment `json:"results"`
}

// DeploymentListOptions represents query parameters for listing deployments
type DeploymentListOptions struct {
	Environment   string `url:"environment,omitempty"`
	LocalUsername string `url:"local_username,omitempty"`
	CreatedAfter  int64  `url:"created_after,omitempty"`  // Unix timestamp
	CreatedBefore int64  `url:"created_before,omitempty"` // Unix timestamp
	Limit         int    `url:"limit,omitempty"`          // Max 25
}

// CheckIn represents a check-in in Honeybadger
type CheckIn struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	ScheduleType  string     `json:"schedule_type"`  // "simple" or "cron"
	ReportPeriod  *string    `json:"report_period"`  // For simple schedules
	GracePeriod   *string    `json:"grace_period"`   // Optional
	CronSchedule  *string    `json:"cron_schedule"`  // For cron schedules
	CronTimezone  *string    `json:"cron_timezone"`  // Optional, defaults to UTC
	ProjectID     int        `json:"project_id"`
	CreatedAt     time.Time  `json:"created_at"`
	LastCheckInAt *time.Time `json:"last_check_in_at"`
}

// CheckInListResponse represents the response from listing check-ins
type CheckInListResponse struct {
	Results []CheckIn `json:"results"`
}

// CheckInRequest represents the request body for creating/updating a check-in
type CheckInRequest struct {
	CheckIn CheckInParams `json:"check_in"`
}

// CheckInParams represents the parameters for creating/updating a check-in
type CheckInParams struct {
	Name         string  `json:"name,omitempty"`
	Slug         string  `json:"slug,omitempty"`
	ScheduleType string  `json:"schedule_type,omitempty"` // "simple" or "cron"
	ReportPeriod *string `json:"report_period,omitempty"` // For simple schedules
	GracePeriod  *string `json:"grace_period,omitempty"`  // Optional
	CronSchedule *string `json:"cron_schedule,omitempty"` // For cron schedules
	CronTimezone *string `json:"cron_timezone,omitempty"` // Optional
}

// CheckInBulkUpdateResponse represents the response for bulk updating check-ins
type CheckInBulkUpdateResponse struct {
	Create []CheckInBulkResult `json:"create"`
	Update []CheckInBulkResult `json:"update"`
	Delete []CheckInBulkResult `json:"delete"`
}

// CheckInBulkResult represents a single result in a bulk operation
type CheckInBulkResult struct {
	Success bool    `json:"success"`
	ID      *string `json:"id,omitempty"`
	Slug    string  `json:"slug,omitempty"`
	Error   string  `json:"error,omitempty"`
}
