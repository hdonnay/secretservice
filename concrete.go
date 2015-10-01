// +build linux

package ss

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/hkdf"
	dbus "github.com/guelfey/go.dbus"
	"github.com/monnand/dhkx"
)

type Prompt struct{ *dbus.Object }

// This runs the prompt.
//
// The prompt will timeout after 1 minute
func (p Prompt) Prompt(window_id string) (dbus.Variant, error) {
	// spec: Prompt(IN String window-id);
	empty := dbus.Variant{}
	conn, err := dbus.SessionBus()
	if err != nil {
		return empty, err
	}
	cmp := make(chan *dbus.Signal, 5)
	conn.Signal(cmp)
	call := p.Call(_PromptPrompt, 0, window_id)
	if call.Err != nil {
		return empty, call.Err
	}
	for {
		select {
		case sig := <-cmp:
			if sig.Name == _PromptCompleted {
				if sig.Body[0].(bool) {
					return empty, PromptDismissed
				}
				return sig.Body[1].(dbus.Variant), nil
			}
		case <-time.After(time.Duration(time.Minute)):
			err := p.Dismiss()
			if err != nil {
				panic(err)
			}
			return empty, Timeout
		}
	}
}

// Make a prompt go away.
func (p Prompt) Dismiss() error {
	// spec: Dismiss(void);
	return p.Call(_PromptDismiss, 0).Err
}

type Item struct{ *dbus.Object }

// Use the passed Session to set the Secret in this Item
func (i Item) SetSecret(s Secret) error {
	// spec: SetSecret(IN Secret secret);
	return simpleCall(i.Path(), _ItemSetSecret, s)
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
	return simpleCall(i.Path(), _ItemDelete)
}
func (i Item) Locked() bool {
	v, err := i.GetProperty(_ItemLocked)
	if err != nil {
		panic(err)
	}
	return v.Value().(bool)
}
func (i Item) Created() time.Time {
	v, err := i.GetProperty(_ItemCreated)
	if err != nil {
		panic(err)
	}
	return time.Unix(int64(v.Value().(uint64)), 0)
}
func (i Item) Modified() time.Time {
	v, err := i.GetProperty(_ItemModified)
	if err != nil {
		panic(err)
	}
	return time.Unix(int64(v.Value().(uint64)), 0)
}
func (i Item) GetAttributes() map[string]string {
	v, err := i.GetProperty(_ItemAttributes)
	if err != nil {
		panic(err)
	}
	return v.Value().(map[string]string)
}
func (i Item) SetAttributes(attr map[string]string) error {
	return i.Call(setProp, 0, _Item, "Attributes", attr).Err
}
func (i Item) GetLabel() string {
	v, err := i.GetProperty(_ItemLabel)
	if err != nil {
		panic(err)
	}
	return v.Value().(string)
}
func (i Item) SetLabel(l string) error {
	return i.Call(setProp, 0, _Item, "Label", l).Err
}

type Service struct{ *dbus.Object }

