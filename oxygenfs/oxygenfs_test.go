package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	/*	"errors"
		"log"
		"os/exec"
		"runtime"
		"syscall"
		"time"

		"oxygen-fuse-fs"
		"oxygen-fuse-fs/fs"
		"oxygen-fuse-fs/fs/fstestutil"
		"oxygen-fuse-fs/fs/fstestutil/record"
		"oxygen-fuse-fs/fuseutil"
		"oxygen-fuse-fs/syscallx"*/)

var _ = rand.Float32
var _ = time.Saturday

const (
	TestOxygenEndpoint   = "http://localhost:9000"
	TestTitaniumEndpoint = "http://localhost:9002"
)

// TO TEST:
//	Lookup(*LookupRequest, *LookupResponse)
//	Getattr(*GetattrRequest, *GetattrResponse)
//	Attr with explicit inode
//	Setattr(*SetattrRequest, *SetattrResponse)
//	Access(*AccessRequest)
//	Open(*OpenRequest, *OpenResponse)
//	Write(*WriteRequest, *WriteResponse)
//	Flush(*FlushRequest, *FlushResponse)

/*func init() {
	fstestutil.DebugByDefault()
}

// childMapFS is an FS with one fixed child named "child".
type childMapFS map[string]fs.Node

var _ = fs.FS(childMapFS{})
var _ = fs.Node(childMapFS{})
var _ = fs.NodeStringLookuper(childMapFS{})

func (f childMapFS) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0777}
}

func (f childMapFS) Root() (fs.Node, fuse.Error) {
	return f, nil
}

func (f childMapFS) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	child, ok := f[name]
	if !ok {
		return nil, fuse.ENOENT
	}
	return child, nil
}

// simpleFS is a trivial FS that just implements the Root method.
type simpleFS struct {
	node fs.Node
}

var _ = fs.FS(simpleFS{})

func (f simpleFS) Root() (fs.Node, fuse.Error) {
	return f.node, nil
}

// file can be embedded in a struct to make it look like a file.
type file struct{}

func (f file) Attr() fuse.Attr { return fuse.Attr{Mode: 0666} }

// dir can be embedded in a struct to make it look like a directory.
type dir struct{}

func (f dir) Attr() fuse.Attr { return fuse.Attr{Mode: os.ModeDir | 0777} }

// symlink can be embedded in a struct to make it look like a symlink.
type symlink struct {
	target string
}

func (f symlink) Attr() fuse.Attr { return fuse.Attr{Mode: os.ModeSymlink | 0666} }

// fifo can be embedded in a struct to make it look like a named pipe.
type fifo struct{}

func (f fifo) Attr() fuse.Attr { return fuse.Attr{Mode: os.ModeNamedPipe | 0666} }

type badRootFS struct{}

func (badRootFS) Root() (fs.Node, fuse.Error) {
	// pick a really distinct error, to identify it later
	return nil, fuse.Errno(syscall.ENAMETOOLONG)
}

func testRootErr(t *testing.T) {
	//t.Parallel()
	fmt.Println("Mounting")
	mnt, err := fstestutil.MountedT(t, badRootFS{})
	if err == nil {
		// path for synchronous mounts (linux): started out fine, now
		// wait for Serve to cycle through
		err = <-mnt.Error
		// without this, unmount will keep failing with EBUSY; nudge
		// kernel into realizing InitResponse will not happen
		mnt.Conn.Close()
		mnt.Close()
	}

	if err == nil {
		t.Fatal("expected an error")
	}
	// TODO this should not be a textual comparison, Serve hides
	// details
	if err.Error() != "cannot obtain root node: file name too long" {
		t.Errorf("Unexpected error: %v", err)
	}
	fmt.Println("Unmounting")
}

type testStatFS struct{}

func (f testStatFS) Root() (fs.Node, fuse.Error) {
	return f, nil
}

func (f testStatFS) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0777}
}

func (f testStatFS) Statfs(req *fuse.StatfsRequest, resp *fuse.StatfsResponse, int fs.Intr) fuse.Error {
	resp.Blocks = 42
	resp.Files = 13
	return nil
}

func testStatfs(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, testStatFS{})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	{
		var st syscall.Statfs_t
		err = syscall.Statfs(mnt.Dir, &st)
		if err != nil {
			t.Errorf("Statfs failed: %v", err)
		}
		t.Logf("Statfs got: %#v", st)
		if g, e := st.Blocks, uint64(42); g != e {
			t.Errorf("got Blocks = %d; want %d", g, e)
		}
		if g, e := st.Files, uint64(13); g != e {
			t.Errorf("got Files = %d; want %d", g, e)
		}
	}

	{
		var st syscall.Statfs_t
		f, err := os.Open(mnt.Dir)
		if err != nil {
			t.Errorf("Open for fstatfs failed: %v", err)
		}
		defer f.Close()
		err = syscall.Fstatfs(int(f.Fd()), &st)
		if err != nil {
			t.Errorf("Fstatfs failed: %v", err)
		}
		t.Logf("Fstatfs got: %#v", st)
		if g, e := st.Blocks, uint64(42); g != e {
			t.Errorf("got Blocks = %d; want %d", g, e)
		}
		if g, e := st.Files, uint64(13); g != e {
			t.Errorf("got Files = %d; want %d", g, e)
		}
	}

}

// Test Stat of root.

type root struct{}

func (f root) Root() (fs.Node, fuse.Error) {
	return f, nil
}

func (root) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0555}
}

func testStatRoot(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, root{})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	fi, err := os.Stat(mnt.Dir)
	if err != nil {
		t.Fatalf("root getattr failed with %v", err)
	}
	mode := fi.Mode()
	if (mode & os.ModeType) != os.ModeDir {
		t.Errorf("root is not a directory: %#v", fi)
	}
	if mode.Perm() != 0555 {
		t.Errorf("root has weird access mode: %v", mode.Perm())
	}
	switch stat := fi.Sys().(type) {
	case *syscall.Stat_t:
		if stat.Ino != 1 {
			t.Errorf("root has wrong inode: %v", stat.Ino)
		}
		if stat.Nlink != 1 {
			t.Errorf("root has wrong link count: %v", stat.Nlink)
		}
		if stat.Uid != 0 {
			t.Errorf("root has wrong uid: %d", stat.Uid)
		}
		if stat.Gid != 0 {
			t.Errorf("root has wrong gid: %d", stat.Gid)
		}
	}
}

// Test Read calling ReadAll.

type readAll struct{ file }

const hi = "hello, world"

func (readAll) ReadAll(intr fs.Intr) ([]byte, fuse.Error) {
	return []byte(hi), nil
}

func testReadAll(t *testing.T, path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if string(data) != hi {
		t.Errorf("readAll = %q, want %q", data, hi)
	}
}

func testtReadAll(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": readAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	testReadAll(t, mnt.Dir+"/child")
}

// Test Read.

type readWithHandleRead struct{ file }

func (readWithHandleRead) Read(req *fuse.ReadRequest, resp *fuse.ReadResponse, intr fs.Intr) fuse.Error {
	fuseutil.HandleRead(req, resp, []byte(hi))
	return nil
}

func testReadAllWithHandleRead(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": readWithHandleRead{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	testReadAll(t, mnt.Dir+"/child")
}

// Test Release.

type release struct {
	file
	record.ReleaseWaiter
}

func testRelease(t *testing.T) {
	//t.Parallel()
	r := &release{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": r})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	f, err := os.Open(mnt.Dir + "/child")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if !r.WaitForRelease(1 * time.Second) {
		t.Error("Close did not Release in time")
	}
}

// Test Write calling basic Write, with an fsync thrown in too.

type write struct {
	file
	record.Writes
	record.Fsyncs
}

func testWrite(t *testing.T) {
	//t.Parallel()
	w := &write{}

	mnt, err := fstestutil.MountedT(t, childMapFS{"child": w})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	f, err := os.Create(mnt.Dir + "/child")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	n, err := f.Write([]byte(hi))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(hi) {
		t.Fatalf("short write; n=%d; hi=%d", n, len(hi))
	}

	err = syscall.Fsync(int(f.Fd()))
	if err != nil {
		t.Fatalf("Fsync = %v", err)
	}
	if w.RecordedFsync() == (fuse.FsyncRequest{}) {
		t.Errorf("never received expected fsync call")
	}

	err = f.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got := string(w.RecordedWriteData()); got != hi {
		t.Errorf("write = %q, want %q", got, hi)
	}
}
*/

