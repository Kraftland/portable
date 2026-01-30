[![GoDoc](https://godoc.org/github.com/KarpelesLab/reflink?status.svg)](https://godoc.org/github.com/KarpelesLab/reflink)

# reflink

A Go library to perform efficient file copies using reflink operations on compatible filesystems (btrfs or xfs).

## What is a reflink?

There are several types of links in file systems:

* symlinks (symbolic links) - pointer to another file path
* hardlinks - multiple directory entries pointing to the same inode/data
* reflinks - copy-on-write links where files initially share the same data blocks

Reflinks are a modern file system feature found in btrfs and xfs that act similar to hard links but with a key difference: modifying one file doesn't affect the other. Only the modified portions will consume additional disk space (copy-on-write). This makes reflinks ideal for efficient file copies, snapshots, and deduplication.

## Benefits of reflinks

* **Space efficiency**: Initial reflink copies use virtually no additional disk space
* **Performance**: Creating a reflink is nearly instantaneous, even for very large files
* **Safety**: Changes to one file don't affect the other file's contents
* **Transparency**: Applications don't need to be aware they're using reflinked files

## Compatibility

A system needs a compatible OS and filesystem to perform reflinks. Currently supported:

* btrfs on Linux
* xfs on Linux (with reflink=1 mount option)

Other operating systems have similar features that may be implemented in future versions:

* Windows has `DUPLICATE_EXTENTS_TO_FILE`
* Solaris has `reflink`
* MacOS has `clonefile`

## Usage

### Basic usage

```golang
// Creates a reflink copy or fails if reflink is not supported
err := reflink.Always("original_file.bin", "snapshot-001.bin")

// Creates a reflink copy if supported, falls back to regular copy if not
err := reflink.Auto("source_img.png", "modified_img.png")
```

### Working with file handles

```golang
src, _ := os.Open("source.dat")
defer src.Close()
dst, _ := os.Create("dest.dat")
defer dst.Close()

// Reflink the entire file
err := reflink.Reflink(dst, src, true) // true enables fallback to regular copy

// Reflink just a portion of the file
// Copy 1MB from offset 2MB in source to offset 0 in destination
err := reflink.Partial(dst, src, 0, 2*1024*1024, 1*1024*1024, true)
```

## Error handling

The library defines these specific errors:

* `ErrReflinkUnsupported` - Returned when reflink is not supported on this OS
* `ErrReflinkFailed` - Returned when reflink is not supported on the specific filesystem or files

Use standard Go error handling to check for these:

```golang
err := reflink.Always(src, dst)
if err != nil {
    if errors.Is(err, reflink.ErrReflinkUnsupported) {
        // OS doesn't support reflinks
    } else if errors.Is(err, reflink.ErrReflinkFailed) {
        // These specific files or filesystem don't support reflinks
    } else {
        // Other error (permissions, file not found, etc.)
    }
}
```

## Fallback mechanism

When using `Auto()` or setting `fallback=true` in other functions, the library tries these methods in order:
1. FICLONE ioctl (real reflink)
2. copy_file_range syscall (efficient kernel-space copy)
3. io.Copy (regular userspace copy)

## Requirements

* Source and destination files must be on the same filesystem
* The filesystem must support reflinks (btrfs, xfs with reflink=1)
* Appropriate file permissions are needed

## Notes

* The arguments are in the same order as `os.Link` or `os.Rename` (src, dst) rather than `io.Copy` (dst, src) as we are dealing with filenames
* For optimal performance with large files, always try to use reflinks instead of regular copies
* Reflinks are transparent to applications - a reflinked file behaves exactly like a regular file
