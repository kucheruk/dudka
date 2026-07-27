package tui

import (
	"strings"
	"testing"
)

func TestRenderProtoMismatchAndPortNote(t *testing.T) {
	out := Render(Snapshot{
		EngineOK: true, MeName: "Вася", Network: NetworkOK,
		ProtoMajor: 1, Incompatible: 1,
		AnnouncePort: 41778, SessionPort: 41779,
		PortRelocated: true, PortNote: "порт 41777 занят — слушаем announce=41778 session=41779",
	})
	if !strings.Contains(out, ProtoMismatchCopy) {
		t.Fatalf("missing proto copy: %s", out)
	}
	if !strings.Contains(out, "порт 41777 занят") {
		t.Fatalf("missing port note: %s", out)
	}
}
