package ss

import (
	"testing"

	dbus "github.com/guelfey/go.dbus"
)

var (
	conn        *dbus.Conn
	totalSecret = []byte("you'llneverGuessme!")
	plainAttrs  = map[string]string{"test": "plain"}
	cryptAttrs  = map[string]string{"test": "crypt"}
)

func getConn() *dbus.Conn {
	if conn == nil {
		conn, _ = dbus.SessionBus()
	}
	return conn
}

func TestBasicWorkflow(t *testing.T) {
	var err error
	//conn := getConn()
	service, err := DialService()
	if err != nil {
		t.Error(err)
	}
	plain, err := service.OpenSession(AlgoPlain)
	if err != nil {
		t.Error(err)
	}
	crypt, err := service.OpenSession(AlgoDH)
	if err != nil {
		t.Error(err)
	}
	testCollection, err := DialCollection(DefaultCollection)
	if err != nil {
		t.Error(err)
	}
	t.Logf("collection:%v\tcreated:%v\tmodified:%v\tlocked:%v\n",
		testCollection.GetLabel(), testCollection.Created(),
		testCollection.Modified(), testCollection.Locked())

	if testCollection.Locked() {
		err := testCollection.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	}

	// this bit is kind of thorny.
	// maybe make a "dbusSecret" type that gets serialized and we can work with
	// "Secret"s?
	secPlain := plain.NewSecret()
	secCrypt := crypt.NewSecret()
	err = secPlain.SetValue(plain, totalSecret)
	if err != nil {
		t.Error(err)
	}
	err = secCrypt.SetValue(crypt, totalSecret)
	if err != nil {
		t.Error(err)
	}
	_, err = testCollection.CreateItem("test-plain", plainAttrs, secPlain, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testCollection.CreateItem("test-crypt", cryptAttrs, secCrypt, true)
	if err != nil {
		t.Fatal(err)
	}

	for _, a := range []map[string]string{plainAttrs, cryptAttrs} {
		items, err := testCollection.SearchItems(a)
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range items {
			var err error
			t.Logf("item:%s\tcreated:%v\tmodified:%v\tlocked:%v\tattrs:%s\n",
				i.GetLabel(), i.Created(), i.Modified(), i.Locked(), i.GetAttributes())
			s, err := i.GetSecret(plain)
			if err != nil {
				t.Error(err)
			}
			x, err := s.GetValue(plain)
			if err != nil {
				t.Error(err)
			}
			t.Logf("\tsecret: %s\n", string(x))
			/*
				if err := i.Delete(); err != nil {
					t.Error(err)
				}
				t.Log("secret deleted")
			*/
		}
	}
	for _, s := range []Session{plain, crypt} {
		s.Close()
	}
}

// Service Tests

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

func TestService_CreateCollection(t *testing.T) {
	dismiss := true
	srv, err := DialService()
	if err != nil {
		t.Fatal(err)
	}
	_, err = srv.OpenSession(AlgoPlain)
	if err != nil {
		t.Fatal(err)
	}
	_, err = srv.CreateCollection("test", "")
	switch err {
	case PromptDismissed:
		fallthrough
	case Timeout:
		// Okay to timeout or dismiss, at least until we write a custom
		// Prompt.Prompt() function for testing.
		t.Log(err)
	case nil:
		dismiss = false
	default:
		t.Fatal(err)
	}

	if !dismiss {
		for _, c := range srv.Collections() {
			t.Logf("collection: %s\n", c.GetLabel())
			if c.GetLabel() == "test" {
				return
			}
		}
		t.Error("unable to find collection")
	}
}

func TestService_SearchItems(t *testing.T) {
	srv, err := DialService()
	if err != nil {
		t.Fatal(err)
	}
	session, err := srv.OpenSession(AlgoPlain)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []map[string]string{plainAttrs, cryptAttrs} {
		unlocked, locked, err := srv.SearchItems(a)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(locked)
		for _, i := range unlocked {
			var err error
			t.Logf("item:%s\tcreated:%v\tmodified:%v\tlocked:%v\tattrs:%s\n",
				i.GetLabel(), i.Created(), i.Modified(), i.Locked(), i.GetAttributes())
			s, err := i.GetSecret(session)
			if err != nil {
				t.Error(err)
			}
			x, err := s.GetValue(session)
			if err != nil {
				t.Error(err)
			}
			t.Logf("\tsecret: %s\n", string(x))
		}
	}
}
