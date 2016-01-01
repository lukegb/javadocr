package javadocr

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type ZipFileSystem struct {
	r    *zip.Reader
	root *ZipFolder
}

func (fs *ZipFileSystem) buildStructure() error {
	fs.root = NewZipFolder("")
	for _, n := range fs.r.File {
		nPath, nName := path.Split(n.Name)
		if nName == "" {
			continue
		}

		dir, err := fs.getDirectoryByPath(nPath)
		if err != nil {
			return err
		}
		zf, err := NewZipFile(n)
		if err != nil {
			return err
		}
		dir.inodes = append(dir.inodes, zf)
		dir.inodesByName[nName] = zf
	}
	return nil
}

func (fs *ZipFileSystem) getDirectoryByPath(name string) (*ZipFolder, error) {
	namePieces := strings.Split(strings.TrimPrefix(name, "/"), "/")
	curDir := fs.root
	for _, piece := range namePieces {
		if piece == "" {
			continue
		}

		// try and find this piece in curDir
		if nextInode, ok := curDir.inodesByName[piece]; !ok {
			nextDir := NewZipFolder(piece)
			curDir.inodes = append(curDir.inodes, nextDir)
			curDir.inodesByName[piece] = nextDir
			curDir = nextDir
		} else if nextDir, ok := nextInode.(*ZipFolder); !ok {
			return nil, os.ErrExist
		} else {
			curDir = nextDir
		}
	}
	return curDir, nil
}

func NewZipFileSystem(r *zip.Reader) (*ZipFileSystem, error) {
	zfs := &ZipFileSystem{
		r: r,
	}
	return zfs, zfs.buildStructure()
}

func (fs *ZipFileSystem) Open(name string) (http.File, error) {
	log.Printf("opening %s", name)

	if name == "/" {
		return fs.root, nil
	}

	nPath, nName := path.Split(name)
	nDir, err := fs.getDirectoryByPath(nPath)
	if err != nil {
		return nil, err
	}

	if f, ok := nDir.inodesByName[nName]; !ok {
		return nil, os.ErrNotExist
	} else if ffolder, ok := f.(*ZipFolder); ok {
		return ffolder, nil
	} else if ffile, ok := f.(*ZipFile); ok {
		return ffile, nil
	} else {
		panic(fmt.Sprintf("found a thing of weird type: %#v", f))

	}
}

type ZipFile struct {
	f  *zip.File
	fc io.ReadCloser

	curoffset  int64
	nextoffset int64
	mustseek   bool
}

func (zf *ZipFile) Close() error {
	return zf.closeFC()
}

func (zf *ZipFile) Read(b []byte) (int, error) {
	log.Println("reading from", zf.Name())
	if zf.fc == nil {
		if err := zf.openFC(); err != nil {
			return 0, err
		}
	}

	if zf.mustseek {
		if zf.nextoffset < zf.curoffset {
			// we need to reopen the file
			err := zf.reopenFC()
			if err != nil {
				return -1, err
			}
		}
		for zf.nextoffset > zf.curoffset {
			// we can just read until we get there
			tmpB := make([]byte, zf.nextoffset-zf.curoffset)
			b, err := zf.fc.Read(tmpB)
			zf.curoffset += int64(b)
			if err != nil {
				return 0, err
			}
		}
		zf.mustseek = false
	}

	n, err := zf.fc.Read(b)
	zf.curoffset += int64(n)
	return n, err
}

func (zf *ZipFile) Readdir(n int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, io.EOF
}

func (zf *ZipFile) openFC() error {
	if zf.fc != nil {
		zf.fc.Close()
		zf.fc = nil
	}
	var err error
	zf.fc, err = zf.f.Open()
	zf.curoffset = 0
	zf.mustseek = false
	return err
}

func (zf *ZipFile) closeFC() error {
	if zf.fc == nil {
		return nil
	}

	err := zf.fc.Close()
	zf.fc = nil
	return err
}

