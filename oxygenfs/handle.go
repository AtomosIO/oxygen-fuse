package main

import (
	//"code.google.com/p/rsc/fuse"
	"oxygen-fuse-fs"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"oxygen-go"
	"sync"
	"syscall"
)

var _ = fmt.Print

const (
	pipeReaderToWriterBufferSize = 1024
)

var (
	ErrPrematureWriteClosed = errors.New("Write channel closed prematurely")
)

type handle struct {
	sync.Mutex
	id fuse.HandleID

	nodeId int64
	dir    bool

	// Was the file opened with read permission?
	readable bool
	// Was the file opened with write permission?
	writable bool
	// Was the file opened for synchronous IO?
	synchronous bool
	// Was file opened in append mode?
	appending bool
	// Was file opened with truncate?
	truncate bool

	// A new node does not need to read old data therefore saving 2 GETS
	// Usual: (Get prefix data, write data, get postfix data)
	// With New Node: (write data)
	newNode bool

	// Determines whether a write is neccesary on flush
	data []byte

	trackingReadCloser  *TrackingReadCloser
	trackingWriteCloser *TrackingWriteCloser

	writerCloseChan chan writerCloseMessage

	handlesMap *handlesMap

	attr *oxygen.NodeAttributes

	refs int
}

type writerCloseMessage struct {
	attr *oxygen.NodeAttributes
	err  error
}

type handlesMap struct {
	client oxygen.Client

	sync.RWMutex
	handles         map[fuse.HandleID]*handle
	nodes           map[fuse.NodeID]*handle
	currentHandleId fuse.HandleID

	log bool
}

func NewHandlesMap(client oxygen.Client, log bool) *handlesMap {
	return &handlesMap{
		handles: make(map[fuse.HandleID]*handle),
		nodes:   make(map[fuse.NodeID]*handle),
		client:  client,
		log:     log,
	}
}

func (handlesMap *handlesMap) Logf(format string, args ...interface{}) {
	if handlesMap.log {
		fmt.Printf(format, args...)
	}
}

func FlagCreateSet(flags fuse.OpenFlags) bool {
	return (flags & fuse.OpenFlags(os.O_CREATE)) != 0
}
func FlagExclusiveSet(flags fuse.OpenFlags) bool {
	return (flags & fuse.OpenFlags(os.O_EXCL)) != 0
}
func FlagTruncateSet(flags fuse.OpenFlags) bool {
	return (flags & fuse.OpenFlags(os.O_TRUNC)) != 0
}
func FlagSyncSet(flags fuse.OpenFlags) bool {
	return (flags & fuse.OpenFlags(os.O_SYNC)) != 0
}
func FlagAppendSet(flags fuse.OpenFlags) bool {
	return (flags & fuse.OpenFlags(os.O_APPEND)) != 0
}
func FlagReadSet(flags fuse.OpenFlags) bool {
	f := uint32(flags) & syscall.O_ACCMODE
	return f == uint32(os.O_RDONLY) || f == uint32(os.O_RDWR)
}
func FlagWriteSet(flags fuse.OpenFlags) bool {
	f := uint32(flags) & syscall.O_ACCMODE
	return f == uint32(os.O_WRONLY) || f == uint32(os.O_RDWR)
}

func (handlesMap *handlesMap) NewHandle(nodeId fuse.NodeID, dir bool, flags fuse.OpenFlags) *handle {
	newHandle := &handle{
		dir:                 dir,
		readable:            FlagReadSet(flags),
		writable:            FlagWriteSet(flags),
		synchronous:         FlagSyncSet(flags),
		appending:           FlagAppendSet(flags),
		truncate:            FlagTruncateSet(flags),
		nodeId:              int64(nodeId),
		handlesMap:          handlesMap,
		writerCloseChan:     make(chan writerCloseMessage),
		trackingReadCloser:  &TrackingReadCloser{},
		trackingWriteCloser: &TrackingWriteCloser{},
		refs:                1,
	}

	handlesMap.Lock()

	handleId := handlesMap.currentHandleId
	handlesMap.handles[handleId] = newHandle
	handlesMap.nodes[nodeId] = newHandle
	handlesMap.currentHandleId++

	handlesMap.Unlock()

	newHandle.id = handleId
	//handlesMap.Logf("New Handle: %d\n%+v\n", handleId, newHandle)
	return newHandle
}

func (handlesMap *handlesMap) GetHandle(handleId fuse.HandleID) *handle {
	handlesMap.RLock()
	value := handlesMap.handles[handleId]
	handlesMap.RUnlock()
	//handlesMap.Logf("Get Handle: %d\n", handleId)

	return value
}

