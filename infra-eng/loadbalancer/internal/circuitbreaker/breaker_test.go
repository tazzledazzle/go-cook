package circuitbreaker

import (
	"testing"
	"time"
)

func TestBreaker_TripsAfterThreshold(t *testing.T) {
	b := New(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("call %d: expected Allow() true while closed", i+1)
		}
		b.RecordFailure()
	}

	if b.CurrentState() != Open {
		t.Fatalf("expected state Open after %d failures, got %s", 3, b.CurrentState())
	}

	if b.Allow() {
		t.Fatal("expected Allow() false immediately after tripping open")
	}
}

func TestBreaker_HalfOpenAllowsOneTrial(t *testing.T) {
	b := New(1, 50*time.Millisecond)

	b.Allow()
	b.RecordFailure() // trips open after 1 failure

	if b.CurrentState() != Open {
		t.Fatal("expected Open after single failure with threshold 1")
	}

	time.Sleep(60 * time.Millisecond) // wait past resetTimeout

	if !b.Allow() {
		t.Fatal("expected first Allow() after timeout to succeed (half-open trial)")
	}
	if b.CurrentState() != HalfOpen {
		t.Fatalf("expected state HalfOpen, got %s", b.CurrentState())
	}

	if b.Allow() {
		t.Fatal("expected second concurrent Allow() during half-open to be rejected")
	}
}

func TestBreaker_HalfOpenSuccessCloses(t *testing.T) {
	b := New(1, 50*time.Millisecond)

	b.Allow()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)

	b.Allow() // the trial request
	b.RecordSuccess()

	if b.CurrentState() != Closed {
		t.Fatalf("expected Closed after successful trial, got %s", b.CurrentState())
	}
	if !b.Allow() {
		t.Fatal("expected Allow() true after closing")
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	b := New(1, 50*time.Millisecond)

	b.Allow()
	b.RecordFailure()
	time.Sleep(60 * time.Millisecond)

	b.Allow() // the trial request
	b.RecordFailure()

	if b.CurrentState() != Open {
		t.Fatalf("expected Open after failed trial, got %s", b.CurrentState())
	}
}