// First argument is the algorithm used. "plain" (AlgoPlain) and
// "dh-ietf1024-sha256-aes128-cbc-pkcs7" (AlgoDH) are supported.
//
// The dbus api has the caller supply their DH public key and returns
// the other side's public key, but this implementation generates a
// new keypair, does the exchange, derives the encryption key, and then
// stores it in the returned Session.
func (s Service) OpenSession(algo string, args ...interface{}) (Session, error) {
	// spec: OpenSession(IN String algorithm, IN Variant input, OUT Variant output, OUT ObjectPath result);
	var ret Session
	conn, err := dbus.SessionBus()
	if err != nil {
		return ret, err
	}
	switch algo {
	case AlgoPlain:
		var discard dbus.Variant
		var sessionPath dbus.ObjectPath
		err = s.Call(_ServiceOpenSession, 0, algo, dbus.MakeVariant("")).Store(&discard, &sessionPath)
		if err != nil {
			return ret, err
		}
		ret = Session{conn.Object(ServiceName, sessionPath), algo, nil}
	case AlgoDH:
		// see http://standards.freedesktop.org/secret-service/ch07s03.html
		var sessionPath dbus.ObjectPath
		var srvReply dbus.Variant
		var srvPub []byte
		symKey := make([]byte, aes.BlockSize)
		grp, err := dhkx.GetGroup(2)
		if err != nil {
			return ret, err
		}
		privKey, err := grp.GeneratePrivateKey(rand.Reader)
		if err != nil {
			return ret, err
		}
		err = s.Call(_ServiceOpenSession, 0, algo, dbus.MakeVariant(privKey.Bytes())).Store(&srvReply, &sessionPath)
		if err != nil {
			return ret, err
		}
		srvPub = srvReply.Value().([]byte)
		sharedKey, err := grp.ComputeKey(dhkx.NewPublicKey(srvPub), privKey)
		if err != nil {
			return ret, err
		}
		_, err = io.ReadFull(hkdf.New(sha256.New, sharedKey.Bytes(), nil, nil), symKey)
		ret = Session{conn.Object(ServiceName, sessionPath), algo, symKey}
	default:
		err = InvalidAlgorithm
	}
	return ret, err
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
	if dbus.ObjectPath("/") != collectionPath {
		return Collection{conn.Object(ServiceName, collectionPath)}, nil
	}
	v, err := checkPrompt(promptPath)
	if err != nil {
		return Collection{}, err
	}
	return Collection{conn.Object(ServiceName, dbus.ObjectPath(v.Value().(string)))},
		fmt.Errorf("unable to create collection")
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
func (s Service) Unlock(o []dbus.ObjectPath) ([]dbus.ObjectPath, error) {
	// spec: Unlock(IN Array<ObjectPath> objects, OUT Array<ObjectPath> unlocked, OUT ObjectPath prompt);
	var ret []dbus.ObjectPath
	var prompt dbus.ObjectPath
	call := s.Call(_ServiceUnlock, 0, o)
	if call.Err != nil {
		return []dbus.ObjectPath{}, call.Err
	}
	err := call.Store(&ret, &prompt)
	if err != nil {
		return []dbus.ObjectPath{}, call.Err
	}
	v, err := checkPrompt(prompt)
	if err != nil {
		return []dbus.ObjectPath{}, call.Err
	}
	for _, o := range v.Value().([]dbus.ObjectPath) {
		ret = append(ret, o)
	}
	return ret, nil
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
	var path dbus.ObjectPath
	conn, err := dbus.SessionBus()
	if err != nil {
		return Collection{}, err
	}
	call := s.Call(_ServiceReadAlias, 0, a)
	if call.Err != nil {
		return Collection{}, call.Err
	}
	err = call.Store(&path)
	if err != nil {
		return Collection{}, err
	}
	return Collection{conn.Object(ServiceName, path)}, nil
}

func (s Service) SetAlias(a string, c Collection) error {
	// spec: SetAlias(IN String name, IN ObjectPath collection);
	return simpleCall(s.Path(), _ServiceSetAlias, a, c.Path())
}

// List Colletions
func (s Service) Collections() []Collection {
	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	v, err := s.GetProperty(_ServiceCollections)
	if err != nil {
		panic(err)
	}
	paths := v.Value().([]dbus.ObjectPath)
	out := make([]Collection, len(paths))
	for i, path := range paths {
		out[i] = Collection{conn.Object(ServiceName, path)}
	}
	return out
}

// type Collection implements the org.freedesktop.SecretService.Collection
// interface, using function calls for property accessors/setters
type Collection struct{ *dbus.Object }

func (c Collection) Delete() error {
	// spec: Delete(OUT ObjectPath prompt);
	return simpleCall(c.Path(), _CollectionDelete)
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
	v, err := c.GetProperty(_CollectionLocked)
	if err != nil {
		panic(err)
	}
	l := v.Value()
	return l.(bool)
}
func (c Collection) Created() time.Time {
	v, err := c.GetProperty(_CollectionCreated)
	if err != nil {
		panic(err)
	}
	return time.Unix(int64(v.Value().(uint64)), 0).UTC()
}
func (c Collection) Modified() time.Time {
	v, err := c.GetProperty(_CollectionModified)
	if err != nil {
		panic(err)
	}
	return time.Unix(int64(v.Value().(uint64)), 0).UTC()
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
	v, err := c.GetProperty(_CollectionLabel)
	if err != nil {
		panic(err)
	}
	return v.Value().(string)
}
func (c Collection) SetLabel(l string) error {
	return c.Call(setProp, 0, _Collection, "Label", l).Err
}

func (c Collection) Unlock() error {
	conn, _ := dbus.SessionBus()
	var prompt dbus.ObjectPath
	u := make([]dbus.ObjectPath, 1)
	srv := conn.Object(ServiceName, ServicePath)
	call := srv.Call(_ServiceUnlock, 0, []dbus.ObjectPath{c.Path()})
	if call.Err != nil {
		return call.Err
	}
	if err := call.Store(&u, &prompt); err != nil {
		return err
	}
	if _, err := checkPrompt(prompt); err != nil {
		return err
	}
	return nil
}

type Session struct {
	*dbus.Object
	Algorithm string
	Key       []byte
}

// Yes, really, it's the only method that exists on a Session.
func (s Session) Close() {
	// spec: Close(void);
	s.Go(_SessionClose, dbus.FlagNoReplyExpected, nil)
}

func (s Session) NewSecret() Secret {
	r := Secret{s.Path(), nil, nil, text_plain}
	switch s.Algorithm {
	case AlgoDH:
		r.Parameters = make([]byte, aes.BlockSize)
		io.ReadFull(rand.Reader, r.Parameters)
	}
	return r
}
