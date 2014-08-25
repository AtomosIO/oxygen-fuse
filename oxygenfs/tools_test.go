package main

import (
	"errors"
	"fmt"
	"github.com/atomosio/oxygen-fuse-fs"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var _ = fmt.Print

// Mount contains information about the mount for the test to use.
type Mount struct {
	// Dir is the temporary directory where the filesystem is mounted.
	Dir string

	Conn *fuse.Conn

	// Error will receive the return value of Serve.
	Error <-chan error

	done   <-chan struct{}
	closed bool
}

// Close unmounts the filesystem and waits for fs.Serve to return. Any
// returned error will be stored in Err. It is safe to call Close
// multiple times.
func (mnt *Mount) Close() {
	if mnt.closed {
		return
	}
	mnt.closed = true
	for tries := 0; tries < 1000; tries++ {
		err := fuse.Unmount(mnt.Dir)
		if err != nil {
			// TODO do more than log?
			if tries == 999 {
				// Only log if we fail at unmount
				log.Printf("unmount error: %d %v", tries, err)
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		break
	}
	<-mnt.done
	mnt.Conn.Close()
	os.Remove(mnt.Dir)
	//log.Printf("Unmounted: %s\n", mnt.Dir)
}

// Mounted mounts the OxygenFS at a temporary directory.
//
// It also waits until the filesystem is known to be visible (OS X
// workaround).
//
// After successful return, caller must clean up by calling Close.
func CreateMountServeOxygenFSInTempDir(token string, logging bool) (*Mount, error) {
	dir, err := ioutil.TempDir("", "fusetest")
	if err != nil {
		return nil, err
	}
	//log.Printf("Mounting: %s\n", dir)
	c, err := fuse.Mount(dir)
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	serveErr := make(chan error, 1)
	mnt := &Mount{
		Dir:   dir,
		Conn:  c,
		Error: serveErr,
		done:  done,
	}
	go func() {
		defer close(done)
		serveErr <- ServeOxygen(TestOxygenEndpoint, token, logging, c)
	}()

	select {
	case <-mnt.Conn.Ready:
		if mnt.Conn.MountError != nil {
			return nil, err
		}
		return mnt, err
	case err = <-mnt.Error:
		// Serve quit early
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Serve exited early")
	}
}
