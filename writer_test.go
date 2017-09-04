package lz4

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	written, err := w.Write(originalData)
	if err != nil {
		t.Fatal(err)
	} else if written != len(originalData) {
		t.Fatalf("written %d != len(originalData) %d", written, len(originalData))
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Original size %d, compressed size %d", len(originalData), buf.Len())

	r := NewReader(&buf)
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

func TestWriterByteAtATime(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for i := range originalData {
		written, err := w.Write(originalData[i : i+1])
		if err != nil {
			t.Fatal(err)
		} else if written != 1 {
			t.Errorf("written %d != 1", written)
		}
	}
	err := w.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Original size %d, compressed size %d", len(originalData), buf.Len())

	r := NewReader(&buf)
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
