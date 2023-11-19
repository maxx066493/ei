/*
 * Copyright 2015 The Kythe Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package vfs defines a generic file system interface commonly used by Kythe
// libraries.
package vfs // import "kythe.io/kythe/go/platform/vfs"

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// globPattern is used by globWalker to escape glob special characters.
var globPattern = regexp.MustCompile(`[[*?\\]`)

// ErrNotSupported is returned for all unsupported VFS operations.
var ErrNotSupported = errors.New("operation not supported")

// Interface is a virtual file system interface for reading and writing files.
// It is used to wrap the normal os package functions so that other file storage
// implementations be used in lieu.  For instance, there could be
// implementations for cloud storage back-ends or databases.  Depending on the
// implementation, the Writer methods can be unsupported and always return
// ErrNotSupported.
type Interface interface {
	Reader
	Writer
}

// TempFile composes io.WriteCloser and access to its "name". For
// file-based implementations, this should be the full path to the full.
type TempFile interface {
	io.WriteCloser
	Name() string
}

// Reader is a virtual file system interface for reading files.
type Reader interface {
	// Stat returns file status information for path, as os.Stat.
	Stat(ctx context.Context, path string) (os.FileInfo, error)

	// Open opens an existing file for reading, as os.Open.
	Open(ctx context.Context, path string) (FileReader, error)

	// Glob returns all the paths matching the specified glob pattern, as
	// filepath.Glob.
	Glob(ctx context.Context, glob string) ([]string, error)
}

// Walker is a virtual file system interface for traversing directories.
type Walker interface {
	// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root.
	// See filepath.Walk for more details.
	Walk(ctx context.Context, root string, walkFn filepath.WalkFunc) error
}

// Writer is a virtual file system interface for writing files.
type Writer interface {
	// MkdirAll recursively creates the specified directory path with the given
	// permissions, as os.MkdirAll.
	MkdirAll(ctx context.Context, path string, mode os.FileMode) error

	// Create creates a new file for writing, as os.Create.
	Create(ctx context.Context, path string) (io.WriteCloser, error)

	// CreateTempFile creates a new temp file returning a TempFile. The
	// name of the file is constructed from dir pattern and per
	// ioutil.TempFile:
	// The filename is generated by taking pattern and adding a random
	// string to the end. If pattern includes a "*", the random string
	// replaces the last "*". If dir is the empty string, CreateTempFile
	// uses an unspecified default directory.
	CreateTempFile(ctx context.Context, dir, pattern string) (TempFile, error)

	// Rename renames oldPath to newPath, as os.Rename, overwriting newPath if
	// it exists.
	Rename(ctx context.Context, oldPath, newPath string) error

	// Remove deletes the file specified by path, as os.Remove.
	Remove(ctx context.Context, path string) error
}

// FileReader composes interfaces from io that readable files from the vfs must
// implement.
type FileReader interface {
	io.ReadCloser
	io.ReaderAt
	io.Seeker
}

// Default is the global default VFS used by Kythe libraries that wish to access
// the file system.  This is usually the LocalFS and should only be changed in
// very specialized cases (i.e. don't change it).
var Default Interface = LocalFS{}

// ReadFile is the equivalent of ioutil.ReadFile using the Default VFS.
func ReadFile(ctx context.Context, filename string) ([]byte, error) {
	f, err := Open(ctx, filename)
	if err != nil {
		return nil, err
	}
	defer f.Close() // ignore errors
	return ioutil.ReadAll(f)
}

// Stat returns file status information for path, using the Default VFS.
func Stat(ctx context.Context, path string) (os.FileInfo, error) { return Default.Stat(ctx, path) }

// MkdirAll recursively creates the specified directory path with the given
// permissions, using the Default VFS.
func MkdirAll(ctx context.Context, path string, mode os.FileMode) error {
	return Default.MkdirAll(ctx, path, mode)
}

// Open opens an existing file for reading, using the Default VFS.
func Open(ctx context.Context, path string) (FileReader, error) { return Default.Open(ctx, path) }

// Create creates a new file for writing, using the Default VFS.
func Create(ctx context.Context, path string) (io.WriteCloser, error) {
	return Default.Create(ctx, path)
}

// CreateTempFile creates a new TempFile, using the Default VFS.
func CreateTempFile(ctx context.Context, dir, pattern string) (TempFile, error) {
	return Default.CreateTempFile(ctx, dir, pattern)
}

// Rename renames oldPath to newPath, using the Default VFS, overwriting newPath
// if it exists.
func Rename(ctx context.Context, oldPath, newPath string) error {
	return Default.Rename(ctx, oldPath, newPath)
}

// Remove deletes the file specified by path, using the Default VFS.
func Remove(ctx context.Context, path string) error { return Default.Remove(ctx, path) }

// Glob returns all the paths matching the specified glob pattern, using the
// Default VFS.
func Glob(ctx context.Context, glob string) ([]string, error) { return Default.Glob(ctx, glob) }

// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root.
// See filepath.Walk for more details.
func Walk(ctx context.Context, root string, walkFn filepath.WalkFunc) error {
	return NewWalker(Default).Walk(ctx, root, walkFn)
}

// NewWalker returns a Walker instance over the provided reader.
func NewWalker(r Reader) Walker {
	w, ok := r.(Walker)
	if ok {
		return w
	}
	return &globWalker{r}
}

// LocalFS implements the VFS interface using the standard Go library.
type LocalFS struct{}

// Stat implements part of the VFS interface.
func (LocalFS) Stat(_ context.Context, path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// MkdirAll implements part of the VFS interface.
func (LocalFS) MkdirAll(_ context.Context, path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

// Open implements part of the VFS interface.
func (LocalFS) Open(_ context.Context, path string) (FileReader, error) {
	if path == "-" {
		return stdinWrapper{os.Stdin}, nil
	}
	return os.Open(path)
}

// Create implements part of the VFS interface.
func (LocalFS) Create(_ context.Context, path string) (io.WriteCloser, error) {
	return os.Create(path)
}

// CreateTempFile implements part of the VFS interface.
func (LocalFS) CreateTempFile(_ context.Context, dir, pattern string) (TempFile, error) {
	return ioutil.TempFile(dir, pattern)
}

// Rename implements part of the VFS interface.
func (LocalFS) Rename(_ context.Context, oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// Remove implements part of the VFS interface.
func (LocalFS) Remove(_ context.Context, path string) error {
	return os.Remove(path)
}

// Glob implements part of the VFS interface.
func (LocalFS) Glob(_ context.Context, glob string) ([]string, error) {
	return filepath.Glob(glob)
}

// Walk implements part of the VFS interface.
func (LocalFS) Walk(_ context.Context, root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

// UnsupportedWriter implements the Writer interface methods with stubs that
// always return ErrNotSupported.
type UnsupportedWriter struct{ Reader }

// Create implements part of Writer interface.  It is not supported.
func (UnsupportedWriter) Create(_ context.Context, _ string) (io.WriteCloser, error) {
	return nil, ErrNotSupported
}

// CreateTempFile implements part of the VFS interface. It is not supported.
func (UnsupportedWriter) CreateTempFile(_ context.Context, dir, pattern string) (TempFile, error) {
	return nil, ErrNotSupported
}

// MkdirAll implements part of Writer interface.  It is not supported.
func (UnsupportedWriter) MkdirAll(_ context.Context, _ string, _ os.FileMode) error {
	return ErrNotSupported
}

// Rename implements part of Writer interface.  It is not supported.
func (UnsupportedWriter) Rename(_ context.Context, _, _ string) error { return ErrNotSupported }

// Remove implements part of Writer interface.  It is not supported.
func (UnsupportedWriter) Remove(_ context.Context, _ string) error { return ErrNotSupported }

// UnseekableFileReader implements the io.Seeker and io.ReaderAt at portion of
// FileReader with stubs that always return ErrNotSupported.
type UnseekableFileReader struct {
	io.ReadCloser
}

// ReadAt implements io.ReaderAt interface. It is not supported.
func (UnseekableFileReader) ReadAt([]byte, int64) (int, error) {
	return 0, ErrNotSupported
}

// Seek implements io.Seeker interface. It is not supported.
func (UnseekableFileReader) Seek(int64, int) (int64, error) {
	return 0, ErrNotSupported
}

// stdinWrapper is similar in purpose to ioutil.NopCloser, but allows access to
// other os.File methods that implement FileReader rather than restricting to
// just ioutil.ReadCloser.
type stdinWrapper struct {
	*os.File
}

func (stdinWrapper) Close() error {
	return nil
}

// globWalker wraps a Reader interface using Glob and Stat to implement Walk.
type globWalker struct {
	r Reader
}

// escapeGlob escapes glob special characters.
func escapeGlob(path string) string {
	return globPattern.ReplaceAllString(path, `\$0`)
}

// Walk implements the Walker interface by delegating to Glob and Stat.
func (gw *globWalker) Walk(ctx context.Context, root string, walkFn filepath.WalkFunc) error {
	info, err := gw.r.Stat(ctx, root)
	if err != nil {
		err = walkFn(root, nil, err)
	} else {
		err = gw.walk(ctx, root, info, walkFn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err

}

// walk recusively descends path using vfs.Glob, calling walkFn on the results.
func (gw *globWalker) walk(ctx context.Context, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}

	names, err := gw.r.Glob(ctx, filepath.Join(escapeGlob(path), "*"))
	userErr := walkFn(path, info, err)
	// If err != nil, walk can't walk into this directory.
	// userErr != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and userErr isn't nil, walk will return.
	if err != nil || userErr != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return userErr
	}
	for _, name := range names {
		fileInfo, err := gw.r.Stat(ctx, name)
		if err != nil {
			if err := walkFn(name, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else if err := gw.walk(ctx, name, fileInfo, walkFn); err != nil {
			if !fileInfo.IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}
	return nil
}