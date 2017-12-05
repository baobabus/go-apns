// Copyright 2017 Aleksey Blinov. All rights reserved.

package syncx

import (
	"testing"
)

// Basic non-contention tests for Counter primitive.

func TestCounter(t *testing.T) {
	var subj Counter
	if subj != 0 {
		t.Fatalf("Bad zero value %v", subj)
	}
	subj.Add(1)
	if subj != 1 {
		t.Fatalf("Bad tick %v", subj)
	}
	subj.Add(9)
	if subj != 10 {
		t.Fatalf("Bad tock %v", subj)
	}
	subj.Add(1)
	i := subj.Draw()
	if subj != 0 || i != 11 {
		t.Fatalf("Bad draw %v %v %v", subj, i)
	}
}