/*
// Test Write calling Setattr+Write+Flush.

type writeTruncateFlush struct {
	file
	record.Writes
	record.Setattrs
	record.Flushes
}

func testWriteTruncateFlush(t *testing.T) {
	//t.Parallel()
	fmt.Println("mount1")
	w := &writeTruncateFlush{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": w})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = ioutil.WriteFile(mnt.Dir+"/child", []byte(hi), 0666)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if w.RecordedSetattr() == (fuse.SetattrRequest{}) {
		t.Errorf("writeTruncateFlush expected Setattr")
	}
	if !w.RecordedFlush() {
		t.Errorf("writeTruncateFlush expected Setattr")
	}
	if got := string(w.RecordedWriteData()); got != hi {
		t.Errorf("writeTruncateFlush = %q, want %q", got, hi)
	}
	fmt.Println("unmount1")
}

// Test Mkdir.

type mkdir1 struct {
	dir
	record.Mkdirs
}

func (f *mkdir1) Mkdir(req *fuse.MkdirRequest, intr fs.Intr) (fs.Node, fuse.Error) {
	f.Mkdirs.Mkdir(req, intr)
	return &mkdir1{}, nil
}

func testMkdir(t *testing.T) {
	//t.Parallel()
	fmt.Println("mount")
	f := &mkdir1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	// uniform umask needed to make os.Mkdir's mode into something
	// reproducible
	defer syscall.Umask(syscall.Umask(0022))
	err = os.Mkdir(mnt.Dir+"/foo", 0771)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	want := fuse.MkdirRequest{Name: "foo", Mode: os.ModeDir | 0751}
	if g, e := f.RecordedMkdir(), want; g != e {
		t.Errorf("mkdir saw %+v, want %+v", g, e)
	}
	fmt.Println("unmount")
}

// Test Create (and fsync)

type create1file struct {
	file
	record.Fsyncs
}

type create1 struct {
	dir
	f create1file
}

func (f *create1) Create(req *fuse.CreateRequest, resp *fuse.CreateResponse, intr fs.Intr) (fs.Node, fs.Handle, fuse.Error) {
	if req.Name != "foo" {
		log.Printf("ERROR create1.Create unexpected name: %q\n", req.Name)
		return nil, nil, fuse.EPERM
	}
	flags := req.Flags

	// OS X does not pass O_TRUNC here, Linux does; as this is a
	// Create, that's acceptable
	flags &^= fuse.OpenFlags(os.O_TRUNC)

	if runtime.GOOS == "linux" {
		// Linux <3.7 accidentally leaks O_CLOEXEC through to FUSE;
		// avoid spurious test failures
		flags &^= fuse.OpenFlags(syscall.O_CLOEXEC)
	}

	if g, e := flags, fuse.OpenFlags(os.O_CREATE|os.O_RDWR); g != e {
		log.Printf("ERROR create1.Create unexpected flags: %v != %v\n", g, e)
		return nil, nil, fuse.EPERM
	}
	if g, e := req.Mode, os.FileMode(0644); g != e {
		log.Printf("ERROR create1.Create unexpected mode: %v != %v\n", g, e)
		return nil, nil, fuse.EPERM
	}
	return &f.f, &f.f, nil
}

func testCreate(t *testing.T) {
	//t.Parallel()
	f := &create1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	// uniform umask needed to make os.Create's 0666 into something
	// reproducible
	defer syscall.Umask(syscall.Umask(0022))
	ff, err := os.Create(mnt.Dir + "/foo")
	if err != nil {
		t.Fatalf("create1 WriteFile: %v", err)
	}

	err = syscall.Fsync(int(ff.Fd()))
	if err != nil {
		t.Fatalf("Fsync = %v", err)
	}

	if f.f.RecordedFsync() == (fuse.FsyncRequest{}) {
		t.Errorf("never received expected fsync call")
	}

	ff.Close()
}

// Test Create + Write + Remove

type create3file struct {
	file
	record.Writes
}

type create3 struct {
	dir
	f          create3file
	fooCreated record.MarkRecorder
	fooRemoved record.MarkRecorder
}

func (f *create3) Create(req *fuse.CreateRequest, resp *fuse.CreateResponse, intr fs.Intr) (fs.Node, fs.Handle, fuse.Error) {
	if req.Name != "foo" {
		log.Printf("ERROR create3.Create unexpected name: %q\n", req.Name)
		return nil, nil, fuse.EPERM
	}
	f.fooCreated.Mark()
	return &f.f, &f.f, nil
}

func (f *create3) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	if f.fooCreated.Recorded() && !f.fooRemoved.Recorded() && name == "foo" {
		return &f.f, nil
	}
	return nil, fuse.ENOENT
}

func (f *create3) Remove(r *fuse.RemoveRequest, intr fs.Intr) fuse.Error {
	if f.fooCreated.Recorded() && !f.fooRemoved.Recorded() &&
		r.Name == "foo" && !r.Dir {
		f.fooRemoved.Mark()
		return nil
	}
	return fuse.ENOENT
}

func testCreateWriteRemove(t *testing.T) {
	//t.Parallel()
	f := &create3{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = ioutil.WriteFile(mnt.Dir+"/foo", []byte(hi), 0666)
	if err != nil {
		t.Fatalf("create3 WriteFile: %v", err)
	}
	if got := string(f.f.RecordedWriteData()); got != hi {
		t.Fatalf("create3 write = %q, want %q", got, hi)
	}

	err = os.Remove(mnt.Dir + "/foo")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	err = os.Remove(mnt.Dir + "/foo")
	if err == nil {
		t.Fatalf("second Remove = nil; want some error")
	}
}

// Test symlink + readlink

// is a Node that is a symlink to target
type symlink1link struct {
	symlink
	target string
}

func (f symlink1link) Readlink(*fuse.ReadlinkRequest, fs.Intr) (string, fuse.Error) {
	return f.target, nil
}

type symlink1 struct {
	dir
	record.Symlinks
}

func (f *symlink1) Symlink(req *fuse.SymlinkRequest, intr fs.Intr) (fs.Node, fuse.Error) {
	f.Symlinks.Symlink(req, intr)
	return symlink1link{target: req.Target}, nil
}

func testSymlink(t *testing.T) {
	//t.Parallel()
	f := &symlink1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	const target = "/some-target"

	err = os.Symlink(target, mnt.Dir+"/symlink.file")
	if err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	want := fuse.SymlinkRequest{NewName: "symlink.file", Target: target}
	if g, e := f.RecordedSymlink(), want; g != e {
		t.Errorf("symlink saw %+v, want %+v", g, e)
	}

	gotName, err := os.Readlink(mnt.Dir + "/symlink.file")
	if err != nil {
		t.Fatalf("os.Readlink: %v", err)
	}
	if gotName != target {
		t.Errorf("os.Readlink = %q; want %q", gotName, target)
	}
}

// Test link

type link1 struct {
	dir
	record.Links
}

func (f *link1) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	if name == "old" {
		return file{}, nil
	}
	return nil, fuse.ENOENT
}

func (f *link1) Link(r *fuse.LinkRequest, old fs.Node, intr fs.Intr) (fs.Node, fuse.Error) {
	f.Links.Link(r, old, intr)
	return file{}, nil
}

func testLink(t *testing.T) {
	//t.Parallel()
	f := &link1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = os.Link(mnt.Dir+"/old", mnt.Dir+"/new")
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	got := f.RecordedLink()
	want := fuse.LinkRequest{
		NewName: "new",
		// unpredictable
		OldNode: got.OldNode,
	}
	if g, e := got, want; g != e {
		t.Fatalf("link saw %+v, want %+v", g, e)
	}
}

// Test Rename

type rename1 struct {
	dir
	renamed record.Counter
}

func (f *rename1) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	if name == "old" {
		return file{}, nil
	}
	return nil, fuse.ENOENT
}

func (f *rename1) Rename(r *fuse.RenameRequest, newDir fs.Node, intr fs.Intr) fuse.Error {
	if r.OldName == "old" && r.NewName == "new" && newDir == f {
		f.renamed.Inc()
		return nil
	}
	return fuse.EIO
}

func testRename(t *testing.T) {
	//t.Parallel()
	f := &rename1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = os.Rename(mnt.Dir+"/old", mnt.Dir+"/new")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if g, e := f.renamed.Count(), uint32(1); g != e {
		t.Fatalf("expected rename didn't happen: %d != %d", g, e)
	}
	err = os.Rename(mnt.Dir+"/old2", mnt.Dir+"/new2")
	if err == nil {
		t.Fatal("expected error on second Rename; got nil")
	}
}

// Test mknod

type mknod1 struct {
	dir
	record.Mknods
}

func (f *mknod1) Mknod(r *fuse.MknodRequest, intr fs.Intr) (fs.Node, fuse.Error) {
	f.Mknods.Mknod(r, intr)
	return fifo{}, nil
}

func testMknod(t *testing.T) {
	//t.Parallel()
	if os.Getuid() != 0 {
		t.Skip("skipping unless root")
	}

	f := &mknod1{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	defer syscall.Umask(syscall.Umask(0))
	err = syscall.Mknod(mnt.Dir+"/node", syscall.S_IFIFO|0666, 123)
	if err != nil {
		t.Fatalf("Mknod: %v", err)
	}

	want := fuse.MknodRequest{
		Name: "node",
		Mode: os.FileMode(os.ModeNamedPipe | 0666),
		Rdev: uint32(123),
	}
	if runtime.GOOS == "linux" {
		// Linux fuse doesn't echo back the rdev if the node
		// isn't a device (we're using a FIFO here, as that
		// bit is portable.)
		want.Rdev = 0
	}
	if g, e := f.RecordedMknod(), want; g != e {
		t.Fatalf("mknod saw %+v, want %+v", g, e)
	}
}

// Test Read served with DataHandle.

type dataHandleTest struct {
	file
}

func (dataHandleTest) Open(*fuse.OpenRequest, *fuse.OpenResponse, fs.Intr) (fs.Handle, fuse.Error) {
	return fs.DataHandle([]byte(hi)), nil
}

func testDataHandle(t *testing.T) {
	//t.Parallel()
	f := &dataHandleTest{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	data, err := ioutil.ReadFile(mnt.Dir + "/child")
	if err != nil {
		t.Errorf("readAll: %v", err)
		return
	}
	if string(data) != hi {
		t.Errorf("readAll = %q, want %q", data, hi)
	}
}

// Test interrupt

type interrupt struct {
	file

	// strobes to signal we have a read hanging
	hanging chan struct{}
}

func (it *interrupt) Read(req *fuse.ReadRequest, resp *fuse.ReadResponse, intr fs.Intr) fuse.Error {
	select {
	case it.hanging <- struct{}{}:
	default:
	}
	<-intr
	return fuse.EINTR
}

func testInterrupt(t *testing.T) {
	//t.Parallel()
	f := &interrupt{}
	f.hanging = make(chan struct{}, 1)
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	// start a subprocess that can hang until signaled
	cmd := exec.Command("cat", mnt.Dir+"/child")

	err = cmd.Start()
	if err != nil {
		t.Errorf("interrupt: cannot start cat: %v", err)
		return
	}

	// try to clean up if child is still alive when returning
	defer cmd.Process.Kill()

	// wait till we're sure it's hanging in read
	<-f.hanging

	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		t.Errorf("interrupt: cannot interrupt cat: %v", err)
		return
	}

	p, err := cmd.Process.Wait()
	if err != nil {
		t.Errorf("interrupt: cat bork: %v", err)
		return
	}
	switch ws := p.Sys().(type) {
	case syscall.WaitStatus:
		if ws.CoreDump() {
			t.Errorf("interrupt: didn't expect cat to dump core: %v", ws)
		}

		if ws.Exited() {
			t.Errorf("interrupt: didn't expect cat to exit normally: %v", ws)
		}

		if !ws.Signaled() {
			t.Errorf("interrupt: expected cat to get a signal: %v", ws)
		} else {
			if ws.Signal() != os.Interrupt {
				t.Errorf("interrupt: cat got wrong signal: %v", ws)
			}
		}
	default:
		t.Logf("interrupt: this platform has no test coverage")
	}
}

// Test truncate

type truncate struct {
	file
	record.Setattrs
}

func testTruncate(t *testing.T, toSize int64) {
	//t.Parallel()
	f := &truncate{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = os.Truncate(mnt.Dir+"/child", toSize)
	if err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	gotr := f.RecordedSetattr()
	if gotr == (fuse.SetattrRequest{}) {
		t.Fatalf("no recorded SetattrRequest")
	}
	if g, e := gotr.Size, uint64(toSize); g != e {
		t.Errorf("got Size = %q; want %q", g, e)
	}
	if g, e := gotr.Valid&^fuse.SetattrLockOwner, fuse.SetattrSize; g != e {
		t.Errorf("got Valid = %q; want %q", g, e)
	}
	t.Logf("Got request: %#v", gotr)
}

func testTruncate42(t *testing.T) {
	testTruncate(t, 42)
}

func testTruncate0(t *testing.T) {
	testTruncate(t, 0)
}

// Test ftruncate

type ftruncate struct {
	file
	record.Setattrs
}

func testFtruncate(t *testing.T, toSize int64) {
	//t.Parallel()
	f := &ftruncate{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	{
		fil, err := os.OpenFile(mnt.Dir+"/child", os.O_WRONLY, 0666)
		if err != nil {
			t.Error(err)
			return
		}
		defer fil.Close()

		err = fil.Truncate(toSize)
		if err != nil {
			t.Fatalf("Ftruncate: %v", err)
		}
	}
	gotr := f.RecordedSetattr()
	if gotr == (fuse.SetattrRequest{}) {
		t.Fatalf("no recorded SetattrRequest")
	}
	if g, e := gotr.Size, uint64(toSize); g != e {
		t.Errorf("got Size = %q; want %q", g, e)
	}
	if g, e := gotr.Valid&^fuse.SetattrLockOwner, fuse.SetattrHandle|fuse.SetattrSize; g != e {
		t.Errorf("got Valid = %q; want %q", g, e)
	}
	t.Logf("Got request: %#v", gotr)
}

func testFtruncate42(t *testing.T) {
	testFtruncate(t, 42)
}

func testFtruncate0(t *testing.T) {
	testFtruncate(t, 0)
}

// Test opening existing file truncates

type truncateWithOpen struct {
	file
	record.Setattrs
}

func testTruncateWithOpen(t *testing.T) {
	//t.Parallel()
	f := &truncateWithOpen{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	fil, err := os.OpenFile(mnt.Dir+"/child", os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		t.Error(err)
		return
	}
	fil.Close()

	gotr := f.RecordedSetattr()
	if gotr == (fuse.SetattrRequest{}) {
		t.Fatalf("no recorded SetattrRequest")
	}
	if g, e := gotr.Size, uint64(0); g != e {
		t.Errorf("got Size = %q; want %q", g, e)
	}
	// osxfuse sets SetattrHandle here, linux does not
	if g, e := gotr.Valid&^(fuse.SetattrLockOwner|fuse.SetattrHandle), fuse.SetattrSize; g != e {
		t.Errorf("got Valid = %q; want %q", g, e)
	}
	t.Logf("Got request: %#v", gotr)
}

// Test readdir

type readdir struct {
	dir
}

func (d *readdir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	return []fuse.Dirent{
		{Name: "one", Inode: 11, Type: fuse.DT_Dir},
		{Name: "three", Inode: 13},
		{Name: "two", Inode: 12, Type: fuse.DT_File},
	}, nil
}

func testReadDir(t *testing.T) {
	//t.Parallel()
	f := &readdir{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	fil, err := os.Open(mnt.Dir)
	if err != nil {
		t.Error(err)
		return
	}
	defer fil.Close()

	// go Readdir is just Readdirnames + Lstat, there's no point in
	// testing that here; we have no consumption API for the real
	// dirent data
	names, err := fil.Readdirnames(100)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("Got readdir: %q", names)

	if len(names) != 3 ||
		names[0] != "one" ||
		names[1] != "three" ||
		names[2] != "two" {
		t.Errorf(`expected 3 entries of "one", "three", "two", got: %q`, names)
		return
	}
}

// Test Chmod.

type chmod struct {
	file
	record.Setattrs
}

func (f *chmod) Setattr(req *fuse.SetattrRequest, resp *fuse.SetattrResponse, intr fs.Intr) fuse.Error {
	if !req.Valid.Mode() {
		log.Printf("setattr not a chmod: %v", req.Valid)
		return fuse.EIO
	}
	f.Setattrs.Setattr(req, resp, intr)
	return nil
}

func testChmod(t *testing.T) {
	//t.Parallel()
	f := &chmod{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = os.Chmod(mnt.Dir+"/child", 0764)
	if err != nil {
		t.Errorf("chmod: %v", err)
		return
	}
	got := f.RecordedSetattr()
	if g, e := got.Mode, os.FileMode(0764); g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}
}

// Test open

type open struct {
	file
	record.Opens
}

func (f *open) Open(req *fuse.OpenRequest, resp *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	f.Opens.Open(req, resp, intr)
	// pick a really distinct error, to identify it later
	return nil, fuse.Errno(syscall.ENAMETOOLONG)

}

func testOpen(t *testing.T) {
	//t.Parallel()
	f := &open{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	// node: mode only matters with O_CREATE
	fil, err := os.OpenFile(mnt.Dir+"/child", os.O_WRONLY|os.O_APPEND, 0)
	if err == nil {
		t.Error("Open err == nil, expected ENAMETOOLONG")
		fil.Close()
		return
	}

	switch err2 := err.(type) {
	case *os.PathError:
		if err2.Err == syscall.ENAMETOOLONG {
			break
		}
		t.Errorf("unexpected inner error: %#v", err2)
	default:
		t.Errorf("unexpected error: %v", err)
	}

	want := fuse.OpenRequest{Dir: false, Flags: fuse.OpenFlags(os.O_WRONLY | os.O_APPEND)}
	if runtime.GOOS == "darwin" {
		// osxfuse does not let O_APPEND through at all
		//
		// https://code.google.com/p/macfuse/issues/detail?id=233
		// https://code.google.com/p/macfuse/issues/detail?id=132
		// https://code.google.com/p/macfuse/issues/detail?id=133
		want.Flags &^= fuse.OpenFlags(os.O_APPEND)
	}
	got := f.RecordedOpen()

	if runtime.GOOS == "linux" {
		// Linux <3.7 accidentally leaks O_CLOEXEC through to FUSE;
		// avoid spurious test failures
		got.Flags &^= fuse.OpenFlags(syscall.O_CLOEXEC)
	}

	if g, e := got, want; g != e {
		t.Errorf("open saw %v, want %v", g, e)
		return
	}
}

// Test Fsync on a dir

type fsyncDir struct {
	dir
	record.Fsyncs
}

func testFsyncDir(t *testing.T) {
	//t.Parallel()
	f := &fsyncDir{}
	mnt, err := fstestutil.MountedT(t, simpleFS{f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	fil, err := os.Open(mnt.Dir)
	if err != nil {
		t.Errorf("fsyncDir open: %v", err)
		return
	}
	defer fil.Close()
	err = fil.Sync()
	if err != nil {
		t.Errorf("fsyncDir sync: %v", err)
		return
	}

	got := f.RecordedFsync()
	want := fuse.FsyncRequest{
		Flags: 0,
		Dir:   true,
		// unpredictable
		Handle: got.Handle,
	}
	if runtime.GOOS == "darwin" {
		// TODO document the meaning of these flags, figure out why
		// they differ
		want.Flags = 1
	}
	if g, e := got, want; g != e {
		t.Fatalf("fsyncDir saw %+v, want %+v", g, e)
	}
}

// Test Getxattr

type getxattr struct {
	file
	record.Getxattrs
}

func (f *getxattr) Getxattr(req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse, intr fs.Intr) fuse.Error {
	f.Getxattrs.Getxattr(req, resp, intr)
	resp.Xattr = []byte("hello, world")
	return nil
}

func testGetxattr(t *testing.T) {
	//t.Parallel()
	f := &getxattr{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	buf := make([]byte, 8192)
	n, err := syscallx.Getxattr(mnt.Dir+"/child", "not-there", buf)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	buf = buf[:n]
	if g, e := string(buf), "hello, world"; g != e {
		t.Errorf("wrong getxattr content: %#v != %#v", g, e)
	}
	seen := f.RecordedGetxattr()
	if g, e := seen.Name, "not-there"; g != e {
		t.Errorf("wrong getxattr name: %#v != %#v", g, e)
	}
}

// Test Getxattr that has no space to return value

type getxattrTooSmall struct {
	file
}

func (f *getxattrTooSmall) Getxattr(req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse, intr fs.Intr) fuse.Error {
	resp.Xattr = []byte("hello, world")
	return nil
}

func testGetxattrTooSmall(t *testing.T) {
	//t.Parallel()
	f := &getxattrTooSmall{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	buf := make([]byte, 3)
	_, err = syscallx.Getxattr(mnt.Dir+"/child", "whatever", buf)
	if err == nil {
		t.Error("Getxattr = nil; want some error")
	}
	if err != syscall.ERANGE {
		t.Errorf("unexpected error: %v", err)
		return
	}
}

// Test Getxattr used to probe result size

type getxattrSize struct {
	file
}

func (f *getxattrSize) Getxattr(req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse, intr fs.Intr) fuse.Error {
	resp.Xattr = []byte("hello, world")
	return nil
}

func testGetxattrSize(t *testing.T) {
	//t.Parallel()
	f := &getxattrSize{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	n, err := syscallx.Getxattr(mnt.Dir+"/child", "whatever", nil)
	if err != nil {
		t.Errorf("Getxattr unexpected error: %v", err)
		return
	}
	if g, e := n, len("hello, world"); g != e {
		t.Errorf("Getxattr incorrect size: %d != %d", g, e)
	}
}

// Test Listxattr

type listxattr struct {
	file
	record.Listxattrs
}

func (f *listxattr) Listxattr(req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse, intr fs.Intr) fuse.Error {
	f.Listxattrs.Listxattr(req, resp, intr)
	resp.Append("one", "two")
	return nil
}

func testListxattr(t *testing.T) {
	//t.Parallel()
	f := &listxattr{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	buf := make([]byte, 8192)
	n, err := syscallx.Listxattr(mnt.Dir+"/child", buf)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	buf = buf[:n]
	if g, e := string(buf), "one\x00two\x00"; g != e {
		t.Errorf("wrong listxattr content: %#v != %#v", g, e)
	}

	want := fuse.ListxattrRequest{
		Size: 8192,
	}
	if g, e := f.RecordedListxattr(), want; g != e {
		t.Fatalf("listxattr saw %+v, want %+v", g, e)
	}
}

// Test Listxattr that has no space to return value

type listxattrTooSmall struct {
	file
}

func (f *listxattrTooSmall) Listxattr(req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse, intr fs.Intr) fuse.Error {
	resp.Xattr = []byte("one\x00two\x00")
	return nil
}

func testListxattrTooSmall(t *testing.T) {
	//t.Parallel()
	f := &listxattrTooSmall{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	buf := make([]byte, 3)
	_, err = syscallx.Listxattr(mnt.Dir+"/child", buf)
	if err == nil {
		t.Error("Listxattr = nil; want some error")
	}
	if err != syscall.ERANGE {
		t.Errorf("unexpected error: %v", err)
		return
	}
}

// Test Listxattr used to probe result size

type listxattrSize struct {
	file
}

func (f *listxattrSize) Listxattr(req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse, intr fs.Intr) fuse.Error {
	resp.Xattr = []byte("one\x00two\x00")
	return nil
}

func testListxattrSize(t *testing.T) {
	//t.Parallel()
	f := &listxattrSize{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	n, err := syscallx.Listxattr(mnt.Dir+"/child", nil)
	if err != nil {
		t.Errorf("Listxattr unexpected error: %v", err)
		return
	}
	if g, e := n, len("one\x00two\x00"); g != e {
		t.Errorf("Getxattr incorrect size: %d != %d", g, e)
	}
}

// Test Setxattr

type setxattr struct {
	file
	record.Setxattrs
}

func testSetxattr(t *testing.T) {
	//t.Parallel()
	f := &setxattr{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = syscallx.Setxattr(mnt.Dir+"/child", "greeting", []byte("hello, world"), 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// fuse.SetxattrRequest contains a byte slice and thus cannot be
	// directly compared
	got := f.RecordedSetxattr()

	if g, e := got.Name, "greeting"; g != e {
		t.Errorf("Setxattr incorrect name: %q != %q", g, e)
	}

	if g, e := got.Flags, uint32(0); g != e {
		t.Errorf("Setxattr incorrect flags: %d != %d", g, e)
	}

	if g, e := string(got.Xattr), "hello, world"; g != e {
		t.Errorf("Setxattr incorrect data: %q != %q", g, e)
	}
}

// Test Removexattr

type removexattr struct {
	file
	record.Removexattrs
}

func testRemovexattr(t *testing.T) {
	//t.Parallel()
	f := &removexattr{}
	mnt, err := fstestutil.MountedT(t, childMapFS{"child": f})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	err = syscallx.Removexattr(mnt.Dir+"/child", "greeting")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	want := fuse.RemovexattrRequest{Name: "greeting"}
	if g, e := f.RecordedRemovexattr(), want; g != e {
		t.Errorf("removexattr saw %v, want %v", g, e)
	}
}

// Test default error.

type defaultErrno struct {
	dir
}

func (f defaultErrno) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	return nil, errors.New("bork")
}

func testDefaultErrno(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, simpleFS{defaultErrno{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	_, err = os.Stat(mnt.Dir + "/trigger")
	if err == nil {
		t.Fatalf("expected error")
	}

	switch err2 := err.(type) {
	case *os.PathError:
		if err2.Err == syscall.EIO {
			break
		}
		t.Errorf("unexpected inner error: Err=%v %#v", err2.Err, err2)
	default:
		t.Errorf("unexpected error: %v", err)
	}
}

// Test custom error.

type customErrNode struct {
	dir
}

type myCustomError struct {
	fuse.ErrorNumber
}

var _ = fuse.ErrorNumber(myCustomError{})

func (myCustomError) Error() string {
	return "bork"
}

func (f customErrNode) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	return nil, myCustomError{
		ErrorNumber: fuse.Errno(syscall.ENAMETOOLONG),
	}
}

func testCustomErrno(t *testing.T) {
	//t.Parallel()
	mnt, err := fstestutil.MountedT(t, simpleFS{customErrNode{}})
	if err != nil {
		t.Fatal(err)
	}
	defer mnt.Close()

	_, err = os.Stat(mnt.Dir + "/trigger")
	if err == nil {
		t.Fatalf("expected error")
	}

	switch err2 := err.(type) {
	case *os.PathError:
		if err2.Err == syscall.ENAMETOOLONG {
			break
		}
		t.Errorf("unexpected inner error: %#v", err2)
	default:
		t.Errorf("unexpected error: %v", err)
	}
}
*/

