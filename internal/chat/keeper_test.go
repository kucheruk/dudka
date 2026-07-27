package chat_test

import (
	"testing"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

func TestSelectTailKeeperMinPeerID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ids  []string
		want string
		ok   bool
	}{
		{name: "empty", ids: nil, want: "", ok: false},
		{name: "single", ids: []string{"peer-b"}, want: "peer-b", ok: true},
		{name: "unordered", ids: []string{"peer-c", "peer-a", "peer-b"}, want: "peer-a", ok: true},
		{name: "uuid-lexicographic", ids: []string{
			"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
			"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			"cccccccc-cccc-4ccc-8ccc-cccccccccccc",
		}, want: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ok: true},
		{name: "skips empty", ids: []string{"", "z", "m", ""}, want: "m", ok: true},
		{name: "duplicates", ids: []string{"b", "a", "a", "c"}, want: "a", ok: true},
		{name: "already sorted", ids: []string{"a", "b", "c"}, want: "a", ok: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := chat.SelectTailKeeper(tc.ids)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("SelectTailKeeper(%v)=(%q,%v) want (%q,%v)", tc.ids, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestSelectTailKeeperAmongSelfAndPeers(t *testing.T) {
	t.Parallel()
	peers := discovery.NewPeerStore()
	_ = peers.Upsert(discovery.Peer{PeerID: "peer-z", DisplayName: "Z"})
	_ = peers.Upsert(discovery.Peer{PeerID: "peer-m", DisplayName: "M"})
	got, ok := chat.SelectTailKeeperAmong("peer-self", peers.List())
	if !ok || got != "peer-m" {
		// min among {peer-self, peer-z, peer-m} is peer-m
		t.Fatalf("got (%q,%v)", got, ok)
	}
	got, ok = chat.SelectTailKeeperAmong("aaa-self", peers.List())
	if !ok || got != "aaa-self" {
		t.Fatalf("self should win when smallest: (%q,%v)", got, ok)
	}
}

func TestSelectTailKeeperStable(t *testing.T) {
	t.Parallel()
	ids := []string{"c", "a", "b"}
	first, ok1 := chat.SelectTailKeeper(ids)
	second, ok2 := chat.SelectTailKeeper([]string{"b", "c", "a"})
	if !ok1 || !ok2 || first != second || first != "a" {
		t.Fatalf("unstable: %q/%q", first, second)
	}
}
