package lz4

// #include "lz4frame.h"
import "C"

import (
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

const writeBlockSize = 64 * 1024

type Writer struct {
	w     io.Writer
	cctx  *C.LZ4F_cctx
	prefs C.LZ4F_preferences_t

	bufSize  C.size_t
	buf      []byte
	bufIndex int
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w, prefs: C.LZ4F_preferences_t{
		frameInfo: C.LZ4F_frameInfo_t{
			blockSizeID: C.LZ4F_max64KB,
			blockMode:   C.LZ4F_blockLinked,
		},
	}}
}

func (w *Writer) init() error {
	if w.cctx != nil {
		return nil
	}

	lz4Err := C.LZ4F_createCompressionContext(&w.cctx, C.LZ4F_VERSION)
	if C.LZ4F_isError(lz4Err) != 0 {
		return fmt.Errorf("lz4 error: %s", C.GoString(C.LZ4F_getErrorName(lz4Err)))
	}
	runtime.SetFinalizer(w, func(w *Writer) {
		if w.cctx == nil {
			return
		}
		C.LZ4F_freeCompressionContext(w.cctx)
	})
	w.bufSize = C.LZ4F_compressBound(writeBlockSize, &w.prefs)
	w.buf = make([]byte, int(w.bufSize))

	n := C.LZ4F_compressBegin(w.cctx, unsafe.Pointer(&w.buf[0]), w.bufSize, &w.prefs)
	ec := C.LZ4F_errorCode_t(n)
	if C.LZ4F_isError(ec) != 0 {
		return fmt.Errorf("LZ4F_compressBegin error: %s", C.GoString(C.LZ4F_getErrorName(ec)))
	}
	_, err := w.w.Write(w.buf[:int(n)])
	return err
}

func (w *Writer) Write(buf []byte) (int, error) {
	if err := w.init(); err != nil {
		return 0, err
	}

	written := 0
	for len(buf) > 0 {
		writeSize := len(buf)
		if writeSize > writeBlockSize {
			writeSize = writeBlockSize
		}

		n := C.LZ4F_compressUpdate(w.cctx, unsafe.Pointer(&w.buf[0]), w.bufSize,
			unsafe.Pointer(&buf[0]), C.size_t(writeSize), nil)
		if C.LZ4F_isError(C.LZ4F_errorCode_t(n)) != 0 {
			return written, fmt.Errorf("lz4 compress error: %s",
				C.GoString(C.LZ4F_getErrorName(C.LZ4F_errorCode_t(n))))
		}

		if n > 0 {
			_, err := w.w.Write(w.buf[:int(n)])
			if err != nil {
				return written, err
			}
		}

		written += writeSize
		buf = buf[writeSize:]
	}
	return written, nil
}

func (w *Writer) Close() error {
	if w.cctx == nil {
		return nil
	}

	n := C.LZ4F_compressEnd(w.cctx, unsafe.Pointer(&w.buf[0]), w.bufSize, nil)
	if C.LZ4F_isError(C.LZ4F_errorCode_t(n)) != 0 {
		return fmt.Errorf("lz4 compress end error: %s",
			C.GoString(C.LZ4F_getErrorName(C.LZ4F_errorCode_t(n))))
	}
	_, err := w.w.Write(w.buf[:int(n)])

	// TODO: Pay attention to this error code.
	C.LZ4F_freeCompressionContext(w.cctx)
	w.cctx = nil
	w.buf = nil
	return err
}
