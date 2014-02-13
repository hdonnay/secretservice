// +build linux

package ss

import (
	"fmt"
	dbus "github.com/guelfey/go.dbus"
	"time"
)

type Prompt struct{ *dbus.Object }

// This runs the prompt.
func (p Prompt) Prompt(window_id string) error {
	// spec: Prompt(IN String window-id);
	return p.Call(_PromptPrompt, 0, window_id).Err
}

// Make a prompt go away.
func (p Prompt) Dismiss() error {
	// spec: Dismiss(void);
	return p.Call(_PromptDismiss, 0).Err
}

type Item struct{ *dbus.Object }

func (i Item) simpleCall(method string, args ...interface{}) error {
	var promptPath dbus.ObjectPath
	if len(args) == 0 {
		args = append(args, 0)
	}
	call := i.Call(fmt.Sprintf("%s.%s", _Item, method), 0, args...)
	if call.Err != nil {
		return call.Err
	}
	call.Store(&promptPath)
	return checkPrompt(promptPath)
}

// Use the passed Session to set the Secret in this Item
func (i Item) SetSecret(s Secret) error {
	// spec: SetSecret(IN Secret secret);
	return i.simpleCall("SetSecret", s)
}

// Use the passed Session to retrieve the Secret in this Item
func (i Item) GetSecret(s Session) (Secret, error) {
	// spec: GetSecret(IN ObjectPath session, OUT Secret secret);
	var ret Secret
	call := i.Call(_ItemGetSecret, 0, s.Path())
	if call.Err != nil {
		return ret, call.Err
	}
	call.Store(&ret)
	return ret, nil
}

// Any prompt should be handled transparently.
func (i Item) Delete() error {
	// spec: Delete (OUT ObjectPath Prompt);
	return i.simpleCall("Delete")
}
func (i Item) Locked() bool {
	v, _ := i.GetProperty(_ItemLocked)
	return v.Value().(bool)
}
func (i Item) Created() time.Time {
	v, _ := i.GetProperty(_ItemCreated)
	return time.Unix(v.Value().(int64), 0)
}
func (i Item) Modified() time.Time {
	v, _ := i.GetProperty(_ItemModified)
	return time.Unix(v.Value().(int64), 0)
}
func (i Item) GetAttributes() map[string]string {
	v, _ := i.GetProperty(_ItemAttributes)
	return v.Value().(map[string]string)
}
func (i Item) SetAttributes(attr map[string]string) error {
	return i.Call(setProp, 0, _Item, "Attributes", attr).Err
}
func (i Item) GetLabel() string {
	v, _ := i.GetProperty(_ItemLabel)
	return v.Value().(string)
}
func (i Item) SetLabel(l string) error {
	return i.Call(setProp, 0, _Item, "Label", l).Err
}

type Service struct{ *dbus.Object }

func (s Service) simpleCall(method string, args ...interface{}) error {
	var promptPath dbus.ObjectPath
	if len(args) == 0 {
		args = append(args, 0)
	}
	call := s.Call(method, 0, args...)
	if call.Err != nil {
		return call.Err
	}
	call.Store(&promptPath)
	return checkPrompt(promptPath)
}

// First argument is the algorithm used. Currently only "plain" is supported.
func (s Service) OpenSession(algo string, args ...interface{}) (Session, error) {
	// spec: OpenSession(IN String algorithm, IN Variant input, OUT Variant output, OUT ObjectPath result);
	var ret Session
	conn, err := dbus.SessionBus()
	if err != nil {
		return ret, err
	}
	switch algo {
	case "plain":
		var discard dbus.Variant
		var sessionPath dbus.ObjectPath
		err := s.Call(_ServiceOpenSession, 0, algo, dbus.MakeVariant("")).Store(&discard, &sessionPath)
		if err != nil {
			return ret, err
		}
		return Session{conn.Object(ServiceName, sessionPath)}, nil
	default:
		return ret, InvalidAlgorithm
	}
	return ret, nil
}

