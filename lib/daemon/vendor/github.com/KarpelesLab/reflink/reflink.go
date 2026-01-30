//go:build !linux

package reflink

import "os"

// reflinkInternal attempts to reflink the entire contents of source file s to destination file d.
// On non-Linux systems, this function always returns ErrReflinkUnsupported.
func reflinkInternal(d, s *os.File) error {
	return ErrReflinkUnsupported
}

// reflinkRangeInternal attempts to reflink a specific range of data from source file to destination file.
// On non-Linux systems, this function always returns ErrReflinkUnsupported.
func reflinkRangeInternal(dst, src *os.File, dstOffset, srcOffset, n int64) error {
	return ErrReflinkUnsupported
}

// copyFileRange is a fallback mechanism that tries to use the copy_file_range syscall.
// On non-Linux systems, this function always returns ErrReflinkUnsupported.
func copyFileRange(dst, src *os.File, dstOffset, srcOffset, n int64) (int64, error) {
	return 0, ErrReflinkUnsupported
}
