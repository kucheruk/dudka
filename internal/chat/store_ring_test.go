package chat_test

import (
	"fmt"
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
