package reflink

import (
	"errors"
	"io"
)

// sectionWriter is a helper used when we need to fallback into copying data manually.
// It implements io.Writer and io.Seeker interfaces to work with standard Go I/O functions.
// This helper makes it possible to write to a specific section of a file when reflink and
// copy_file_range aren't available.
type sectionWriter struct {
	w    io.WriterAt // target file
	base int64       // base position in file
	off  int64       // current relative offset
}

// Write implements the io.Writer interface. It writes data to the underlying file
// at the current offset position and updates the internal offset.
func (s *sectionWriter) Write(p []byte) (int, error) {
	n, err := s.w.WriteAt(p, s.base+s.off)
	s.off += int64(n)
	return n, err
}

// Seek implements the io.Seeker interface. It changes the offset of the section writer
// relative to the base offset. This allows seeking within a section of a file.
func (s *sectionWriter) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		// nothing needed
	case io.SeekCurrent:
		offset += s.off
	case io.SeekEnd:
		// we don't support io.SeekEnd
		fallthrough
	default:
		return s.off, errors.New("Seek: invalid whence")
	}
	if offset < 0 {
		return s.off, errors.New("Seek: invalid offset")
	}
	s.off = offset
	return offset, nil
}
