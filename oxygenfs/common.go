package main

import (
	//"encoding/binary"
	"fmt"
	//"hash/fnv"
	//"io"
	"reflect"
	"strings"
	//"sync"
	//"syscall"
	//"time"
)

import (
	"bazil.org/fuse"
)

func Debug(msg fmt.Stringer) {
	fmt.Println(msg.String())
}

type request struct {
	Op      string
	Request *fuse.Header
	In      interface{} `json:",omitempty"`
}

func (r request) String() string {
	return r.Op
}

// An Intr is a channel that signals that a request has been interrupted.
// Being able to receive from the channel means the request has been
// interrupted.
type Intr chan struct{}

func (Intr) String() string { return "fuse.Intr" }

type serveRequest struct {
	Request fuse.Request
	Intr    Intr
}

func opName(req fuse.Request) string {
	t := reflect.Indirect(reflect.ValueOf(req)).Type()
	s := t.Name()
	s = strings.TrimSuffix(s, "Request")
	return s
}
