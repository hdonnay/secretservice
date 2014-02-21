// +build linux

package ss

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	dbus "github.com/guelfey/go.dbus"
	"github.com/vgorin/cryptogo/pad"
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
	_PromptPrompt  = "org.freedesktop.Secret.Prompt.Prompt"
	_PromptDismiss = "org.freedesktop.Secret.Prompt.Dismiss"
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
	_ServiceReadAlias        = "org.freedesktop.Secret.Service.ReadAlias"
	_ServiceSetAlias         = "org.freedesktop.Secret.Service.SetAlias"
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

	AlgoPlain = "plain"
	AlgoDH    = "dh-ietf1024-sha256-aes128-cbc-pkcs7"

	text_plain = "text/plain; charset=utf8"
)

var (
	UnknownContentType = fmt.Errorf("Content-Type is unknown for this Secret")
	InvalidAlgorithm   = fmt.Errorf("unknown algorithm")
	InvalidSession     = fmt.Errorf("invalid session object")
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
func (s *Secret) SetSecret(session Session, secret []byte) error {
	switch session.Algorithm {
	case AlgoPlain:
		s.Value = secret
	case AlgoDH:
		block, err := aes.NewCipher(session.Key)
		if err != nil {
			return err
		}
		enc := cipher.NewCBCEncrypter(block, s.Parameters)
		ciphertext := pad.PKCS7Pad(secret, aes.BlockSize)
		s.Value = make([]byte, len(ciphertext))
		enc.CryptBlocks(s.Value, ciphertext)
	default:
		return InvalidSession
	}
	return nil
}

// This method is specific to the bindings
func (s *Secret) GetSecret(session Session) ([]byte, error) {
	switch session.Algorithm {
	case AlgoPlain:
		return s.Value, nil
	case AlgoDH:
		paddedPlaintext := make([]byte, len(s.Value))
		block, err := aes.NewCipher(session.Key)
		if err != nil {
			return []byte{}, err
		}
		dec := cipher.NewCBCDecrypter(block, s.Parameters)
		dec.CryptBlocks(paddedPlaintext, s.Value)
		return pad.PKCS7Unpad(paddedPlaintext)
	default:
		return []byte{}, InvalidSession
	}
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
