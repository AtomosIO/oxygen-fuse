// FUSE service loop, for servers that wish to use it.

package main

import (
	//"encoding/binary"
	"fmt"
	//"hash/fnv"
	//"reflect"
	//"strings"
	"sync"
	"time"
)

import (
	"bazil.org/fuse"
	//"bazil.org/fuse/fuseutil"
)

var FuncStart time.Time

type OxygenFS struct {
	requestsMutex sync.Mutex
	requests      map[fuse.RequestID]*serveRequest
}

func (fs *OxygenFS) processRequest(r fuse.Request) {
	fmt.Println(r)
	FuncStart = time.Now()

	intr := make(Intr)
	request := &serveRequest{Request: r, Intr: intr}

	//Debug(request{
	//	Op:      opName(r),
	//	Request: r.Hdr(),
	//	In:      r,
	//})

	fs.requestsMutex.Lock()
	header := r.Hdr()
	if fs.requests[header.ID] != nil {
		// This happens with OSXFUSE.  Assume it's okay and
		// that we'll never see an interrupt for this one.
		// Otherwise everything wedges.  TODO: Report to OSXFUSE?
		//
		// TODO this might have been because of missing done() calls
		intr = nil
	} else {
		fs.requests[header.ID] = request
	}
	fs.requestsMutex.Unlock()

	// Call this before responding.
	// After responding is too late: we might get another request
	// with the same ID and be very confused.
	done := func(resp interface{}) {
		/*msg := response{
			Op:      opName(r),
			Request: logResponseHeader{ID: hdr.ID},
		}
		if err, ok := resp.(error); ok {
			msg.Error = err.Error()
			if ferr, ok := err.(fuse.ErrorNumber); ok {
				errno := ferr.Errno()
				msg.Errno = errno.ErrnoName()
				if errno == err {
					// it's just a fuse.Errno with no extra detail;
					// skip the textual message for log readability
					msg.Error = ""
				}
			} else {
				msg.Errno = fuse.DefaultErrno.ErrnoName()
			}
		} else {
			msg.Out = resp
		}
		Debug(msg)*/

		fs.requestsMutex.Lock()
		delete(fs.requests, header.ID)
		fs.requestsMutex.Unlock()
	}

	fmt.Println(time.Since(FuncStart).Nanoseconds())

	switch r := r.(type) {
	default:
		// Note: To FUSE, ENOSYS means "this server never implements this request."
		// It would be inappropriate to return ENOSYS for other operations in this
		// switch that might only be unavailable in some contexts, not all.
		done(fuse.ENOSYS)
		r.RespondError(fuse.ENOSYS)

	// FS operations.
	case *fuse.InitRequest:
		s := &fuse.InitResponse{
			MaxWrite: 4096,
		}
		done(s)
		r.Respond(s)

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

	//case *fuse.RemoveRequest:
	//	/*n, ok := node.(NodeRemover)
	//	if !ok {
	//		done(fuse.EIO) /// XXX or EPERM?
	//		r.RespondError(fuse.EIO)
	//		break
	//	}
	//	err := n.Remove(r, intr)
	//	if err != nil {
	//		done(err)
	//		r.RespondError(err)
	//		break
	//	}
	//	done(nil)
	//	r.Respond()*/

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

	case *fuse.OpenRequest:
		s := &fuse.OpenResponse{Flags: fuse.OpenDirectIO}
		//TODO: fuse.RootID

		//var dest string
		//if r.Node == fuse.RootID {
		//	dest = "/"
		//} else {
		//	/*r.Flags
		//	  s.Handle*/
		//	//dest = "/" + r.Header.

		//}
		/*var h2 Handle
		if n, ok := node.(NodeOpener); ok {
			hh, err := n.Open(r, s, intr)
			if err != nil {
				done(err)
				r.RespondError(err)
				break
			}
			h2 = hh
		} else {
			h2 = node
		}
		s.Handle = c.saveHandle(h2, hdr.Node)*/
		done(s)
		r.Respond(s)

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

	//// Handle operations.
	case *fuse.ReadRequest:
		//shandle := c.getHandle(r.Handle)
		//if shandle == nil {
		//	done(fuse.ESTALE)
		//	r.RespondError(fuse.ESTALE)
		//	return
		//}
		//handle := shandle.handle

		s := &fuse.ReadResponse{Data: make([]byte, 0, r.Size)}
		if r.Dir {
			//if h, ok := handle.(HandleReadDirer); ok {
			//	if shandle.readData == nil {
			//		dirs, err := h.ReadDir(intr)
			//		if err != nil {
			//			done(err)
			//			r.RespondError(err)
			//			break
			//		}
			//		var data []byte
			//		for _, dir := range dirs {
			//			if dir.Inode == 0 {
			//				dir.Inode = c.dynamicInode(snode.inode, dir.Name)
			//			}
			//			data = fuse.AppendDirent(data, dir)
			//		}
			//		shandle.readData = data
			//	}
			//	fuseutil.HandleRead(r, s, shandle.readData)
			done(s)
			r.Respond(s)
			break
			//}
		} else {
			//if h, ok := handle.(HandleReadAller); ok {
			//	if shandle.readData == nil {
			//		data, err := h.ReadAll(intr)
			//		if err != nil {
			//			done(err)
			//			r.RespondError(err)
			//			break
			//		}
			//		if data == nil {
			//			data = []byte{}
			//		}
			//		shandle.readData = data
			//	}
			//	fuseutil.HandleRead(r, s, shandle.readData)
			//	done(s)
			//	r.Respond(s)
			//	break
			//}
			//h, ok := handle.(HandleReader)
			//if !ok {
			//	fmt.Printf("NO READ FOR %T\n", handle)
			//	done(fuse.EIO)
			//	r.RespondError(fuse.EIO)
			//	break
			//}
			//if err := h.Read(r, s, intr); err != nil {
			//	done(err)
			//	r.RespondError(err)
			//	break
			//}
		}

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

	case *fuse.LookupRequest:
		s := &fuse.LookupResponse{
			AttrValid:  0,
			EntryValid: 0,
		}

		// Not found
		//done(fuse.ENOENT)
		//r.RespondError(fuse.ENOENT)

		done(s)
		r.Respond(s)

	case *fuse.ForgetRequest:
		done(nil)
		r.Respond()
	}
}
