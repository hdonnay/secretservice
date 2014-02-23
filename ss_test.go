package ss

import "testing"

func TestDialService(t *testing.T) {
	empty := Service{}
	s, err := DialService()
	if err != nil {
		t.Fatal(err)
	}
	if s == empty {
		t.Fatal("empty Service returned")
	}
}

func TestDialCollection(t *testing.T) {
	empty := Collection{}
	c, err := DialCollection(DefaultCollection)
	if err != nil {
		t.Fatal(err)
	}
	if c == empty {
		t.Fatal("empty Collection returned")
	}
}
