// FUSE service loop, for servers that wish to use it.
package oxygenfuse

import (
	"bazil.org/fuse"
	"github.com/atomosio/oxygen-go"

	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"
)

const (
	ALPHANUMERIC = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

var OxygenClient oxygengo.Client

var _ = fmt.Errorf
var FuncStart time.Time

type OxygenFS struct {
	client oxygengo.Client
	log    bool

	requestInterrupts *requestsInterruptMap

	handlesMap *handlesMap
	stopChan   chan error
}

var (
	ErrNoSuchHandle  = errors.New("No such handle")
	ErrNotADirectory = errors.New("Not a directory")
	ErrJSONUnmarshal = errors.New("Error while unmarshaling JSON")
)

func NewOxygenFS(client oxygengo.Client, log bool) *OxygenFS {
	output := &OxygenFS{
		requestInterrupts: NewRequestInterruptsMap(),
		client:            client,
		log:               log,
		handlesMap:        NewHandlesMap(client, log),
		stopChan:          make(chan error),
	}
	//go startStackServer() // For Debugging
	return output
}

// Serve serves the FUSE connection by making calls to the methods
// of fs and the Nodes and Handles it makes available.  It returns only
// when the connection has been closed or an unexpected error occurs.

func ServeOxygen(endpoint, token string, log bool, c *fuse.Conn) error {
	client := oxygengo.NewHttpClient(endpoint, token)
	if log {
		client.StartLogging()
	}
	fs := NewOxygenFS(client, log)

	requestChannel := make(chan fuse.Request)
	go func() {
		for {
			req, err := c.ReadRequest()
			if err != nil {
				if err != io.EOF {
					fs.stopChan <- err
					//fmt.Printf("Stopped because %s\n", err)
					return
				}
			} else {
				requestChannel <- req
			}
		}
	}()

	for {
		select {
		case err := <-fs.stopChan:
			return err
		case req := <-requestChannel:
			go fs.processRequest(req)
		}
	}

	return nil
}

func MountAndServeOxygen(mountpoint, endpoint, token string) {
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	<-c.Ready
	err = ServeOxygen(endpoint, token, false, c)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

func startStackServer() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/printstack", printStack)
	log.Println(http.ListenAndServe(":10000", serveMux))
}

func printStack(w http.ResponseWriter, r *http.Request) {
	pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
}

func (fs *OxygenFS) processRequest(request fuse.Request) {
	requestId := request.Hdr().ID

	// Store the interrupt channel for this request
	fs.requestInterrupts.Set(requestId, make(interruptChannel))

	//fmt.Printf("%s\n", request)

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
