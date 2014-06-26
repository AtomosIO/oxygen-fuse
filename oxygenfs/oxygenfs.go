// FUSE service loop, for servers that wish to use it.

package main

import (
	//"encoding/binary"
	"fmt"
	//"hash/fnv"
	//"reflect"
	//	"runtime/debug"
	//"strings"
	//"code.google.com/p/rsc/fuse"
	"errors"
	"oxygen-fuse-fs"
	//	"os"
	"oxygen-go"
	//	"sync"
	"time"
)

var OxygenClient oxygen.Client

var _ = fmt.Errorf
var FuncStart time.Time

type OxygenFS struct {
	client oxygen.Client
	log    bool

	requestInterrupts *requestsInterruptMap

	handlesMap *handlesMap
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
	}

	return output
}

func (fs *OxygenFS) processRequest(request fuse.Request) {

	requestId := request.Hdr().ID

	// Store the interrupt channel for this request
	fs.requestInterrupts.Set(requestId, make(interruptChannel))

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
	// Flush
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
	}
	//fmt.Printf("DONE -> %s\n", request)

	// TODO: Check to make sure every request was 'Done'ed
}

//case *fuse.StatfsRequest:
//	/*s := &fuse.StatfsResponse{}
//	if fs, ok := c.fs.(FSStatfser); ok {
//		if err := fs.Statfs(r, s, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//	}
//	done(s)
//	r.Respond(s)*/

//// Node operations.
//case *fuse.GetattrRequest:
//	/*s := &fuse.GetattrResponse{}
//	if n, ok := node.(NodeGetattrer); ok {
//		if err := n.Getattr(r, s, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//	} else {
//		s.AttrValid = attrValidTime
//		s.Attr = snode.attr()
//	}
//	done(s)
//	r.Respond(s)*/

//case *fuse.SetattrRequest:
//	/*s := &fuse.SetattrResponse{}
//	if n, ok := node.(NodeSetattrer); ok {
//		if err := n.Setattr(r, s, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//		done(s)
//		r.Respond(s)
//		break
//	}

//	if s.AttrValid == 0 {
//		s.AttrValid = attrValidTime
//	}
//	s.Attr = snode.attr()
//	done(s)
//	r.Respond(s)*/

//case *fuse.WriteRequest:
//	/*shandle := c.getHandle(r.Handle)
//	if shandle == nil {
//		done(fuse.ESTALE)
//		r.RespondError(fuse.ESTALE)
//		return
//	}

//	s := &fuse.WriteResponse{}
//	if h, ok := shandle.handle.(HandleWriter); ok {
//		if err := h.Write(r, s, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//		done(s)
//		r.Respond(s)
//		break
//	}
//	done(fuse.EIO)
//	r.RespondError(fuse.EIO)*/

//case *fuse.FlushRequest:
//	/*shandle := c.getHandle(r.Handle)
//	if shandle == nil {
//		done(fuse.ESTALE)
//		r.RespondError(fuse.ESTALE)
//		return
//	}
//	handle := shandle.handle

//	if h, ok := handle.(HandleFlusher); ok {
//		if err := h.Flush(r, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//	}
//	done(nil)
//	r.Respond()*/

//case *fuse.ReleaseRequest:
//	/*shandle := c.getHandle(r.Handle)
//	if shandle == nil {
//		done(fuse.ESTALE)
//		r.RespondError(fuse.ESTALE)
//		return
//	}
//	handle := shandle.handle

//	// No matter what, release the handle.
//	c.dropHandle(r.Handle)

//	if h, ok := handle.(HandleReleaser); ok {
//		if err := h.Release(r, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//	}
//	done(nil)
//	r.Respond()*/

//case *fuse.DestroyRequest:
//	/*if fs, ok := c.fs.(FSDestroyer); ok {
//		fs.Destroy()
//	}*/
//	done(nil)
//	r.Respond()

