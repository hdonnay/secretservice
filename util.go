// +build linux

package ss

import (
	dbus "github.com/guelfey/go.dbus"
)

var (
	noPrompt = dbus.ObjectPath("/")
)

func checkPrompt(promptPath dbus.ObjectPath) error {
	if promptPath == noPrompt {
		return nil
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	pr := Prompt{conn.Object(ServiceName, promptPath)}
	// I have no idea what this argument is or how to use it.
	err = pr.Prompt("secretservice.go")
	return err
}

//// Introspect the object and return it casted to the proper interface
//func Coerce(o Object) (interface{}, error) {
//	return nil, nil
//}
