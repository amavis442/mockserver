package domain

// Times controls how many times an expectation may be matched before it
// expires. When Unlimited is true the expectation never expires and Remaining
// is ignored. Otherwise Remaining is decremented on each match and the
// expectation expires once it reaches zero.
type Times struct {
	Unlimited bool `json:"unlimited"`
	Remaining int  `json:"remaining"`
}

// Available reports whether the expectation may still be matched.
func (t Times) Available() bool {
	return t.Unlimited || t.Remaining > 0
}

// Consume records a single match. It decrements Remaining for limited
// expectations (never below zero) and is a no-op when Unlimited.
func (t *Times) Consume() {
	if t.Unlimited {
		return
	}
	if t.Remaining > 0 {
		t.Remaining--
	}
}