//case *fuse.RenameRequest:
//	/*c.meta.Lock()
//	var newDirNode *serveNode
//	if int(r.NewDir) < len(c.node) {
//		newDirNode = c.node[r.NewDir]
//	}
//	c.meta.Unlock()
//	if newDirNode == nil {
//		c.debug(renameNewDirNodeNotFound{
//			Request: r.Hdr(),
//			In:      r,
//		})
//		done(fuse.EIO)
//		r.RespondError(fuse.EIO)
//		break
//	}
//	n, ok := node.(NodeRenamer)
//	if !ok {
//		done(fuse.EIO) // XXX or EPERM like Mkdir?
//		r.RespondError(fuse.EIO)
//		break
//	}
//	err := n.Rename(r, newDirNode.node, intr)
//	if err != nil {
//		done(err)
//		r.RespondError(err)
//		break
//	}
//	done(nil)
//	r.Respond()*/

//case *fuse.MknodRequest:
//	/*n, ok := node.(NodeMknoder)
//	if !ok {
//		done(fuse.EIO)
//		r.RespondError(fuse.EIO)
//		break
//	}
//	n2, err := n.Mknod(r, intr)
//	if err != nil {
//		done(err)
//		r.RespondError(err)
//		break
//	}
//	s := &fuse.LookupResponse{}
//	c.saveLookup(s, snode, r.Name, n2)
//	done(s)
//	r.Respond(s)*/

//case *fuse.FsyncRequest:
//	/*n, ok := node.(NodeFsyncer)
//	if !ok {
//		done(fuse.EIO)
//		r.RespondError(fuse.EIO)
//		break
//	}
//	err := n.Fsync(r, intr)
//	if err != nil {
//		done(err)
//		r.RespondError(err)
//		break
//	}
//	done(nil)
//	r.Respond()*/

//case *fuse.InterruptRequest:
//	/*c.meta.Lock()
//	ireq := c.req[r.IntrID]
//	if ireq != nil && ireq.Intr != nil {
//		close(ireq.Intr)
//		ireq.Intr = nil
//	}
//	c.meta.Unlock()
//	done(nil)
//	r.Respond()*/

//case *fuse.CreateRequest:
//	/*n, ok := node.(NodeCreater)
//	if !ok {
//		// If we send back ENOSYS, FUSE will try mknod+open.
//		done(fuse.EPERM)
//		r.RespondError(fuse.EPERM)
//		break
//	}MountOxygenFSInTempDir
//	s := &fuse.CreateResponse{OpenResponse: fuse.OpenResponse{Flags: fuse.OpenDirectIO}}
//	n2, h2, err := n.Create(r, s, intr)
//	if err != nil {
//		done(err)
//		r.RespondError(err)
//		break
//	}
//	c.saveLookup(&s.LookupResponse, snode, r.Name, n2)
//	s.Handle = c.saveHandle(h2, hdr.Node)
//	done(s)
//	r.Respond(s)*/

//case *fuse.AccessRequest:
//	/*if n, ok := node.(NodeAccesser); ok {
//		if err := n.Access(r, intr); err != nil {
//			done(err)
//			r.RespondError(err)
//			break
//		}
//	}
//	done(nil)
//	r.Respond()*/

//case *fuse.MkdirRequest:
//	/*s := &fuse.MkdirResponse{}
//	n, ok := node.(NodeMkdirer)
//	if !ok {
//		done(fuse.EPERM)
//		r.RespondError(fuse.EPERM)
//		break
//	}
//	n2, err := n.Mkdir(r, intr)
//	if err != nil {
//		done(err)
//		r.RespondError(err)
//		break
//	}
//	c.saveLookup(&s.LookupResponse, snode, r.Name, n2)
//	done(s)
//	r.Respond(s)*/

//case *fuse.RemoveRequest:
/*n, ok := node.(NodeRemover)
if !ok {
	done(fuse.EIO) /// XXX or EPERM?
	r.RespondError(fuse.EIO)
	break
}
err := n.Remove(r, intr)
if err != nil {
	done(err)
	r.RespondError(err)
	break
}*/
//fmt.
//r.Respond()
