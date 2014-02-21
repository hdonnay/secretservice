package ss

/*
This uses a static iv and known key to check that the crypto functions are still
doing somthing approximating correct.

*/

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"testing"
)

var (
	iv = bytes.Repeat([]byte{0xFF}, aes.BlockSize)

	plainSession = Session{nil, AlgoPlain, nil}
	plainTest    = []struct {
		in  []byte
		out Secret
	}{
		{
			[]byte("test"),
			Secret{"/", []byte{}, []byte("test"), text_plain}},
		{
			[]byte("another_test"),
			Secret{"/", []byte{}, []byte("another_test"), text_plain}},
	}

	cryptSession = Session{nil, AlgoDH, bytes.Repeat([]byte{44}, aes.BlockSize)}
	cryptTest    = []struct {
		in  []byte
		out Secret
	}{
		{
			[]byte("test"),
			Secret{"/", nil, []byte{28, 239, 162, 148, 204, 54, 150, 230, 172, 209, 199, 69, 41, 133, 67, 155}, text_plain}},
		{
			[]byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Donec a diam lectus. Sed sit amet ipsum mauris. Maecenas congue ligula ac quam viverra nec consectetur ante hendrerit.
Donec et mollis dolor. Praesent et diam eget libero egestas mattis sit amet vitae augue.`),
			Secret{"/", nil, []byte{113, 134, 102, 206, 86, 139, 254, 38, 60, 8, 194,
				189, 38, 136, 166, 129, 54, 164, 167, 94, 163, 82, 252, 249, 172, 46, 13,
				236, 23, 215, 250, 97, 42, 62, 122, 127, 163, 53, 142, 17, 1, 112, 51, 237,
				2, 219, 0, 38, 233, 156, 101, 71, 146, 229, 111, 141, 56, 43, 215, 19, 83,
				18, 123, 2, 35, 45, 89, 23, 9, 182, 192, 250, 226, 145, 98, 113, 37, 16,
				53, 167, 168, 42, 181, 230, 106, 22, 59, 65, 139, 240, 56, 149, 107, 152,
				177, 29, 247, 149, 37, 208, 10, 232, 108, 132, 4, 65, 169, 61, 108, 189, 8,
				50, 33, 170, 128, 69, 159, 23, 84, 182, 139, 223, 99, 75, 111, 54, 53, 122,
				33, 7, 204, 109, 246, 223, 3, 255, 48, 250, 68, 183, 58, 70, 129, 201, 166,
				59, 143, 85, 115, 122, 177, 215, 120, 74, 62, 156, 160, 173, 218, 9, 120,
				5, 206, 145, 226, 113, 225, 198, 23, 108, 10, 105, 22, 79, 181, 167, 154,
				104, 194, 169, 173, 107, 216, 186, 241, 120, 29, 140, 140, 36, 14, 119,
				226, 229, 107, 10, 117, 238, 142, 87, 146, 44, 147, 224, 73, 121, 150, 9,
				243, 85, 144, 178, 238, 1, 142, 25, 47, 8, 120, 163, 169, 206, 151, 4, 251,
				34, 31, 16, 237, 135, 154, 159, 68, 9, 102, 53, 248, 135, 118, 167, 137,
				71, 209, 153, 223, 221, 137, 168, 231, 154, 70, 60, 164, 238, 253, 226,
				157, 53, 161, 221, 69, 103, 35, 173, 206, 227, 178, 119, 235, 193, 242, 216},
				text_plain}},
	}
)

func TestPlainEncrypt(t *testing.T) {
	for _, p := range plainTest {
		var s Secret
		err := s.SetSecret(plainSession, p.in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(s.Value, p.out.Value) {
			t.Errorf("%s become %s not %s", p.in, s.Value, p.out.Value)
		}
		t.Logf("%s -> %s", s.Value, p.out.Value)
	}
}

func TestPlainDecrypt(t *testing.T) {
	for _, p := range plainTest {
		s, err := p.out.GetSecret(plainSession)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(s, p.in) {
			t.Errorf("%s become %s not %s", p.out.Value, s, p.in)
		}
		t.Logf("%s -> %s", s, p.in)
	}
}

func TestDHEncrypt(t *testing.T) {
	for _, p := range cryptTest {
		var s Secret
		s.Parameters = iv
		err := s.SetSecret(cryptSession, p.in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(s.Value, p.out.Value) {
			t.Errorf("%s become %v not %v",
				p.in,
				s.Value,
				p.out.Value)
		}
		t.Logf("%s -> %s",
			p.in,
			hex.EncodeToString(p.out.Value))
	}
}

func TestDHDecrypt(t *testing.T) {
	for _, p := range cryptTest {
		p.out.Parameters = iv
		s, err := p.out.GetSecret(cryptSession)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(s, p.in) {
			t.Errorf("%s become %v not %v",
				p.out.Value,
				s,
				p.in)
		}
		t.Logf("%s -> %s",
			hex.EncodeToString(s),
			p.in)
	}
}
