package ss

import (
	"testing"

	dbus "github.com/guelfey/go.dbus"
)

var conn *dbus.Conn

func getConn() *dbus.Conn {
	if conn == nil {
		conn, _ = dbus.SessionBus()
	}
	return conn
}
func TestService_OpenSession(t *testing.T) {
	srv, err := DialService()
	if err != nil {
		t.Fatal(err)
	}

	_, err = srv.OpenSession(AlgoPlain)
	if err != nil {
		t.Fatal(err)
	}
	_, err = srv.OpenSession(AlgoDH)
	if err != nil {
		t.Fatal(err)
	}
	_, err = srv.OpenSession("garbage")
	if err != InvalidAlgorithm {
		t.Fatal("session 'garbage' returned nil error")
	}
}
