/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package crypto

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

// Test 1: Encryption and decryption.
func TestCipheringBasic(t *testing.T) {
	privkey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		t.Fatal("failed to generate private key")
	}

	in := []byte("Hey there dude. How are you doing? This is a test.")

	out, err := EncryptAndSign(privkey.PubKey(), in)
	if err != nil {
		t.Fatal("failed to encrypt:", err)
	}

	dec, err := DecryptAndCheck(privkey, out)
	if err != nil {
		t.Fatal("failed to decrypt:", err)
	}

	if !bytes.Equal(in, dec) {
		t.Error("decrypted data doesn't match original")
	}
}

func TestCipheringErrors(t *testing.T) {
	privkey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		t.Fatal("failed to generate private key")
	}

	tests1 := []struct {
		ciphertext []byte // input ciphertext
	}{
		{bytes.Repeat([]byte{0x00}, 133)},                   // errInputTooShort
		{bytes.Repeat([]byte{0x00}, 134)},                   // errUnsupportedCurve
		{bytes.Repeat([]byte{0x02, 0xCA}, 134)},             // errInvalidXLength
		{bytes.Repeat([]byte{0x02, 0xCA, 0x00, 0x20}, 134)}, // errInvalidYLength
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // IV
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x02, 0xCA, 0x00, 0x20, // curve and X length
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // X
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x20, // Y length
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Y
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ciphertext
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MAC
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}}, // invalid pubkey
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // IV
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x02, 0xCA, 0x00, 0x20, // curve and X length
			0x11, 0x5C, 0x42, 0xE7, 0x57, 0xB2, 0xEF, 0xB7, // X
			0x67, 0x1C, 0x57, 0x85, 0x30, 0xEC, 0x19, 0x1A,
			0x13, 0x59, 0x38, 0x1E, 0x6A, 0x71, 0x12, 0x7A,
			0x9D, 0x37, 0xC4, 0x86, 0xFD, 0x30, 0xDA, 0xE5,
			0x00, 0x20, // Y length
			0x7E, 0x76, 0xDC, 0x58, 0xF6, 0x93, 0xBD, 0x7E, // Y
			0x70, 0x10, 0x35, 0x8C, 0xE6, 0xB1, 0x65, 0xE4,
			0x83, 0xA2, 0x92, 0x10, 0x10, 0xDB, 0x67, 0xAC,
			0x11, 0xB1, 0xB5, 0x1B, 0x65, 0x19, 0x53, 0xD2,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ciphertext
			// padding not aligned to 16 bytes
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MAC
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}}, // errInvalidPadding
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // IV
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x02, 0xCA, 0x00, 0x20, // curve and X length
			0x11, 0x5C, 0x42, 0xE7, 0x57, 0xB2, 0xEF, 0xB7, // X
			0x67, 0x1C, 0x57, 0x85, 0x30, 0xEC, 0x19, 0x1A,
			0x13, 0x59, 0x38, 0x1E, 0x6A, 0x71, 0x12, 0x7A,
			0x9D, 0x37, 0xC4, 0x86, 0xFD, 0x30, 0xDA, 0xE5,
			0x00, 0x20, // Y length
			0x7E, 0x76, 0xDC, 0x58, 0xF6, 0x93, 0xBD, 0x7E, // Y
			0x70, 0x10, 0x35, 0x8C, 0xE6, 0xB1, 0x65, 0xE4,
			0x83, 0xA2, 0x92, 0x10, 0x10, 0xDB, 0x67, 0xAC,
			0x11, 0xB1, 0xB5, 0x1B, 0x65, 0x19, 0x53, 0xD2,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // ciphertext
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MAC
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}}, // ErrInvalidMAC
	}

	for i, test := range tests1 {
		_, err = DecryptAndCheck(privkey, test.ciphertext)
		if err == nil {
			t.Errorf("decrypt #%d did not get error", i)
		}
	}

	// test error from removePKCSPadding
	tests2 := []struct {
		in []byte // input data
	}{
		{bytes.Repeat([]byte{0x11}, 17)},
		{bytes.Repeat([]byte{0x07}, 15)},
	}
	for i, test := range tests2 {
		_, err = RemovePKCSPadding(test.in)
		if err == nil {
			t.Errorf("removePKCSPadding #%d did not get error", i)
		}
	}
}

func TestAddPKCSPadding(t *testing.T) {
	Convey("padding", t, func() {
		data := []byte("xxxxxx")
		padData := AddPKCSPadding(data)
		cleanData, err := RemovePKCSPadding(padData)

		So(cleanData, ShouldResemble, data)
		So(err, ShouldBeNil)
	})
	Convey("non-padding", t, func() {
		data := []byte("xxxxxxxxxxxxxxxx")
		padData := AddPKCSPadding(data)
		log.Infof("len %d after pkcs#7: %d", len(data), len(padData))
		cleanData, err := RemovePKCSPadding(padData)

		So(cleanData, ShouldResemble, data)
		So(err, ShouldBeNil)
	})

}
