package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket_AllowsUpToCapacity(t *testing.T) {
	tb := NewTokenBucket(3, 1)

	for i := 0; i < 3; i++ {
		if !tb.Allow() {
			t.Fatal("token bucket allowed at least once (bucket should start full)", i+1)
		}
	}

	if tb.Allow() {
		t.Fatal("expected 4th request to be denied, bucket should be empty")
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	tb := NewTokenBucket(1, 10)

	if !tb.Allow() {
		t.Fatal("expected first request to be allowed")
	}
	if tb.Allow() {
		t.Fatal("expected second immediate request to be denied")
	}

	time.Sleep(150 * time.Millisecond)

	if !tb.Allow() {
		t.Fatal("expected request to be allowed after refill window")
	}
}
