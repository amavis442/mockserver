package domain

import (
	"encoding/json"
	"testing"
)

func TestTimes_Available(t *testing.T) {
	tests := []struct {
		name  string
		times Times
		want  bool
	}{
		{name: "unlimited is always available", times: Times{Unlimited: true}, want: true},
		{name: "unlimited ignores remaining", times: Times{Unlimited: true, Remaining: 0}, want: true},
		{name: "remaining above zero is available", times: Times{Remaining: 1}, want: true},
		{name: "remaining zero is not available", times: Times{Remaining: 0}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.times.Available(); got != tt.want {
				t.Errorf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimes_Consume(t *testing.T) {
	t.Run("unlimited does not decrement", func(t *testing.T) {
		times := Times{Unlimited: true, Remaining: 5}
		times.Consume()
		if times.Remaining != 5 {
			t.Errorf("Remaining = %d, want 5 (unlimited must not decrement)", times.Remaining)
		}
	})

	t.Run("limited decrements", func(t *testing.T) {
		times := Times{Remaining: 3}
		times.Consume()
		if times.Remaining != 2 {
			t.Errorf("Remaining = %d, want 2", times.Remaining)
		}
	})

	t.Run("limited does not go below zero", func(t *testing.T) {
		times := Times{Remaining: 0}
		times.Consume()
		if times.Remaining != 0 {
			t.Errorf("Remaining = %d, want 0", times.Remaining)
		}
	})
}

// A zero-value Times (no "times" key in JSON) must default to unlimited so that
// expectations without an explicit times field never expire.
func TestTimes_DefaultUnlimitedWhenAbsent(t *testing.T) {
	var exp Expectation
	if err := json.Unmarshal([]byte(`{"request":{"method":"GET","path":"/x"},"response":{"status":200}}`), &exp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !exp.Times.Available() {
		t.Errorf("expectation without times field should be available (unlimited)")
	}
	exp.Times.Consume()
	if !exp.Times.Available() {
		t.Errorf("expectation without times field should stay available after consume")
	}
}
