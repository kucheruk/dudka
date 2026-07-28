package tui

// Layout describes fixed panel geometry for the interactive TUI (P046).
type Layout struct {
	Width     int
	Height    int
	StatusH   int
	PeersW    int
	FeedW     int
	BodyH     int
	ComposeH  int
	NoticeH   int
	HelpH     int
	FeedLines int
	PeerLines int
}

// LayoutFor computes panel sizes for a terminal of width×height.
// Topology matches DESIGN.md Linux TUI: status | peers+feed | compose.
func LayoutFor(width, height int) Layout {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}
	l := Layout{
		Width:    width,
		Height:   height,
		StatusH:  1,
		ComposeH: 1,
		NoticeH:  1,
		HelpH:    1,
	}
	l.BodyH = height - l.StatusH - l.ComposeH - l.NoticeH - l.HelpH
	if l.BodyH < 3 {
		l.BodyH = 3
	}
	l.PeersW = width / 4
	if l.PeersW < 14 {
		l.PeersW = 14
	}
	if l.PeersW > 28 {
		l.PeersW = 28
	}
	if l.PeersW >= width-20 {
		l.PeersW = width / 3
	}
	// 1 column for vertical separator between peers and feed
	l.FeedW = width - l.PeersW - 1
	if l.FeedW < 20 {
		l.FeedW = 20
		l.PeersW = width - l.FeedW - 1
	}
	if l.PeersW < 10 {
		l.PeersW = 10
		l.FeedW = width - l.PeersW - 1
	}
	// one label row inside each pane
	l.PeerLines = l.BodyH - 1
	l.FeedLines = l.BodyH - 1
	if l.PeerLines < 1 {
		l.PeerLines = 1
	}
	if l.FeedLines < 1 {
		l.FeedLines = 1
	}
	return l
}