// The first argument is the Label for the collection, and the second is an (optional) alias.
func (s Service) CreateCollection(label, alias string) (Collection, error) {
	// spec: CreateCollection(IN Dict<String,Variant> properties, IN String alias, OUT ObjectPath collection, OUT ObjectPath prompt);
	var collectionPath, promptPath dbus.ObjectPath
	conn, err := dbus.SessionBus()
	if err != nil {
		return Collection{}, err
	}
	properties := map[string]dbus.Variant{
		_CollectionLabel: dbus.MakeVariant(label),
	}
	call := s.Call(_ServiceCreateCollection, 0, properties, alias)
	if call.Err != nil {
		return Collection{}, call.Err
	}
	err = call.Store(&collectionPath, &promptPath)
	if err != nil {
		return Collection{}, err
	}
	err = checkPrompt(promptPath)
	if err != nil {
		return Collection{}, err
	}
	return Collection{conn.Object(ServiceName, collectionPath)}, nil
}

func (s Service) SearchItems(attrs map[string]string) ([]Item, []Item, error) {
	// spec: SearchItems(IN Dict<String,String> attributes, OUT Array<ObjectPath> unlocked, OUT Array<ObjectPath> locked);
	var unlocked, locked []dbus.ObjectPath
	conn, err := dbus.SessionBus()
	if err != nil {
		return []Item{}, []Item{}, err
	}
	call := s.Call(_ServiceSearchItems, 0, attrs)
	err = call.Store(&unlocked, &locked)
	if err != nil {
		return []Item{}, []Item{}, err
	}
	retUnlocked := make([]Item, len(unlocked))
	retLocked := make([]Item, len(locked))
	for i, v := range unlocked {
		retUnlocked[i] = Item{conn.Object(ServiceName, v)}
	}
	for i, v := range locked {
		retLocked[i] = Item{conn.Object(ServiceName, v)}
	}
	return retUnlocked, retLocked, nil
}

// UNIMPLEMENTED
func (s Service) Unlock(o []Object) ([]Object, error) {
	// spec: Unlock(IN Array<ObjectPath> objects, OUT Array<ObjectPath> unlocked, OUT ObjectPath prompt);
	return nil, nil
}

// UNIMPLEMENTED
func (s Service) Lock(o []Object) ([]Object, error) {
	// spec: Lock(IN Array<ObjectPath> objects, OUT Array<ObjectPath> locked, OUT ObjectPath Prompt);
	return nil, nil
}

// The specified action is to return map[ObjectPath]Secret, but map[Label]Secret is much more useful.
func (s Service) GetSecrets(items []Item, ses Session) (map[dbus.ObjectPath]Secret, error) {
	// spec: GetSecrets(IN Array<ObjectPath> items, IN ObjectPath session, OUT Dict<ObjectPath,Secret> secrets);
	arg := make([]dbus.ObjectPath, len(items))
	for i, o := range items {
		arg[i] = o.Path()
	}
	call := s.Call(_ServiceGetSecrets, 0, arg, ses.Path())
	if call.Err != nil {
		return map[dbus.ObjectPath]Secret{}, call.Err
	}
	ret := make(map[dbus.ObjectPath]Secret)
	err := call.Store(&ret)
	return ret, err
}

func (s Service) ReadAlias(a string) (Collection, error) {
	// spec: ReadAlias(IN String name, OUT ObjectPath collection);
	conn, err := dbus.SessionBus()
	if err != nil {
		return Collection{}, call.Err
	}
	call := s.Call(_ServiceReadAlias, 0, a)
	if call.Err != nil {
		return Collection{}, call.Err
	}
	path := call.Value.(dbus.ObjectPath)
	return Collection{conn.Object(ServiceName, path)}, nil
}

func (s Service) SetAlias(a string, c Collection) error {
	// spec: SetAlias(IN String name, IN ObjectPath collection);
	return simpleCall("SetAlias", a, c.Path())
}

