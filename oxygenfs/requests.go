package main

import (
	"fmt"
	"github.com/atomosio/oxygen-fuse-fs"
	"github.com/atomosio/oxygen-go"
	"os"
	"runtime/debug"
	"syscall"
	"time"
)

const (
	MAX_WRITE_LENGTH = 65535
	//MAX_READ_AHEAD_LENGTH = 65535
)

var _ = fmt.Print

type DirectoryNode struct {
	Id         int64  `json:"id"`
	EntityType string `json:"type"`
	Size       int64  `json:"size"` // Bytes stored in file. Approximate number of files in directory.
}

type DirectoryResponse struct {
	Nodes map[string]DirectoryNode `json:"nodes"`
}

func (fs *OxygenFS) Done(header *fuse.Header) {
	fs.requestInterrupts.Delete(header.ID)
}

func (fs *OxygenFS) HandleInitRequest(request *fuse.InitRequest) {
	response := &fuse.InitResponse{
		MaxWrite:     MAX_WRITE_LENGTH,
		MaxReadahead: request.MaxReadahead,
	}
	fs.Done(request.Hdr())
	request.Respond(response)
}

func (fs *OxygenFS) HandleLookupRequest(request *fuse.LookupRequest) {
	currentNode := int64(request.Header.Node)

	nodeAttr, err := fs.client.ResolvePathFromNode(currentNode, request.Name)
	if err != nil {
		// Not found
		fs.Done(request.Hdr())
		//debug.PrintStack(); fmt.Println(request);

		request.RespondError(fuse.ENOENT)
		return
	}

	response := fs.createLookupResponse(nodeAttr)

	fs.Done(request.Hdr())
	request.Respond(&response)
}

func (fs *OxygenFS) HandleOpenRequest(request *fuse.OpenRequest) {
	handle, err := fs.handlesMap.OpenNode(request.Node, request.Dir, request.Flags)
	if err != nil {
		// TODO: Check for errors

		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)

		request.RespondError(fuse.ENOENT)
		return
	}

	response := fs.createOpenResponse(handle.id)
	fs.Done(request.Hdr())
	request.Respond(&response)
}

func (fs *OxygenFS) HandleForgetRequest(request *fuse.ForgetRequest) {
	if request.Node == 1 {
		// If we are forgetting root, just shut down
		fs.Stop()
	}
	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleReadRequest(request *fuse.ReadRequest) {
	// Get handle for this request
	currentHandle := fs.handlesMap.GetHandle(request.Handle)
	if currentHandle == nil || currentHandle.dir != request.Dir {

		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)

		request.RespondError(fuse.ESTALE)
		return
	}

	// Make sure handle was opened for reading
	if !currentHandle.readable {

		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)

		request.RespondError(fuse.EPERM)
		return
	}

	// Perform read
	data, err := currentHandle.Read(request.Offset, request.Size)
	switch err {
	case nil, oxygen.ErrRangeNotSatisfiable:
		break
	case oxygen.ErrNotEnoughPermissions:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	default:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EIO)
		return
	}

	// Respond
	fs.Done(request.Hdr())
	request.Respond(&fuse.ReadResponse{Data: data})
}

func (fs *OxygenFS) HandleReleaseRequest(request *fuse.ReleaseRequest) {
	handle := fs.handlesMap.GetHandle(request.Handle)
	handle.Flush()
	handle.Finalize()
	fs.handlesMap.DeleteHandle(request.Handle)

	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleGetattrRequest(request *fuse.GetattrRequest) {
	currentNode := int64(request.Header.Node)
	nodeAttr, err := fs.client.ResolveNode(currentNode)
	if err != nil {
		// Not found
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(err)
		fmt.Println(request)
		request.RespondError(fuse.ENOENT)
		return
	}

	fs.Done(request.Hdr())
	request.Respond(&fuse.GetattrResponse{
		Attr: fuse.Attr{
			Inode: uint64(nodeAttr.Id),
			Mode:  SetMode(nodeAttr),
			Size:  uint64(nodeAttr.Size),
			Atime: time.Now(),
			Mtime: time.Now(),
			Ctime: time.Now(),

			//TODO: Implement Size and Atime
		},
	})
}

// Setattr does not actually set any attributes, does the same exact thing as Getattr
func (fs *OxygenFS) HandleSetattrRequest(request *fuse.SetattrRequest) {
	currentNode := int64(request.Header.Node)
	nodeAttr, err := fs.client.ResolveNode(currentNode)
	if err != nil {
		// Not found
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)

		request.RespondError(fuse.ENOENT)
		return
	}

	fs.Done(request.Hdr())
	request.Respond(&fuse.SetattrResponse{
		Attr: fuse.Attr{
			Inode: uint64(nodeAttr.Id),
			Mode:  SetMode(nodeAttr),
			Size:  uint64(nodeAttr.Size),
			Atime: time.Now(),
			Mtime: time.Now(),
			Ctime: time.Now(),

			//TODO: Implement Size and Atime
		},
	})
}

