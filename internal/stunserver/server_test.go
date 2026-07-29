package stunserver

import (
	"net"
	"testing"
	"time"

	"github.com/pion/stun/v3"
)

func TestBindingReturnsObservedUDPAddress(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = Serve(server) }()

	client, err := net.DialUDP("udp4", nil, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest, stun.Fingerprint)
	if _, err := client.Write(request.Raw); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1500)
	count, err := client.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	response := &stun.Message{Raw: append([]byte(nil), buffer[:count]...)}
	if err := response.Decode(); err != nil {
		t.Fatal(err)
	}
	var mapped stun.XORMappedAddress
	if err := mapped.GetFrom(response); err != nil {
		t.Fatal(err)
	}
	local := client.LocalAddr().(*net.UDPAddr)
	if !mapped.IP.Equal(local.IP) || mapped.Port != local.Port {
		t.Fatalf("mapped=%s local=%s", mapped.String(), local.String())
	}
}
