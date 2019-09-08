package shutil

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type CopyOptions struct {
	FollowSymlinks bool
}

// Copy data from src to dst.
//
// If follow_symlinks is not set and src is a symbolic link, a new
// symlink will be created instead of copying the file it points to.
func CopyFile(src, dst string, options *CopyOptions) (string, error) {

	followSymlinks := true
	if options != nil {
		followSymlinks = options.FollowSymlinks
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	if srcInfo.Mode()&os.ModeNamedPipe == os.ModeNamedPipe {
		return "", &SpecialFileError{src}
	}

	dstInfo, err := os.Stat(dst)
	if err == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return "", &SameFileError{src, dst}
		}
		if dstInfo.Mode()&os.ModeNamedPipe == os.ModeNamedPipe {
			return "", &SpecialFileError{dst}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if !followSymlinks && ((srcInfo.Mode() & os.ModeSymlink) == os.ModeSymlink) {
		srcOrigin, err := os.Readlink(src)
		if err != nil {
			return "", err
		}

		err = os.Symlink(srcOrigin, dst)
		if err != nil {
			return "", err
		}
	} else {
		fsrc, err := os.Open(src)
		if err != nil {
			return "", err
		}
		defer fsrc.Close()

		fdst, err := os.Create(dst)
		if err != nil {
			return "", err
		}
		defer fdst.Close()

		size, err := io.Copy(fdst, fsrc)
		if err != nil {
			return "", err
		}

		if size != srcInfo.Size() {
			log.Printf("%s: %d/%d copied", src, size, srcInfo.Size())
			return "", &CopyNotCompleteError{src, dst}
		}
	}

	return dst, nil
}

// Copy all stat info (mode bits, atime, mtime, flags) from src to dst.
//
// If the optional flag `follow_symlinks` is not set, symlinks aren't followed if and
// only if both `src` and `dst` are symlinks.
func CopyStat(src, dst string, options *CopyOptions) error {
	// TODO
	return nil
}

// Copy data and all stat info ("cp -p src dst"). Return the file's
// destination."
//
// The destination may be a directory.
//
// If follow_symlinks is false, symlinks won't be followed. This
// resembles GNU's "cp -P src dst".
func Copy2(src, dst string, options *CopyOptions) (string, error) {
	// log.Println("copy2", "from", src, "to", dst)
	followSymlinks := true
	if options != nil {
		followSymlinks = options.FollowSymlinks
	}

	dstInfo, err := os.Stat(dst)
	if err == nil {
		if dstInfo.IsDir() {
			dst = filepath.Join(dst, filepath.Base(src))
		}
	}

	_, err = CopyFile(src, dst, &CopyOptions{FollowSymlinks: followSymlinks})
	if err != nil {
		return "", err
	}

	err = CopyStat(src, dst, &CopyOptions{FollowSymlinks: followSymlinks})
	if err != nil {
		return "", err
	}

	return dst, nil
}

type CopyTreeOptions struct {
	Symlinks               bool
	Ignore                 func(string, []os.FileInfo) []string
	CopyFunction           func(string, string, *CopyOptions) (string, error)
	IgnoreDanglingSymlinks bool
}

// Recursively copy a directory tree.
//
// The destination directory must not already exist.
// If exception(s) occur, an Error is raised with a list of reasons.
//
// If the optional symlinks flag is true, symbolic links in the
// source tree result in symbolic links in the destination tree; if
// it is false, the contents of the files pointed to by symbolic
// links are copied. If the file pointed by the symlink doesn't
// exist, an exception will be added in the list of errors raised in
// an Error exception at the end of the copy process.
//
// You can set the optional ignore_dangling_symlinks flag to true if you
// want to silence this exception. Notice that this has no effect on
// platforms that don't support os.symlink.
//
// The optional ignore argument is a callable. If given, it
// is called with the `src` parameter, which is the directory
// being visited by copytree(), and `names` which is the list of
// `src` contents, as returned by os.listdir():
//
//     callable(src, names) -> ignored_names
//
// Since copytree() is called recursively, the callable will be
// called once for each directory that is copied. It returns a
// list of names relative to the `src` directory that should
// not be copied.
//
// The optional copy_function argument is a callable that will be used
// to copy each file. It will be called with the source path and the
// destination path as arguments. By default, copy2() is used, but any
// function that supports the same signature (like copy()) can be used.
func CopyTree(src, dst string, options *CopyTreeOptions) (string, error) {
	// log.Println("copy tree", "from", src, "to", dst)
	copyFunction := Copy2
	if options != nil && options.CopyFunction != nil {
		copyFunction = options.CopyFunction
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	subs, err := ioutil.ReadDir(src)
	if err != nil {
		return "", err
	}

	ignoredNames := []string{}
	if options != nil && options.Ignore != nil {
		ignoredNames = options.Ignore(src, subs)
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return "", err
	}

	for _, sub := range subs {
		isIgnored := false
		for _, ignoredName := range ignoredNames {
			if sub.Name() == ignoredName {
				isIgnored = true
				break
			}
		}

		if isIgnored {
			continue
		}

		subSrc := filepath.Join(src, sub.Name())
		subDst := filepath.Join(dst, sub.Name())

		subSrcInfo, err := os.Lstat(subSrc)
		if err != nil {
			return "", err
		}

		if (subSrcInfo.Mode() & os.ModeSymlink) == os.ModeSymlink {
			// TODO
		} else if subSrcInfo.IsDir() {
			_, err = CopyTree(subSrc, subDst, options)
			if err != nil {
				return "", err
			}
		} else {
			_, err = copyFunction(subSrc, subDst, nil)
			if err != nil {
				return "", err
			}
		}
	}

	err = CopyStat(src, dst, nil)
	if err != nil {
		return "", err
	}

	return dst, nil
}

type RmTreeOptions struct {
	IgnoreErrors bool
	OnError      func(fn func(string), path string, exec_info interface{})
}

// Recursively delete a directory tree.
//
// If ignore_errors is set, errors are ignored; otherwise, if onerror
// is set, it is called to handle the error with arguments (func,
// path, exc_info) where func is platform and implementation dependent;
// path is the argument to that function that caused it to fail; and
// exc_info is a tuple returned by sys.exc_info().  If ignore_errors
// is false and onerror is None, an exception is raised.
func RmTree(path string, options *RmTreeOptions) error {
	// onerror := func(fn func(string), path string, exec_info interface{}) {}
	// if options != nil && !options.IgnoreErrors && options.OnError != nil {
	// 	onerror = options.OnError
	// }
	return os.Remove(path)
}
