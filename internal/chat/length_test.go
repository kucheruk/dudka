package chat_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

func TestSendRejectsOverMaxCodePoints(t *testing.T) {
	t.Parallel()
	hub := chat.NewHub(chat.Config{
		PeerID: "p",
		Name:   "N",
		Store:  chat.NewStore(),
		Peers:  discovery.NewPeerStore(),
	})
	oversized := strings.Repeat("я", chat.MaxTextCodePoints+1) // multi-byte runes
	if utf8.RuneCountInString(oversized) != chat.MaxTextCodePoints+1 {
		t.Fatalf("fixture rune count=%d", utf8.RuneCountInString(oversized))
	}
	_, err := hub.Send(oversized)
	if err == nil {
		t.Fatal("expected error for oversized text")
	}
	if !errors.Is(err, chat.ErrTextTooLong) {
		t.Fatalf("err=%v want ErrTextTooLong", err)
	}
	if !strings.Contains(err.Error(), "4000") {
		t.Fatalf("error should mention limit: %v", err)
	}
	if len(hub.Messages()) != 0 {
		t.Fatalf("oversized must not be stored: %+v", hub.Messages())
	}
}

func TestSendAcceptsExactMaxCodePoints(t *testing.T) {
	t.Parallel()
	hub := chat.NewHub(chat.Config{
		PeerID: "p",
		Name:   "N",
		Store:  chat.NewStore(),
		Peers:  discovery.NewPeerStore(),
	})
	exact := strings.Repeat("x", chat.MaxTextCodePoints)
	msg, err := hub.Send(exact)
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(msg.Text) != chat.MaxTextCodePoints {
		t.Fatalf("len=%d", utf8.RuneCountInString(msg.Text))
	}
}
