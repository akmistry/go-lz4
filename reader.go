package lz4

// #include "lz4frame.h"
import "C"

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"unsafe"
)

const compressedBufSize = 64 * 1024

var readerBufferPool = sync.Pool{New: func() interface{} { return make([]byte, compressedBufSize) }}

type Reader struct {
	r    io.Reader
	dctx *C.LZ4F_dctx

	buf      []byte
	bufIndex int
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, buf: readerBufferPool.Get().([]byte)[:0]}
}

func (r *Reader) init() error {
	if r.dctx != nil {
		return nil
	}

	lz4Err := C.LZ4F_createDecompressionContext(&r.dctx, C.LZ4F_VERSION)
	if C.LZ4F_isError(lz4Err) != 0 {
		return fmt.Errorf("lz4 error: %s", C.GoString(C.LZ4F_getErrorName(lz4Err)))
	}
	runtime.SetFinalizer(r, func(r *Reader) {
		if r.dctx == nil {
			return
		}
		C.LZ4F_freeDecompressionContext(r.dctx)
	})
	return nil
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
	if err := r.init(); err != nil {
		return 0, err
	}

	for {
		fillErr := r.fill()
		if fillErr != nil && fillErr != io.EOF {
			return 0, fillErr
		}

		dstSize := C.size_t(len(buf))
		srcSize := C.size_t(len(r.buf) - r.bufIndex)
		srcBuf := unsafe.Pointer(nil)
		if srcSize > 0 {
			srcBuf = unsafe.Pointer(&(r.buf[r.bufIndex]))
		}
		n := C.LZ4F_decompress(r.dctx, unsafe.Pointer(&buf[0]), &dstSize,
			srcBuf, &srcSize, nil)
		if C.LZ4F_isError(C.LZ4F_errorCode_t(n)) != 0 {
			return 0, fmt.Errorf("lz4 decompress error: %s",
				C.GoString(C.LZ4F_getErrorName(C.LZ4F_errorCode_t(n))))
		}
		// TODO: Use |n| to increase buffer size.
		r.bufIndex += int(srcSize)
		if dstSize == 0 {
			// No bytes decoded. Probably ran out of data and buffer needs a refill.
			if fillErr == io.EOF {
				return 0, io.EOF
			}
			continue
		}
		return int(dstSize), nil
	}
}

func (r *Reader) Close() error {
	if r.dctx == nil {
		return nil
	}

	// TODO: Pay attention to this error code.
	C.LZ4F_freeDecompressionContext(r.dctx)
	r.dctx = nil
	readerBufferPool.Put(r.buf)
	r.buf = nil
	return nil
}
