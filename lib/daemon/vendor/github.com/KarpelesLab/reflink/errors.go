package reflink

import "errors"

// ErrReflinkUnsupported is returned by Always() if the reflink operation is not
// supported on the current operating system. This happens on non-Linux systems or
// when the necessary system calls aren't available.
//
// Auto() will never return this error since it falls back to regular copy methods.
var (
	// ErrReflinkUnsupported indicates that reflink operations are not supported on the current OS
	ErrReflinkUnsupported = errors.New("reflink is not supported on this OS")

	// ErrReflinkFailed indicates that the reflink operation failed because the specific 
	// filesystem or files don't support it. This can happen if the files are on different
	// filesystems or if the filesystem doesn't support reflinks (e.g., ext4).
	ErrReflinkFailed      = errors.New("reflink is not supported on this OS or file")
)