func (zf *ZipFile) reopenFC() error {
	err := zf.closeFC()
	if err != nil {
		return err
	}
	return zf.openFC()
}

func (zf *ZipFile) Seek(offset int64, whence int) (int64, error) {
	oldOffset := zf.curoffset

	var newOffset int64
	if whence == 0 {
		newOffset = offset
	} else if whence == 1 {
		newOffset = oldOffset + offset
	} else if whence == 2 {
		newOffset = int64(zf.f.UncompressedSize64) + offset
	}

	if newOffset < 0 || whence < 0 || whence > 2 {
		return -1, os.ErrInvalid
	}

	zf.nextoffset = newOffset
	zf.mustseek = true

	return newOffset, nil
}

func (zf *ZipFile) Stat() (os.FileInfo, error) {
	return zf, nil
}

func (zf *ZipFile) Name() string {
	_, name := path.Split(zf.f.FileInfo().Name())
	return name
}

func (zf *ZipFile) IsDir() bool {
	return zf.f.FileInfo().IsDir()
}

func (zf *ZipFile) Mode() os.FileMode {
	return zf.f.FileInfo().Mode()
}

func (zf *ZipFile) Size() int64 {
	return zf.f.FileInfo().Size()
}

func (zf *ZipFile) ModTime() time.Time {
	return zf.f.FileInfo().ModTime()
}

func (zf *ZipFile) Sys() interface{} {
	return zf.f.FileInfo()
}

func NewZipFile(f *zip.File) (*ZipFile, error) {
	return &ZipFile{
		f:          f,
		curoffset:  0,
		nextoffset: 0,
		mustseek:   false,
	}, nil
}

type ZipInode interface{}

type ZipFolder struct {
	name         string
	inodes       []ZipInode
	inodesByName map[string]ZipInode

	fileinfoPos int
}

func (zf *ZipFolder) Close() error {
	return nil
}

func (zf *ZipFolder) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (zf *ZipFolder) Readdir(n int) (fi []os.FileInfo, err error) {
	if n <= 0 {
		fi = make([]os.FileInfo, len(zf.inodes))
		for ipos, inode := range zf.inodes {
			fi[ipos], err = inode.(http.File).Stat()
			if err != nil {
				return fi, err
			}
		}
		return fi, err
	}

	if zf.fileinfoPos == len(zf.inodes) {
		zf.fileinfoPos = 0
		return make([]os.FileInfo, 0), io.EOF
	}

	if n > len(zf.inodes)-zf.fileinfoPos {
		n = len(zf.inodes) - zf.fileinfoPos
	}

	fi = make([]os.FileInfo, n)
	for ipos := zf.fileinfoPos; ipos < len(zf.inodes) && ipos < zf.fileinfoPos+n; ipos++ {
		fi[ipos-zf.fileinfoPos], err = zf.inodes[ipos].(http.File).Stat()
		if err != nil {
			zf.fileinfoPos += ipos
			return fi, err
		}
	}
	zf.fileinfoPos += n
	return fi, err
}

func (zf *ZipFolder) Seek(n int64, whence int) (int64, error) {
	return 0, io.EOF
}

func (zf *ZipFolder) Stat() (os.FileInfo, error) {
	return zf, nil
}

func (zf *ZipFolder) IsDir() bool {
	return true
}

func (zf *ZipFolder) Name() string {
	return zf.name
}

func (zf *ZipFolder) Size() int64 {
	return 0
}

func (zf *ZipFolder) Sys() interface{} {
	return nil
}

func (zf *ZipFolder) ModTime() time.Time {
	return time.Now()
}

func (zf *ZipFolder) Mode() os.FileMode {
	return os.ModeDir | 0555
}

func NewZipFolder(name string) *ZipFolder {
	return &ZipFolder{
		name:         name,
		inodes:       make([]ZipInode, 0),
		inodesByName: make(map[string]ZipInode),
		fileinfoPos:  0,
	}
}
