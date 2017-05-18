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

func TestReader(t *testing.T) {
	originalBuf, err := ioutil.ReadFile(testTar)
	if err != nil {
		t.Fatal(err)
	}
	compressedBuf, err := ioutil.ReadFile(testTarLz4)
	if err != nil {
		t.Fatal(err)
	}

	r, err := NewReader(bytes.NewReader(compressedBuf))
	if err != nil {
		t.Fatal(err)
	}
	decompressedBuf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if bytes.Compare(originalBuf, decompressedBuf) != 0 {
		t.Errorf("decompressed buf (len = %d) != original (len = %d)",
			len(decompressedBuf), len(originalBuf))
	}
}
