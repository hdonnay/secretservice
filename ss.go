// +build linux

package ss

import (
	"fmt"
	dbus "github.com/guelfey/go.dbus"
)

const (
	// These constants are defined in the spec.
	//
	// They should be fixed across SecretService implementations
	ServiceName       = "org.freedesktop.secrets"
	ServicePath       = "/org/freedesktop/secrets"
	DefaultCollection = "/org/freedesktop/secrets/collection/default"
	CollectionPath    = "/org/freedesktop/secrets/collection"

	setProp = "org.freedesktop.DBus.Properties.Set"

	_Item = "org.freedesktop.Secret.Item"
	// Methods
	_ItemSetSecret = "org.freedesktop.Secret.Item.SetSecret"
	_ItemGetSecret = "org.freedesktop.Secret.Item.GetSecret"
	_ItemDelete    = "org.freedesktop.Secret.Item.Delete"
	// Properties
	_ItemLocked     = "org.freedesktop.Secret.Item.Locked"
	_ItemCreated    = "org.freedesktop.Secret.Item.Created"
	_ItemModified   = "org.freedesktop.Secret.Item.Modified"
	_ItemLabel      = "org.freedesktop.Secret.Item.Label"
	_ItemAttributes = "org.freedesktop.Secret.Item.Attributes"

	_Prompt = "org.freedesktop.Secret.Prompt"
	// Methods
	_PromptPrompt = "org.freedesktop.Secret.Prompt.Prompt"
	// Properties

	_Session = "org.freedesktop.Secret.Session"
	// Methods
	_SessionClose = "org.freedesktop.Secret.Session.Close"

	_Service = "org.freedesktop.Secret.Service"
	// Methods
	_ServiceOpenSession      = "org.freedesktop.Secret.Service.OpenSession"
	_ServiceCreateCollection = "org.freedesktop.Secret.Service.CreateCollection"
	_ServiceSearchItems      = "org.freedesktop.Secret.Service.SearchItems"
	_ServiceUnlock           = "org.freedesktop.Secret.Service.Unlock"
	_ServiceLock             = "org.freedesktop.Secret.Service.Lock"
	_ServiceGetSecrets       = "org.freedesktop.Secret.Service.GetSecrets"
	//Properties
	_ServiceAlias       = "org.freedesktop.Secret.Service.Alias"
	_ServiceCollections = "org.freedesktop.Secret.Service.Collections"

	_Collection = "org.freedesktop.Secret.Collection"
	// Methods
	_CollectionDelete      = "org.freedesktop.Secret.Collection.Delete"
	_CollectionSearchItems = "org.freedesktop.Secret.Collection.SearchItems"
	_CollectionCreateItem  = "org.freedesktop.Secret.Collection.CreateItem"
	// Properties
	_CollectionLabel    = "org.freedesktop.Secret.Collection.Label"
	_CollectionLocked   = "org.freedesktop.Secret.Collection.Locked"
	_CollectionCreated  = "org.freedesktop.Secret.Collection.Created"
	_CollectionModified = "org.freedesktop.Secret.Collection.Modified"
	_CollectionItems    = "org.freedesktop.Secret.Collection.Items"

	text_plain = "text/plain; charset=utf8"
)

var (
	UnknownContentType = fmt.Errorf("Content-Type is unknown for this Secret")
	InvalidAlgorithm   = fmt.Errorf("unknown algorithm")
)

type Object interface {
	Path() dbus.ObjectPath
}

// Secret as defined in the Spec
// Note: Order is important for marshalling
type Secret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string `dbus:"content_type"`
}

// Uses text/plain as the Content-type which may need to change in the future.
// Probably not, though.
func (s *Secret) SetSecret(secret string) error {
	s.Value = []byte(secret)
	return nil
}

// The return vaue could conceivably be an actual []byte...
// The ContentType should be able to be relied upon...
func (s *Secret) GetValue() string {
	return string(s.Value)
}

func DialService() (Service, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return Service{}, err
	}
	obj := conn.Object(ServiceName, ServicePath)
	return Service{obj}, nil
}

