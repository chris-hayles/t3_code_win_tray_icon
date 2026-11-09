package pairlink

import (
	"net/url"
	"strings"
	"testing"
)

func TestParsePairingURL_WithFragment(t *testing.T) {
	out := "noise\nPairing URL: http://127.0.0.1:3773/pair#token=ABCDEF\n"
	u, err := ParsePairingURLLine(out)
	if err != nil {
		t.Fatal(err)
	}
	if u != "http://127.0.0.1:3773/pair#token=ABCDEF" {
		t.Fatalf("got %q", u)
	}
	link, err := BuildAppLink(u)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link, "t3code://pair?") {
		t.Fatalf("link %q", link)
	}
	q, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	pu := q.Query().Get("pairingUrl")
	if pu != u {
		t.Fatalf("decoded %q want %q", pu, u)
	}
	if !strings.Contains(link, "%23token%3D") {
		t.Fatalf("fragment not encoded in query: %q", link)
	}
}

func TestParsePairingURL_RejectMissing(t *testing.T) {
	_, err := ParsePairingURLLine("hello\n")
	if err != ErrNoPairingURL {
		t.Fatalf("got %v", err)
	}
}

func TestBuildAppLink_ExampleShape(t *testing.T) {
	u := "http://127.0.0.1:3773/pair#token=ABCDEF"
	link, _ := BuildAppLink(u)
	wantSub := "pairingUrl=http%3A%2F%2F127.0.0.1%3A3773%2Fpair%23token%3DABCDEF"
	if !strings.Contains(link, wantSub) {
		t.Fatalf("got %q want contains %q", link, wantSub)
	}
}
