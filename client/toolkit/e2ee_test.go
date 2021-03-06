/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except raw compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to raw writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package toolkit

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

// Test cases for all implementations.
// Because iv is random, so Encrypted data is not always the same,
// 	but Decrypt(possibleEncrypted) will get raw.
//  `raw` and `possibleEncrypted` are in hex
//  `pass` is raw string
var testCases = []struct {
	raw               string
	pass              string
	possibleEncrypted string
}{
	{
		raw:               "11",
		pass:              ";#K]As9C*6L",
		possibleEncrypted: "a372ea2c158a2f99d386e309db4355a659a7a8dd3986fd1d94f7604256061609",
	},
	{
		raw: "111282C128421286712857128C2128EF" +
			"128B7671283C128571287512830128EC" +
			"128391281A1312849128381281E1286A" +
			"12871128621287A9D12857128C412886" +
			"128FD12834128DA128F5",
		pass: "",
		possibleEncrypted: "1bfb6a7fda3e3eb1e14c9afd0baefe86" +
			"c90979101f179db7e48a0fa7617881e8" +
			"f752c59fb512bb86b8ed69c5644bf2dc" +
			"30fbcd3bf79fb20342595c84fad00e46" +
			"2fab3e51266492a3d5d085e650c1e619" +
			"6278d7f5185c263440ec6fd940ffbb85",
	},
	{
		raw:               "11",
		pass:              "'K]\"#'pi/1/JD2",
		possibleEncrypted: "a83d152777ce3a1c0710b03676ae867c86ab0a47b3ca080f825683ac1079eb41",
	},
	{
		raw:  "11111111111111111111111111111111",
		pass: "",
		possibleEncrypted: "7dda438c4256a63c62d6816617fcbf9c" +
			"7773b9b4f87902b7253848ba2b0ed0ba" +
			"f70a3ac976a835b7bc3008e9ba43da74",
	},
	{
		raw:  "11111111111111111111111111111111",
		pass: "youofdas1312",
		possibleEncrypted: "cab07967cf377dbc010fbf5f84d12bcb" +
			"6f8b188e6965738cf9007a671b4bfeb9" +
			"f52257aac3808048c341dcaa1c125ca7",
	},
	{
		raw:               "11111111111111111111111111",
		pass:              "??????Bottle????",
		possibleEncrypted: "4384874473945c5b70519ad5ace6305ef6b78c60c3c694add08a8b81899c4171",
	},
}

func TestEncryptDecryptCases(t *testing.T) {
	defaultLevel := log.GetLevel()
	log.SetLevel(log.DebugLevel)
	defer log.SetLevel(defaultLevel)
	Convey("encrypt & decrypt cases", t, func() {
		for i, c := range testCases {
			in, _ := hex.DecodeString(c.raw)
			pass := []byte(c.pass)
			out, _ := hex.DecodeString(c.possibleEncrypted)
			log.Infof("TestEncryptDecryptCases: %d", i)
			enc, err := Encrypt(in, pass)
			log.Debugf("Enc: %x", enc)
			So(err, ShouldBeNil)
			dec1, err := Decrypt(enc, pass)
			if !bytes.Equal(dec1, in) {
				t.Errorf("\nExpected:\n%x\nActual:\n%x\n", in, dec1)
			}
			dec2, err := Decrypt(out, pass)
			So(err, ShouldBeNil)
			if !bytes.Equal(dec2, in) {
				t.Errorf("\nExpected:\n%x\nActual:\n%x\n", in, dec2)
			}
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	var password = "CovenantSQL.io"
	Convey("encrypt & decrypt 0 length string with aes128", t, func() {
		enc, err := Encrypt([]byte(""), []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 0 length bytes with aes128", t, func() {
		enc, err := Encrypt([]byte(nil), []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 1 byte with aes128", t, func() {
		enc, err := Encrypt([]byte{0x11}, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, []byte{0x11})
		So(len(dec), ShouldEqual, 1)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 1747 length bytes", t, func() {
		in := bytes.Repeat([]byte{0xff}, 1747)
		enc, err := Encrypt(in, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, (1747/aes.BlockSize+2)*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, in)
		So(len(dec), ShouldEqual, 1747)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 32 length bytes", t, func() {
		in := bytes.Repeat([]byte{0xcc}, 32)
		enc, err := Encrypt(in, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, (32/aes.BlockSize+2)*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, in)
		So(len(dec), ShouldEqual, 32)
		So(err, ShouldBeNil)
	})

	Convey("decrypt error length bytes", t, func() {
		in := bytes.Repeat([]byte{0xaa}, 1747)
		dec, err := Decrypt(in, []byte(password))
		So(dec, ShouldBeNil)
		So(err.Error(), ShouldEqual, "cipher data size not match")
	})
}
