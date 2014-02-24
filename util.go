// +build linux

package ss

import dbus "github.com/guelfey/go.dbus"

var (
	noPrompt = dbus.ObjectPath("/")
)

func checkPrompt(promptPath dbus.ObjectPath) (dbus.Variant, error) {
	// if we don't need to prompt, just return.
	empty := dbus.Variant{}
	if promptPath == noPrompt {
		return empty, nil
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		return empty, err
	}
	pr := Prompt{conn.Object(ServiceName, promptPath)}
	return pr.Prompt("secretservice.go")
}

func simpleCall(path dbus.ObjectPath, method string, args ...interface{}) error {
	var call *dbus.Call
	var promptPath dbus.ObjectPath
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	obj := conn.Object(ServiceName, path)
	if args == nil {
		call = obj.Call(method, 0)
	} else {
		call = obj.Call(method, 0, args...)
	}
	if call.Err != nil {
		return call.Err
	}
	call.Store(&promptPath)
	_, err = checkPrompt(promptPath)
	return err
}

//// Introspect the object and return it casted to the proper interface
//func Coerce(o Object) (interface{}, error) {
//	return nil, nil
//}