func (handlesMap *handlesMap) DeleteHandle(handleId fuse.HandleID) {
	handle := handlesMap.GetHandle(handleId)
	handle.Finalize()

	// Reduce ref count on handle
	handle.Lock()
	handle.refs--
	if handle.refs == 0 {
		handlesMap.Lock()
		delete(handlesMap.handles, handleId)
		delete(handlesMap.nodes, fuse.NodeID(handle.nodeId))
		handlesMap.Unlock()
	}
	handle.Unlock()
}

func (handlesMap *handlesMap) OpenNode(nodeId fuse.NodeID, dir bool, flags fuse.OpenFlags) (handle *handle, err error) {
	handlesMap.RLock()
	handle, has := handlesMap.nodes[nodeId]
	handlesMap.RUnlock()
	if has {
		fmt.Printf("opened already open node %d\n", nodeId)

		handle.Lock()
		handle.refs++
		handle.Unlock()

		return handle, nil
	}

	attr, err := handlesMap.client.ResolveNode(int64(nodeId))
	if err != nil {
		return nil, err
	}

	handle = handlesMap.NewHandle(nodeId, dir, flags)
	handle.attr = attr

	return handle, err
}

func (handlesMap *handlesMap) CreateFile(nodeId fuse.NodeID, name string, flags fuse.OpenFlags) (handle *handle, err error) {
	var attr *oxygen.NodeAttributes
	if FlagCreateSet(flags) && FlagExclusiveSet(flags) {
		attr, err = handlesMap.client.CreatePathFromNode(int64(nodeId), name, NewEmptyReader())
	} else {
		attr, err = handlesMap.client.OverwritePathFromNode(int64(nodeId), name, 0, NewEmptyReader())
	}

	if err != nil {
		return nil, err
	}

	handle = handlesMap.NewHandle(fuse.NodeID(attr.Id), false, flags)
	handle.attr = attr
	handle.newNode = true

	return handle, err
}

func (handlesMap *handlesMap) CreateDir(nodeId fuse.NodeID, name string, mode os.FileMode) (attr *oxygen.NodeAttributes, err error) {
	if name[len(name)-1:] != "/" {
		name = name + "/"
	}

	attr, err = handlesMap.client.CreatePathFromNode(int64(nodeId), name, NewEmptyReader())
	if err != nil {
		return nil, err
	}

	return attr, err
}

// Prepares this handle to be released (called when file descriptor is closed)
func (handle *handle) Finalize() {
}

func (handle *handle) populateDirectoryEntries() error {
	// Get the node data
	//TODO: Limit the maximum number of entries read per call
	attr, reader, err := handle.handlesMap.client.ReadNode(handle.nodeId, 0, -1)
	if err != nil {
		return err
	}
	defer reader.Close()

	// FUSE says we're reading a directory, make sure node is actually a directory
	if attr.Type != oxygen.DIRECTORY {
		return ErrNotADirectory
	}

	// Read whole body
	respBody, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	// Unmarshal body into a response variable
	directoryResponse := &DirectoryResponse{}
	err = json.Unmarshal(respBody, directoryResponse)
	if err != nil {
		return err
	}

	var data []byte
	for name, node := range directoryResponse.Nodes {
		var entityType fuse.DirentType
		switch node.EntityType {
		case "directory":
			entityType = fuse.DT_Dir
		case "file":
			entityType = fuse.DT_File
		}
		dirEntry := fuse.Dirent{
			Name:  name,
			Inode: uint64(node.Id), //TODO: Do we need to send inode id?
			Type:  entityType,
		}
		data = fuse.AppendDirent(data, dirEntry)
	}

	handle.data = data
	return nil
}

// Read from directory or file. Seeking while reading from a file causes a new HTTP
// GET request, increasing latency.
func (handle *handle) Read(offset int64, size int) ([]byte, error) {
	handle.Lock()
	defer handle.Unlock()

	return handle.read(offset, size)
}

// Seeking while writing will cause huge amounts of latency
func (handle *handle) Write(data []byte, offset int64) (int, error) {
	handle.Lock()
	defer handle.Unlock()

	return handle.write(data, offset)
}

func (handle *handle) Flush() error {
	handle.Lock()
	defer handle.Unlock()

	return handle.flush()
}

func (handle *handle) read(offset int64, size int) ([]byte, error) {
	if handle.dir {
		return handle.readDir(offset, size)
	}

	return handle.readFile(offset, size)
}

func (handle *handle) readDir(offset int64, size int) ([]byte, error) {
	if len(handle.data) == 0 {
		if err := handle.populateDirectoryEntries(); err != nil {
			return nil, err
		}
	}

	if offset > int64(len(handle.data)) {
		// "Empty" data. This keeps the underlying array for later use by another
		// directory call. When the handle is released it will be reclaimed.
		handle.data = handle.data[:0]
		return []byte{}, nil
	}

	output := handle.data[offset:]
	if len(output) > size {
		output = output[:size]
	}

	return output, nil
}

