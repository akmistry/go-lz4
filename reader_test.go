package lz4

import (
	"bytes"
	"io/ioutil"
	"testing"
)

const (
	testTar    = "lz4.tar"
	testTarLz4 = "lz4.tar.lz4"
)

var (
	originalData   []byte
	compressedData []byte
)

func init() {
	var err error
	originalData, err = ioutil.ReadFile(testTar)
	if err != nil {
		panic(err)
	}
	compressedData, err = ioutil.ReadFile(testTarLz4)
	if err != nil {
		panic(err)
	}
}

func TestReader(t *testing.T) {
	r := NewReader(bytes.NewReader(compressedData))
	decompressedBuf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if bytes.Compare(originalData, decompressedBuf) != 0 {
		t.Errorf("decompressed buf (len = %d) != original (len = %d)",
			len(decompressedBuf), len(originalData))
	}
}

func TestReaderByteAtATime(t *testing.T) {
	r := NewReader(bytes.NewReader(compressedData))
	defer r.Close()
	for i, b := range originalData {
		var buf [1]byte
		n, err := r.Read(buf[:])
		if err != nil {
			t.Errorf("i: %d, err %v != nil", i, err)
		} else if n != 1 {
			t.Errorf("i: %d, n %d != 1", i, n)
		}
		if buf[0] != b {
			t.Errorf("i: %d, decompressed %d != original %d", i, buf[0], b)
		}
	}
}
