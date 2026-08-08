package domain

import "encoding/json"

// AuthConfig specifies authentication requirements for an expectation.
type AuthConfig struct {
	Required bool `json:"required"`
}

// Response is the reply returned when an expectation matches an incoming
// request. Body may be a JSON string, object, array, number, etc.; it is stored
// verbatim as raw JSON and written to the client as-is.
type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// Expectation ties a request matcher to the response that should be returned
// when it matches. Higher Priority values are evaluated first.
type Expectation struct {
	ID       string         `json:"id,omitempty"`
	Priority int            `json:"priority,omitempty"`
	Times    Times          `json:"times"`
	Request  RequestMatcher `json:"request"`
	Response Response       `json:"response"`
	Auth     *AuthConfig    `json:"auth,omitempty"`
}

// expectationAlias avoids infinite recursion in UnmarshalJSON while letting us
// detect whether the "times" key was present in the source JSON.
type expectationAlias struct {
	ID       string         `json:"id,omitempty"`
	Priority int            `json:"priority,omitempty"`
	Times    *Times         `json:"times"`
	Request  RequestMatcher `json:"request"`
	Response Response       `json:"response"`
	Auth     *AuthConfig    `json:"auth,omitempty"`
}

// UnmarshalJSON defaults a missing "times" field to unlimited, so expectations
// without an explicit times never expire.
func (e *Expectation) UnmarshalJSON(data []byte) error {
	var a expectationAlias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	e.ID = a.ID
	e.Priority = a.Priority
	e.Request = a.Request
	e.Response = a.Response
	e.Auth = a.Auth
	if a.Times == nil {
		e.Times = Times{Unlimited: true}
	} else {
		e.Times = *a.Times
	}
	return nil
}
