package asset

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Assets interface {
	List() []Asset // return all files
	Find(name string) (Asset, error)
	ServeHTTP(w http.ResponseWriter, r *http.Request) // implement http.FileServer
}

type Asset interface {
	List() ([]Asset, error)
	Bytes() []byte // return file bytes
	Name() string  // return file serve name
	http.File      // implement http.FileSystem
	Template() *template.Template
}

var (
	errSeekInvalid  = errors.New("invalid whence")
	errSeekNegative = errors.New("negative position")
	errNotDir       = errors.New("file is not a dir")
)

type data struct {
	prefix string
	files  map[string]*file
	all    *file
}

func (d *data) Find(name string) (Asset, error) {
	if f, found := d.files[name]; found {
		return &fileReader{f, 0}, nil
	}
	return nil, os.ErrNotExist
}

func (d *data) List() []Asset {
	return d.all.assets
}

func (d *data) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f, found := d.files[r.RequestURI]; found {
		index, indexFound := d.files[filepath.Join(r.RequestURI, "index.html")]
		if indexFound {
			f = index
		} else if f.isDir {
			dirList(w, r, &fileReader{f, 0})
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(f.size))
		w.Header().Set("Content-Type", f.cType)
		w.Header().Set("Date", fmt.Sprint(f.mTime))
		w.Write(f.b)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

type fileReader struct {
	*file
	i int64
}

func (r *fileReader) Read(p []byte) (n int, err error) {
	if r.i >= r.size {
		return 0, io.EOF
	}
	n = copy(p, r.file.b[r.i:])
	r.i += int64(n)
	return
}

func (r *fileReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = r.size + offset
	default:
		return 0, errSeekInvalid
	}
	if abs < 0 {
		return 0, errSeekNegative
	}
	r.i = abs
	return abs, nil
}

func (f *fileReader) Close() error {
	return nil
}

type file struct {
	*fileInfo
	id     int    // file id (depth_num)
	path   string // path
	dirP   string // dir path
	sPath  string // serve path
	b      []byte // data
	cb     []byte // data
	infos  []os.FileInfo
	files  []*file
	assets []Asset
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, errors.New("not dir")
	}
	if count < 0 {
		return f.infos, nil
	}
	if count >= len(f.infos) {
		count = len(f.infos) - 1
	}
	return f.infos[:count], nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *file) File() (http.File, error) {
	return &fileReader{f, 0}, nil
}

func (f *file) Bytes() []byte {
	return f.b
}

func (f *file) Name() string {
	return f.sPath
}

func (f *file) List() ([]Asset, error) {
	if f.isDir {
		return f.assets, nil
	}
	return nil, errNotDir
}

func (f *file) Template() *template.Template {
	t, err := template.New(f.name).Parse(string(f.b))
	if err != nil {
		panic(err)
	}
	return t
}

func (f *file) keyFileName() string {
	return fmt.Sprintf("_file_%d", f.id)
}

func (f *file) keyBytesName() string {
	return fmt.Sprintf("_compress_bytes_%d", f.id)
}

func (f *file) keyMTime() string {
	return fmt.Sprintf("_mTime_%d", f.id)
}

type fileInfo struct {
	name  string
	isDir bool
	size  int64
	mode  os.FileMode
	mTime time.Time
	cType string
}

// base name of the file
func (f *fileInfo) Name() string {
	return f.name
}

// length in bytes for regular files; system-dependent for others
func (f *fileInfo) Size() int64 {
	return f.size
}

// file mode bits
func (f *fileInfo) Mode() os.FileMode {
	return f.mode
}

// modification time
func (f *fileInfo) ModTime() time.Time {
	return f.mTime
}

// abbreviation for Mode().IsDir()
func (f *fileInfo) IsDir() bool {
	return f.isDir
}

// underlying data source (can return nil)
func (f *fileInfo) Sys() interface{} {
	return nil
}

// dirList copied from http.FileSystem
func dirList(w http.ResponseWriter, r *http.Request, f http.File) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error reading directory")
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<pre>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		url := url.URL{Path: filepath.Join(r.RequestURI, name)}
		fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", url.String(), d.Name())
	}
	fmt.Fprintf(w, "</pre>\n")
}

func unCompress(in []byte) []byte {
	r := bytes.NewBuffer(in)
	zr, _ := zlib.NewReader(r)
	defer zr.Close()
	bs, _ := ioutil.ReadAll(zr)
	return bs
}

var (
	Root Assets
	fs   []*file
	root = &data{}
)

func init() {
	for _, f := range fs {
		f.b = unCompress(f.cb)
		if !f.isDir || len(f.files) != 0 {
			continue
		}
		for _, ff := range fs {
			if ff.dirP == f.path {
				f.infos = append(f.infos, ff.fileInfo)
				f.files = append(f.files, ff)
				f.assets = append(f.assets, &fileReader{ff, 0})
			}
		}
	}

	all := &file{fileInfo: &fileInfo{isDir: true}}
	for _, f := range fs {
		if f.IsDir() {
			root.files[f.sPath+"/"] = f
		}
		root.files[f.sPath] = f
		root.files[f.path] = f
		all.files = append(all.files, f)
		all.infos = append(all.infos, f.fileInfo)
		all.assets = append(all.assets, &fileReader{f, 0})
	}
	root.all = all
	Root = root
}

func List() []Asset {
	return root.List()
}

func Find(name string) (Asset, error) {
	return root.Find(name)
}

var Handler = func(w http.ResponseWriter, r *http.Request) {
	root.ServeHTTP(w, r)
}
