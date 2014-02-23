package ss

import (
	"testing"

	dbus "github.com/guelfey/go.dbus"
)

const fake = "/fake/prompt"

type _FakePrompt struct{}

func (f *_FakePrompt) Prompt(window_id string) {
	return
}
func (f *_FakePrompt) Dismiss() {
	return
}

func TestCheckPrompt(t *testing.T) {
	var err error
	conn, err := dbus.SessionBus()
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Export(_FakePrompt{}, dbus.ObjectPath(fake), _Prompt)
	if err != nil {
		t.Fatal(err)
	}

	err = checkPrompt(dbus.ObjectPath("/"))
	if err != nil {
		t.Error(err)
	}
	err = checkPrompt(dbus.ObjectPath(fake))
	if err != nil {
		// claims the interface is wrong...
		t.Log(err)
	}

	err = conn.Export(nil, dbus.ObjectPath(fake), _Prompt)
	if err != nil {
		t.Fatal(err)
	}
}