func SetMode(nodeAttr *oxygen.NodeAttributes) (mode os.FileMode) {
	if nodeAttr.Type == oxygen.DIRECTORY {
		mode = mode | os.ModeDir
	}
	mode = mode | os.ModePerm
	return
}

func (fs *OxygenFS) HandleCreateRequest(request *fuse.CreateRequest) {
	handle, err := fs.handlesMap.CreateFile(request.Node, request.Name, request.Flags)
	switch err {
	case nil:
		break
	case oxygen.ErrNotEnoughPermissions:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	default:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		fmt.Println(err)
		request.RespondError(fuse.EIO)
		return
	}

	fs.Done(request.Hdr())
	request.Respond(&fuse.CreateResponse{
		LookupResponse: fs.createLookupResponse(handle.attr),
		OpenResponse:   fs.createOpenResponse(handle.id),
	})
}

func (fs *OxygenFS) HandleFlushRequest(request *fuse.FlushRequest) {
	fs.handlesMap.GetHandle(request.Handle).Flush()

	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleFsyncRequest(request *fuse.FsyncRequest) {
	fs.handlesMap.GetHandle(request.Handle).Flush()

	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleWriteRequest(request *fuse.WriteRequest) {
	// Get handle for this request
	currentHandle := fs.handlesMap.GetHandle(request.Handle)
	if currentHandle == nil || currentHandle.dir == true {
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.ESTALE)
		return
	}

	// Make sure handle was opened for writing
	if !currentHandle.writable {
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	}

	// Perform write
	n, err := currentHandle.Write(request.Data, request.Offset)
	switch err {
	case nil:
		break
	case oxygen.ErrNotEnoughPermissions:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	default:
		fmt.Println(err)
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EIO)
		return
	}

	// Respond
	fs.Done(request.Hdr())
	request.Respond(&fuse.WriteResponse{Size: n})
}

func (fs *OxygenFS) HandleRemoveRequest(request *fuse.RemoveRequest) {
	currentNode := int64(request.Header.Node)
	err := fs.client.DeleteFromNode(currentNode, request.Name)
	if err == oxygen.ErrDirectoryNotEmpty {
		// Directory not empty
		fs.Done(request.Hdr())
		request.RespondError(fuse.Errno(syscall.ENOTEMPTY))
		return
	} else if err != nil {
		// Not found
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.ENOENT)
		return
	}

	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleMkdirRequest(request *fuse.MkdirRequest) {
	attr, err := fs.handlesMap.CreateDir(request.Node, request.Name, request.Mode)
	switch err {
	case nil:
	case oxygen.ErrNotEnoughPermissions:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	default:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EIO)
		return
	}

	fs.Done(request.Hdr())
	request.Respond(&fuse.MkdirResponse{
		LookupResponse: fuse.LookupResponse{
			Node:       fuse.NodeID(attr.Id),
			EntryValid: time.Second,
			AttrValid:  time.Second,
			Attr: fuse.Attr{
				Inode: uint64(attr.Id),
				Mode:  request.Mode,
				Size:  uint64(attr.Size),
				Atime: time.Now(),
				Mtime: time.Now(),
				Ctime: time.Now(),
			},
		},
	})
}

func (fs *OxygenFS) HandleRenameRequest(request *fuse.RenameRequest) {
	err := fs.handlesMap.client.RenameFromNodeToNode(int64(request.Node), request.OldName,
		int64(request.NewDir), request.NewName)

	switch err {
	case nil:
	case oxygen.ErrNotEnoughPermissions:
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EPERM)
		return
	default:
		//fmt.Println(err)
		fs.Done(request.Hdr())
		debug.PrintStack()
		fmt.Println(request)
		request.RespondError(fuse.EIO)
		return
	}

	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleInterruptRequest(request *fuse.InterruptRequest) {
	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) HandleDestroyRequest(request *fuse.DestroyRequest) {
	fs.Stop()
	fs.Done(request.Hdr())
	request.Respond()
}

func (fs *OxygenFS) Stop() {
	fs.stopChan <- true
}

func (fs *OxygenFS) createLookupResponse(nodeAttr *oxygen.NodeAttributes) fuse.LookupResponse {
	// Create response structure
	return fuse.LookupResponse{
		Node:       fuse.NodeID(nodeAttr.Id),
		EntryValid: time.Second,
		AttrValid:  time.Second,
		Attr: fuse.Attr{
			Inode: uint64(nodeAttr.Id),
			Mode:  SetMode(nodeAttr),
			Size:  uint64(nodeAttr.Size),
			Atime: time.Now(),
			Mtime: time.Now(),
			Ctime: time.Now(),
		}}
}

func (fs *OxygenFS) createOpenResponse(handleId fuse.HandleID) fuse.OpenResponse {
	return fuse.OpenResponse{
		//Flags:  fuse.OpenDirectIO,
		Handle: handleId,
	}
}
