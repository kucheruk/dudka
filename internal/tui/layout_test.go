package tui_test

import (
	"testing"

	"dudka/internal/tui"
)

func TestLayoutForPanelsFit(t *testing.T) {
	t.Parallel()
	lay := tui.LayoutFor(80, 24)
	if lay.StatusH+lay.BodyH+lay.ComposeH+lay.HelpH != lay.Height {
		t.Fatalf("vertical sum %d want %d", lay.StatusH+lay.BodyH+lay.ComposeH+lay.HelpH, lay.Height)
	}
	if lay.PeersW+1+lay.FeedW != lay.Width {
		t.Fatalf("horizontal peers+sep+feed=%d want %d", lay.PeersW+1+lay.FeedW, lay.Width)
	}
	if lay.PeersW < 10 || lay.FeedW < 20 {
		t.Fatalf("panels too narrow: peers=%d feed=%d", lay.PeersW, lay.FeedW)
	}
}

func TestLayoutForTinyTerminal(t *testing.T) {
	t.Parallel()
	lay := tui.LayoutFor(20, 5)
	if lay.Width < 40 {
		// LayoutFor bumps minimum width
		t.Fatalf("width=%d want >=40", lay.Width)
	}
	if lay.BodyH < 3 {
		t.Fatalf("body=%d", lay.BodyH)
	}
}
