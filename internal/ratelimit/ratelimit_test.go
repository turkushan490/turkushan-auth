package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	clock := time.Unix(1_000_000, 0)
	l := New(3, time.Minute)
	l.now = func() time.Time { return clock }

	for i := 0; i < 3; i++ {
		if !l.Allow("ip1") {
			t.Fatalf("hit %d blocked", i+1)
		}
	}
	if l.Allow("ip1") {
		t.Fatal("4th hit allowed")
	}
	if !l.Allow("ip2") {
		t.Fatal("other key blocked")
	}

	clock = clock.Add(time.Minute)
	if !l.Allow("ip1") {
		t.Fatal("not reset after the window")
	}
}