func TestReadAndWrite(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	// Read root directory
	rootPath := mount.Dir
	list, err := ioutil.ReadDir(rootPath)
	if err != nil {
		t.Fatalf("Error while reading root: %s\n", err)
	}

	// Make sure username shows up in root directory
	var foundUsername bool
	for _, entry := range list {
		//fmt.Printf("%+v %d\n", entry, entry.IsDir())
		if entry.Name() == client.username && entry.IsDir() {
			foundUsername = true
		}
	}
	if !foundUsername {
		t.Fatalf("Returned directory entries must include username. %+v\n", list)
	}

	projectName := client.CreateRandomProject(false)

	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	filePath := projectDirectory + "test"
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	file, err = os.OpenFile(filePath, syscall.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writeLength := rand.Intn(48600)
	writeContents := RandomByteSlice(writeLength)
	n, err := file.Write(writeContents)
	if err != nil || n != writeLength {
		t.Fatalf("Failed to write. Wrote %d of %d bytes. Error: %s\n", n, writeLength, err)
	}

	//fmt.Printf("Ret: %d, Err: %s\n", ret, err)
	readOffset := int64(5)
	readSize := int64(writeLength) - readOffset
	file.Seek(readOffset, 0)

	readBuffer := make([]byte, readSize)
	n, err = file.Read(readBuffer)

	if err != nil {
		t.Fatalf("Failed to read. Error: %s\n", err)
	}
	if !bytes.Equal(writeContents[readOffset:readOffset+int64(n)], readBuffer[:n]) {
		t.Fatalf("Read contents not the same as written content.\nGot: %s\nExp: %s\n", readBuffer[:20], writeContents[readOffset:readOffset+20])
	}
}

func TestCreateAndRemoveDirectoryInProject(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	projectName := client.CreateRandomProject(false)

	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	// Create directory in project
	tempDirectory := projectDirectory + "temp"
	err = os.Mkdir(tempDirectory, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Create file in directory
	fileInTempDirectory := tempDirectory + "/file"
	file, err := os.Create(fileInTempDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Attempt and fail at directory remove because of directory not empty
	err = os.Remove(tempDirectory)
	if err == nil ||
		!strings.Contains(err.Error(), "directory not empty") {
		t.Fatalf("Expecting error \"Directory not empty\", got: %s\n", err)
	}

	// Remove file in the temp directory
	err = os.Remove(fileInTempDirectory)
	if err != nil {
		t.Fatal(err)
	}

	// Since we removed the file in the directory, now it should be empty so delete it
	err = os.Remove(tempDirectory)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRename(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	projectName := client.CreateRandomProject(false)

	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	// Create directory in project
	tempDirectory := projectDirectory + "directory"
	err = os.Mkdir(tempDirectory, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Create file in directory
	fileInTempDirectory := projectDirectory + "file"
	file, err := os.Create(fileInTempDirectory)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	// Rename (move) file into directory
	newFileName := tempDirectory + "/file"
	err = os.Rename(fileInTempDirectory, newFileName)
	if err != nil {
		t.Fatal(err)
	}

	// Rename directory in project
	newDirectoryName := projectDirectory + "newDirectory"
	err = os.Rename(tempDirectory, newDirectoryName)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRenameToAlreadyExistsFile(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()
	//fmt.Println(client.token)
	projectName := client.CreateRandomProject(false)

	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	// Create file in directory
	firstFile := projectDirectory + "firstFile"
	file, err := os.Create(firstFile)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	// Create file in directory
	secondFile := projectDirectory + "secondFile"
	file, err = os.Create(secondFile)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	err = os.Rename(firstFile, secondFile)
	if err != nil {
		t.Fatal(err)
	}
	//TODO: Test to make sure we get error: for mv: cannot overwrite non-directory `2' with directory `test1'
	// When moving directory into file
}

func TestWriteOutsideFileBoundary(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	projectName := client.CreateRandomProject(false)

	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	filePath := projectDirectory + "test"
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Seek outside file boundary
	file.Seek(10, 0)

	// Write
	writeLength := rand.Intn(48600)
	writeContents := RandomByteSlice(writeLength)
	n, err := file.Write(writeContents)
	if err != nil || n != writeLength {
		t.Fatalf("Failed to write. Wrote %d of %d bytes. Error: %s\n", n, writeLength, err)
	}
}

const testGitRepoPyCompute = "https://github.com/hesamrabeti/PyCompute"

//const testGitRepo = "https://github.com/dotcloud/docker.git"

//const testGitRepo = "https://github.com/GoodBoyDigital/pixi.js.git"

func TestGitClonePyCompute(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	// Create project
	projectName := client.CreateRandomProject(false)
	projectDirectory := fmt.Sprintf("%s/%s/%s/", mount.Dir, client.username, projectName)

	// Run clone command in project directory on oxygen-fs
	cmd := exec.Command("git", "clone", testGitRepoPyCompute)
	cmd.Dir = projectDirectory
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Error while cloning repo to oxygen-fs: %s\n", err)
	}

	// Clone repo onto local disk
	expectedDirectory, err := ioutil.TempDir("", "oxygenfstest")
	if err != nil {
		t.Fatalf("Error while creating temp dir for cloning repo: %s\n", err)
	}
	cmd = exec.Command("git", "clone", testGitRepoPyCompute)
	cmd.Dir = expectedDirectory
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Error while cloning repo to local disk: %s\n", err)
	}

	// Verify contents of repo with local cloned copy
	DirectoryEqual(projectDirectory, expectedDirectory, t)
}

func DirectoryEqual(dir, expectedDir string, t *testing.T) {
	expectedDirEntries, err := ioutil.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("Unable to read directory %s: %s", expectedDir, err)
	}

	dirEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Unable to read directory %s: %s", dir, err)
	}

	if len(expectedDirEntries) != len(dirEntries) {
		t.Fatalf("Number of entries in directory do not match expected value. Got %d, Expected %d.",
			len(dirEntries), len(expectedDirEntries))
	}

outerFor:
	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ".git" {
			continue outerFor
		}

		fileName := AddTrailingSlash(dir) + dirEntry.Name()

		// Recursively verify directories
		if dirEntry.IsDir() {
			DirectoryEqual(fileName, AddTrailingSlash(expectedDir)+dirEntry.Name(), t)
			continue
		}

		for _, expectedDirEntry := range expectedDirEntries {
			if expectedDirEntry.Name() == dirEntry.Name() {
				expectedFileName := AddTrailingSlash(expectedDir) + expectedDirEntry.Name()
				if expectedDirEntry.Size() == dirEntry.Size() {
					FileEqual(fileName, expectedFileName, t)
				} else {
					t.Errorf("File has wrong size %s: Got %d, Expected %d.\n", expectedFileName,
						dirEntry.Size(), expectedDirEntry.Size())
				}
				continue outerFor
			}
		}

		t.Errorf("Could not find entry: %s\n", fileName)
	}
}

func FileEqual(fileName, expectedFileName string, t *testing.T) {
	file, err := os.Open(fileName)
	if err != nil {
		t.Errorf("Error while opening file %s: %s\n", fileName, err)
	}
	defer file.Close()

	expectedFile, err := os.Open(expectedFileName)
	if err != nil {
		t.Errorf("Error while opening file %s: %s\n", expectedFileName, err)
	}
	defer expectedFile.Close()

	index := 0
	buf := make([]byte, 4096)
	expectedBuf := make([]byte, 4096)
	for {
		fn, ferr := file.Read(buf)
		en, eerr := expectedFile.Read(expectedBuf)
		if ferr != eerr {
			t.Fatalf("Errors do not match: \n%s -> %s\n%s -> %s\n", fileName, ferr,
				expectedFileName, eerr)
		}
		if fn != en {
			t.Fatalf("Errors do not match: \n%s -> %d\n%s -> %d\n", fileName, fn,
				expectedFileName, en)
		}
		if !bytes.Equal(buf[:fn], expectedBuf[:en]) {
			t.Fatalf("Buffers did not match. %s and %s.", fileName, expectedFileName)
		}
		if ferr != nil {
			if ferr != io.EOF {
				t.Fatal(ferr)
			}
			return
		}
		index++
	}
}

func aTestWriteLongFile(t *testing.T) {
	// Create client and user
	client := NewTitaniumClient(TestTitaniumEndpoint)
	client.CreateRandomUser()

	mount, err := CreateMountServeOxygenFSInTempDir(client.token, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mount.Close()

	projectName := client.CreateRandomProject(false)
	rootDirectory := mount.Dir
	userDirectory := fmt.Sprintf("%s/%s/", rootDirectory, client.username)
	projectDirectory := fmt.Sprintf("%s%s/", userDirectory, projectName)

	// Write
	testSeconds := 30
	totalByteMutex := sync.Mutex{}
	totalBytes := 0

	//startTime := time.Now()
	stop := make(chan bool)
	go func() {
		lastTotalBytes := totalBytes
		for i := 0; i < testSeconds; i++ {
			lastTotalBytes = totalBytes
			time.Sleep(time.Second)

			bytesThisSecond := (float32(totalBytes) - float32(lastTotalBytes)) / float32(1024*1024)
			fmt.Printf("%fMB/s\n", bytesThisSecond)
		}
		close(stop)
	}()

	buffer := make([]byte, 48600)

	writerFunc := func() {

		filePath := projectDirectory + strconv.FormatInt(rand.Int63(), 10)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()

	ForLoop:
		for {
			writeLength := 48600 //rand.Intn(48600)
			writeContents := buffer[:writeLength]
			n, err := file.Write(writeContents)

			totalByteMutex.Lock()
			totalBytes += writeLength
			totalByteMutex.Unlock()

			if err != nil || n != writeLength {
				t.Fatalf("Failed to write. Wrote %d of %d bytes. Error: %s\n", n, writeLength, err)
			}

			select {
			case <-stop:
				break ForLoop
			default:
			}
		}
	}

	writerFunc()
}