func DialCollection(path string) (Collection, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return Collection{}, err
	}
	obj := conn.Object(ServiceName, dbus.ObjectPath(path))
	return Collection{obj}, nil
}

/*
// ssProvider implements the provider interface freedesktop SecretService
type ssProvider struct {
	*dbus.Conn
	srv *dbus.Object
}

// This is used to open a seassion for every get/set. Alternative might be to
// defer() the call to close when constructing the ssProvider
func (s *ssProvider) openSession() (*dbus.Object, error) {
	var disregard dbus.Variant
	var sessionPath dbus.ObjectPath
	method := fmt.Sprint(ssServiceIface, "OpenSession")
	err := s.srv.Call(method, 0, "plain", dbus.MakeVariant("")).Store(&disregard, &sessionPath)
	if err != nil {
		return nil, err
	}
	return s.Object(ssServiceName, sessionPath), nil
}

// Unsure how the .Prompt call surfaces, it hasn't come up.
func (s *ssProvider) unlock(p dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	method := fmt.Sprint(ssServiceIface, "Unlock")
	err := s.srv.Call(method, 0, []dbus.ObjectPath{p}).Store(&unlocked, &prompt)
	if err != nil {
		return fmt.Errorf("keyring/dbus: Unlock error: %s", err)
	}
	if prompt != dbus.ObjectPath("/") {
		method = fmt.Sprint(ssPromptIface, "Prompt")
		call := s.Object(ssServiceName, prompt).Call(method, 0, "unlock")
		return call.Err
	}
	return nil
}

func (s *ssProvider) Get(collection, item string) (string, error) {
	results := []dbus.ObjectPath{}
	var secret Secret
	search := map[string]string{
		"username": u,
		"service":  c,
	}

	session, err := s.openSession()
	if err != nil {
		return "", err
	}
	defer session.Call(fmt.Sprint(ssSessionIface, "Close"), 0)
	s.unlock(ssCollectionPath)
	collection_ := s.Object(ssServiceName, ssCollectionPath)

	method := fmt.Sprint(ssCollectionIface, "SearchItems")
	call := collection_.Call(method, 0, search)
	err = call.Store(&results)
	if call.Err != nil {
		return "", call.Err
	}
	// results is a slice. Just grab the first one.
	if len(results) == 0 {
		return "", ErrNotFound
	}

	method = fmt.Sprint(ssItemIface, "GetSecret")
	err = s.Object(ssServiceName, results[0]).Call(method, 0, session.Path()).Store(&secret)
	if err != nil {
		return "", err
	}
	return string(secret.Value), nil
}

func (s *ssProvider) Set(c, u, p string) error {
	var item, prompt dbus.ObjectPath
	properties := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label": dbus.MakeVariant(fmt.Sprintf("%s - %s", u, c)),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(map[string]string{
			"username": u,
			"service":  c,
		}),
	}

	session, err := s.openSession()
	if err != nil {
		return err
	}
	defer session.Call(fmt.Sprint(ssSessionIface, "Close"), 0)
	s.unlock(ssCollectionPath)
	collection := s.Object(ssServiceName, ssCollectionPath)

	secret := newSSSecret(session.Path(), p)
	// the bool is "replace"
	err = collection.Call(fmt.Sprint(ssCollectionIface, "CreateItem"), 0, properties, secret, true).Store(&item, &prompt)
	if err != nil {
		return fmt.Errorf("keyring/dbus: CreateItem error: %s", err)
	}
	if prompt != "/" {
		s.Object(ssServiceName, prompt).Call(fmt.Sprint(ssPromptIface, "Prompt"), 0, "unlock")
	}
	return nil
}

func init() {
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ss/dbus: Error connecting to dbus session, not registering SecretService provider")
		return
	}
	srv := conn.Object(ssServiceName, ssServicePath)
	p := &ssProvider{conn, srv}

	// Everything should implement dbus peer, so ping to make sure we have an object...
	if session, err := p.openSession(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open dbus session %v: %v\n", srv, err)
		return
	}
	session.Call(fmt.Sprint(ssSessionIface, "Close"), 0)

	defaultProvider = p
}
*/
