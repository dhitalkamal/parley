package tui

import (
	"testing"
	"time"
)

func TestEscIsDoubleTap_TrueWithinWindow(t *testing.T) {
	last := time.Now()
	now := last.Add(200 * time.Millisecond)
	if !escIsDoubleTap(last, now, 600*time.Millisecond) {
		t.Error("expected a double-tap within the window")
	}
}

func TestEscIsDoubleTap_FalseOutsideWindow(t *testing.T) {
	last := time.Now()
	now := last.Add(700 * time.Millisecond)
	if escIsDoubleTap(last, now, 600*time.Millisecond) {
		t.Error("expected no double-tap outside the window")
	}
}

func TestEscIsDoubleTap_FalseWithNoPriorEsc(t *testing.T) {
	if escIsDoubleTap(time.Time{}, time.Now(), 600*time.Millisecond) {
		t.Error("expected no double-tap when there was no prior esc")
	}
}