func (handle *handle) readFile(offset int64, size int) ([]byte, error) {
	// Seek writer to 0 so we force a flush
	if err := handle.flush(); err != nil {
		return []byte{}, err
	}

	// Open a reader if we don't have it open already or it's at the wrong offset
	if err := handle.seekReader(offset, -1); err != nil {
		return []byte{}, err
	}

	output := make([]byte, size)
	n, err := io.ReadFull(handle.trackingReadCloser, output)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return []byte{}, err
	}

	return output[:n], nil
}

func (handle *handle) seekReader(offset int64, size int) error {
	if offset < 0 || handle.trackingReadCloser.offset < 0 {
		panic("handle.seekWriter: offset < 0")
	}

	if handle.trackingReadCloser.readCloser == nil || handle.trackingReadCloser.offset != offset {
		attr, data, err := handle.handlesMap.client.ReadNode(handle.nodeId, offset, size)
		if err != nil {
			return err
		}

		handle.attr = attr
		handle.trackingReadCloser.NewReader(data, offset)
	}

	return nil
}

func (handle *handle) write(data []byte, offset int64) (int, error) {
	if err := handle.seekWriter(offset); err != nil {
		return 0, err
	}

	// If writeCloseChan has a message, the writer has closed. Send an error.
	select {
	case <-handle.writerCloseChan:
		return 0, ErrPrematureWriteClosed
	default:
	}

	return handle.trackingWriteCloser.Write(data)
}

func (handle *handle) seekWriter(offset int64) error {
	if offset < 0 || handle.trackingWriteCloser.offset < 0 {
		panic("handle.seekWriter: offset < 0")
	}

	// Since we only seek for writes (reads flush() to clean up writer), we will need
	// a writer so we have to initialize writer.
	if handle.trackingWriteCloser.writeCloser == nil {
		handle.newWriter(0)
	}

	// currentOffset == offset -> No Op
	if handle.trackingWriteCloser.offset == offset {
		return nil
	}

	// We're seeking behind the current location, need to save what we have and start
	// over
	if handle.trackingWriteCloser.offset > offset {
		// Flush will save the rest of the file, close writer, and set writer to nil
		handle.flush()
		// Create new writer
		handle.newWriter(0)
	}

	// By this point, offset must be equal to or larger than current writer offset
	seekSize := offset - handle.trackingWriteCloser.offset
	if seekSize > 0 {
		// We have to seek forward, so write the contents of the file in the bytes we
		// will be skipping.
		if err := handle.seekReader(handle.trackingWriteCloser.offset, int(seekSize)); err != nil {
			if err != oxygen.ErrRangeNotSatisfiable {
				return err
			}

			// Reader was not able to seek, create a 0 filling reader
			zeroReader := ioutil.NopCloser(io.LimitReader(NewZeroReader(), seekSize))
			handle.trackingReadCloser.NewReader(zeroReader, offset)
		}

		if err := handle.pipeReaderToWriter(); err != nil {
			return err
		}
	}

	return nil
}

func (handle *handle) newWriter(offset int64) {
	handle.writerCloseChan = make(chan writerCloseMessage)
	preadCloser, pwriteCloser := io.Pipe()
	handle.trackingWriteCloser.NewWriter(pwriteCloser, offset)

	go func() {
		attr, err := handle.handlesMap.client.OverwriteNode(handle.nodeId, offset, preadCloser)
		handle.writerCloseChan <- writerCloseMessage{
			attr: attr,
			err:  err,
		}
	}()

}

func (handle *handle) flush() error {
	if handle.trackingWriteCloser.writeCloser != nil {
		if !handle.newNode {
			// Write the rest of the file
			seekOffset := handle.trackingWriteCloser.offset
			if err := handle.seekReader(seekOffset, -1); err == nil {
				handle.pipeReaderToWriter()
			}
		}

		handle.trackingWriteCloser.Close()

		// Wait for writer goroutine to send back the results of the OverwriteNode
		msg := <-handle.writerCloseChan
		if msg.err != nil {
			return msg.err
		}

		handle.attr = msg.attr
		handle.trackingWriteCloser.writeCloser = nil
	}

	return nil
}

func (handle *handle) pipeReaderToWriter() error {
	buf := make([]byte, pipeReaderToWriterBufferSize)
	done := false
	var outputError error
	for !done {
		n, err := handle.trackingReadCloser.Read(buf)
		if err != nil {
			if err != io.EOF {
				outputError = err
			}
			done = true
		}

		_, err = handle.trackingWriteCloser.Write(buf[:n])
		if err != nil {
			outputError = err
			done = true
		}
	}

	return outputError
}

/*func (handle *handle) GetAttributes() {

}*/