// List Colletions
func (s Service) Collections() ([]Collection, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return []Collection{}, err
	}
	v, _ := s.GetProperty(_ServiceCollections)
	paths := v.Value().([]dbus.ObjectPath)
	out := make([]Collection, len(paths))
	for i, path := range paths {
		out[i] = Collection{conn.Object(ServiceName, path)}
	}
	return out, nil
}

// type Collection implements the org.freedesktop.SecretService.Collection
// interface, using function calls for property accessors/setters
type Collection struct{ *dbus.Object }

func (c Collection) simpleCall(method string, args ...interface{}) error {
	var promptPath dbus.ObjectPath
	if len(args) == 0 {
		args = append(args, 0)
	}
	call := c.Call(fmt.Sprintf("%s.%s", _Collection, method), 0, args...)
	if call.Err != nil {
		return call.Err
	}
	call.Store(&promptPath)
	return checkPrompt(promptPath)
}

func (c Collection) Delete() error {
	// spec: Delete(OUT ObjectPath prompt);
	return c.simpleCall("Delete")
}

func (c Collection) SearchItems(attr map[string]string) ([]Item, error) {
	// spec: SearchItems(IN Dict<String,String> attributes, OUT Array<ObjectPath> results);
	i := []Item{}
	conn, err := dbus.SessionBus()
	if err != nil {
		return i, err
	}
	call := c.Call(_CollectionSearchItems, 0, attr)
	if call.Err != nil {
		return i, call.Err
	}
	var value []dbus.ObjectPath
	call.Store(&value)
	for _, objPath := range value {
		i = append(i, Item{conn.Object(ServiceName, objPath)})
	}
	return i, nil
}

func (c Collection) CreateItem(label string, attr map[string]string, s Secret, replace bool) (Item, error) {
	// spec: CreateItem(IN Dict<String,Variant> properties, IN Secret secret, IN Boolean replace, OUT ObjectPath item, OUT ObjectPath prompt);
	i := Item{}
	conn, err := dbus.SessionBus()
	if err != nil {
		return i, err
	}

	prop := make(map[string]dbus.Variant)
	prop[_ItemLabel] = dbus.MakeVariant(label)
	prop[_ItemAttributes] = dbus.MakeVariant(attr)

	call := c.Call(_CollectionCreateItem, 0, prop, s, replace)
	if call.Err != nil {
		return i, call.Err
	}
	var newItem dbus.ObjectPath
	call.Store(&newItem)

	i = Item{conn.Object(ServiceName, newItem)}

	return i, nil
}
func (c Collection) Locked() bool {
	v, _ := c.GetProperty(_CollectionLocked)
	return v.Value().(bool)
}
func (c Collection) Created() time.Time {
	v, _ := c.GetProperty(_CollectionCreated)
	return time.Unix(v.Value().(int64), 0)
}
func (c Collection) Modified() time.Time {
	v, _ := c.GetProperty(_CollectionModified)
	return time.Unix(v.Value().(int64), 0)
}
func (c Collection) Items() []Item {
	// How did we get here if we can't get on the bus now?
	conn, _ := dbus.SessionBus()
	p, _ := c.GetProperty(_CollectionItems)
	objs := p.Value().([]dbus.ObjectPath)
	i := make([]Item, len(objs))
	for idx, objPath := range objs {
		i[idx] = Item{conn.Object(ServiceName, objPath)}
	}
	return i
}
func (c Collection) GetLabel() string {
	v, _ := c.GetProperty(_CollectionLabel)
	return v.Value().(string)
}
func (c Collection) SetLabel(l string) error {
	return c.Call(setProp, 0, _Collection, "Label", l).Err
}

type Session struct{ *dbus.Object }

// Yes, really, it's the only method that exists on a Session.
func (s Session) Close() {
	// spec: Close(void);
	s.Go(_SessionClose, dbus.FlagNoReplyExpected, nil)
}
