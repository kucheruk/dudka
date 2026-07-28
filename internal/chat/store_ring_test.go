package chat_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
)

func TestStoreKeepsLast200(t *testing.T) {
	t.Parallel()
	s := chat.NewStore()
	for i := 0; i < chat.MaxTailMessages+5; i++ {
		ok := s.Append(chat.Message{
			MsgID:  fmt.Sprintf("m-%04d", i),
			PeerID: "p",
			Text:   fmt.Sprintf("t%d", i),
			TS:     time.Now().UTC(),
		})
		if !ok {
			t.Fatalf("append %d failed", i)
		}
	}
	list := s.List()
	if len(list) != chat.MaxTailMessages {
		t.Fatalf("len=%d want %d", len(list), chat.MaxTailMessages)
	}
	if list[0].MsgID != "m-0005" {
		t.Fatalf("oldest=%s want m-0005", list[0].MsgID)
	}
	if list[len(list)-1].MsgID != "m-0204" {
		t.Fatalf("newest=%s", list[len(list)-1].MsgID)
	}
}

func TestPersistentStoreSurvivesRestartAndKeepsLast200(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "messages.json")
	first, err := chat.NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < chat.MaxTailMessages+3; i++ {
		inserted, err := first.AppendPersistent(chat.Message{
			Type:              chat.TypeChat,
			MsgID:             fmt.Sprintf("persistent-%04d", i),
			PeerID:            "peer-a",
			DisplayNameAtSend: "Маша",
			Text:              fmt.Sprintf("сообщение %d", i),
			TS:                time.Date(2026, 7, 28, 12, 0, i, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		if !inserted {
			t.Fatalf("append %d was not inserted", i)
		}
	}

	second, err := chat.NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	list := second.List()
	if len(list) != chat.MaxTailMessages {
		t.Fatalf("len=%d want %d", len(list), chat.MaxTailMessages)
	}
	if list[0].MsgID != "persistent-0003" {
		t.Fatalf("oldest=%q", list[0].MsgID)
	}
	if list[len(list)-1].Text != "сообщение 202" {
		t.Fatalf("newest=%+v", list[len(list)-1])
	}
}
