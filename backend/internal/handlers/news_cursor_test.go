package handlers

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewsCursorRoundTrip(t *testing.T) {
	enc := encodeNewsCursor("2026-03-15", 42)
	cur, ok := decodeNewsCursor(enc)
	if !ok {
		t.Fatalf("decode failed for %q", enc)
	}
	if cur.Date != "2026-03-15" || cur.ID != 42 {
		t.Fatalf("got %+v", cur)
	}
}

func TestPasswordOK(t *testing.T) {
	if !passwordOK("secret", "secret") {
		t.Fatal("plain compare failed")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	phpStyle := "$2y$" + string(hash)[4:]
	if !passwordOK("secret", phpStyle) {
		t.Fatal("bcrypt $2y$ compare failed")
	}
	if passwordOK("wrong", phpStyle) {
		t.Fatal("expected mismatch")
	}
}
