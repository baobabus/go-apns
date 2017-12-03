// Copyright 2017 Aleksey Blinov. All rights reserved.

package syncx

import (
	"testing"
)

// Basic non-contention tests for TickTock primitives.

func TestTickTockCounter(t *testing.T) {
	var subj TickTockCounter
	if subj != 0 {
		t.Fatalf("Bad zero value %v", subj)
	}
	subj.Tick()
	if subj != 0x0100000000 {
		t.Fatalf("Bad tick %v", subj)
	}
	subj.Tock()
	if subj != 0x0100000001 {
		t.Fatalf("Bad tock %v", subj)
	}
	subj.Tick()
	i, o := subj.Fold()
	if subj != 0x0100000000 || i != 2 || o != 1 {
		t.Fatalf("Bad fold %v %v %v", subj, i, o)
	}
}

func TestTickTockFolder(t *testing.T) {
	var subj TickTockFolder
	if subj != 0 {
		t.Fatalf("Bad zero value %v", subj)
	}
	subj.Tick()
	if subj != 1 {
		t.Fatalf("Bad tick %v", subj)
	}
	subj.Tock()
	if subj != 0x0100000000 {
		t.Fatalf("Bad tock %v", subj)
	}
	subj.Tick()
	if subj != 0x0100000001 {
		t.Fatalf("Bad second tick %v", subj)
	}
	subj.Tick()
	c, p := subj.Draw()
	if subj != 2 || c != 1 || p != 2 {
		t.Fatalf("Bad draw %v %v %v", subj, c, p)
	}
}

