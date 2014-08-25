// FUSE service loop, for servers that wish to use it.

package main

import (
	"errors"
	"fmt"
	"github.com/atomosio/oxygen-fuse-fs"
	"github.com/atomosio/oxygen-go"
	"time"
)

const (
	ALPHANUMERIC = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

var OxygenClient oxygen.Client

var _ = fmt.Errorf
var FuncStart time.Time

type OxygenFS struct {
	client oxygen.Client
	log    bool

	requestInterrupts *requestsInterruptMap

	handlesMap *handlesMap
	stopChan   chan bool
}

var (
	ErrNoSuchHandle  = errors.New("No such handle")
	ErrNotADirectory = errors.New("Not a directory")
	ErrJSONUnmarshal = errors.New("Error while unmarshaling JSON")
)

func NewOxygenFS(client oxygen.Client, log bool) *OxygenFS {
	output := &OxygenFS{
		requestInterrupts: NewRequestInterruptsMap(),
		client:            client,
		log:               log,
		handlesMap:        NewHandlesMap(client, log),
		stopChan:          make(chan bool),
	}

	return output
}

func (fs *OxygenFS) processRequest(request fuse.Request) {

	requestId := request.Hdr().ID

	// Store the interrupt channel for this request
	fs.requestInterrupts.Set(requestId, make(interruptChannel))

	fmt.Printf("%s\n", request)

	switch request := request.(type) {
	default:
		// Note: To FUSE, ENOSYS means "this server never implements this request."
		// It would be inappropriate to return ENOSYS for other operations in this
		// switch that might only be unavailable in some contexts, not all.
		//fmt.Println(request)
		//debug.PrintStack()
		fs.Done(request.Hdr())
		request.RespondError(fuse.ENOSYS)

		// Init
	case *fuse.InitRequest:
		fs.HandleInitRequest(request)
	// Lookup
	case *fuse.LookupRequest:
		fs.HandleLookupRequest(request)
	// Open
	case *fuse.OpenRequest:
		fs.HandleOpenRequest(request)
	// Forget
	case *fuse.ForgetRequest:
		fs.HandleForgetRequest(request)
	// Read
	case *fuse.ReadRequest:
		fs.HandleReadRequest(request)
	// Write
	case *fuse.WriteRequest:
		fs.HandleWriteRequest(request)
	// Release
	case *fuse.ReleaseRequest:
		fs.HandleReleaseRequest(request)
	// Getattr
	case *fuse.GetattrRequest:
		fs.HandleGetattrRequest(request)
	// Setattr
	case *fuse.SetattrRequest:
		fs.HandleSetattrRequest(request)
	// Create
	case *fuse.CreateRequest:
		fs.HandleCreateRequest(request)
	// Flush
	case *fuse.FlushRequest:
		fs.HandleFlushRequest(request)
	// Fsync
	case *fuse.FsyncRequest:
		fs.HandleFsyncRequest(request)
	// Remove
	case *fuse.RemoveRequest:
		fs.HandleRemoveRequest(request)
	// Mkdir
	case *fuse.MkdirRequest:
		fs.HandleMkdirRequest(request)
	// Rename
	case *fuse.RenameRequest:
		fs.HandleRenameRequest(request)
	// Interrupt
	case *fuse.InterruptRequest:
		fs.HandleInterruptRequest(request)
	// Destroy
	case *fuse.DestroyRequest:
		fs.HandleDestroyRequest(request)
	}

	// TODO: Check to make sure every request was 'Done'ed
}
