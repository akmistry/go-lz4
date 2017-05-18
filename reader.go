package lz4

/*
#cgo CFLAGS: -O2 -I${SRCDIR}/c-lz4/lib

#include "lz4frame.h"
*/
import "C"

import (
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

const compressedBufSize = 4096

type Reader struct {
	r    io.Reader
	dctx *C.LZ4F_dctx

	buf      []byte
	bufIndex int
}

func NewReader(r io.Reader) (*Reader, error) {
	lr := &Reader{r: r, buf: make([]byte, 0, compressedBufSize)}
	lz4Err := C.LZ4F_createDecompressionContext(&lr.dctx, C.LZ4F_VERSION)
	if C.LZ4F_isError(lz4Err) != 0 {
		return nil, fmt.Errorf("lz4 error: %s", C.GoString(C.LZ4F_getErrorName(lz4Err)))
	}
	runtime.SetFinalizer(lr, func(r *Reader) {
		if r.dctx == nil {
			return
		}
		C.LZ4F_freeDecompressionContext(r.dctx)
	})
	return lr, nil
}

func (r *Reader) fill() error {
	if r.bufIndex < len(r.buf) {
		return nil
	}
	r.bufIndex = 0
	r.buf = r.buf[:0]
	n, err := r.r.Read(r.buf[:cap(r.buf)])
	if n > 0 {
		r.buf = r.buf[:n]
		if err == io.EOF {
			// Only return EOF after reading 0 bytes.
			return nil
		}
	}
	return err
}

func (r *Reader) Read(buf []byte) (int, error) {
	for {
		err := r.fill()
		if err != nil {
			return 0, err
		}

		dstSize := C.size_t(len(buf))
		srcSize := C.size_t(len(r.buf) - r.bufIndex)
		n := C.LZ4F_decompress(r.dctx, unsafe.Pointer(&buf[0]), &dstSize,
			unsafe.Pointer(&(r.buf[r.bufIndex])), &srcSize, nil)
		if C.LZ4F_isError(C.LZ4F_errorCode_t(n)) != 0 {
			return 0, fmt.Errorf("lz4 decompress error: %s",
				C.GoString(C.LZ4F_getErrorName(C.LZ4F_errorCode_t(n))))
		}
		// TODO: Use |n| to increase buffer size.
		r.bufIndex += int(srcSize)
		if dstSize == 0 {
			// No bytes decoded. Probably ran out of data and buffer needs a refill.
			continue
		}
		return int(dstSize), nil
	}
}

func (r *Reader) Close() error {
	// TODO: Pay attention to this error code.
	C.LZ4F_freeDecompressionContext(r.dctx)
	r.dctx = nil
	r.buf = nil
	return nil
}
