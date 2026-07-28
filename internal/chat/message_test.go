package chat_test

import (
	"testing"
	"time"

	"dudka/internal/chat"
)

func TestEncodeDecodeChatMessage(t *testing.T) {
	t.Parallel()
	in := chat.Message{
		MsgID:             "11111111-1111-4111-8111-111111111111",
		PeerID:            "peer-a",
		DisplayNameAtSend: "Аня",
		TS:                time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
		Text:              "привет из комнаты",
	}
	raw, err := chat.EncodeMessage(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := chat.DecodeMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != "chat" {
		t.Fatalf("type=%q", out.Type)
	}
	if out.MsgID != in.MsgID || out.PeerID != in.PeerID || out.DisplayNameAtSend != in.DisplayNameAtSend || out.Text != in.Text {
		t.Fatalf("got %+v want %+v", out, in)
	}
	if !out.TS.Equal(in.TS) {
		t.Fatalf("ts got %v want %v", out.TS, in.TS)
	}
}

func TestStoreDedupeByMsgID(t *testing.T) {
	t.Parallel()
	s := chat.NewStore()
	m := chat.Message{MsgID: "m1", PeerID: "p", Text: "hi", TS: time.Now().UTC()}
	if !s.Append(m) {
		t.Fatal("first append should insert")
	}
	if s.Append(m) {
		t.Fatal("duplicate should not insert")
	}
	if len(s.List()) != 1 {
		t.Fatalf("len=%d", len(s.List()))
	}
}
