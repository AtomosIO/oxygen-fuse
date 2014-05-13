package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

import (
	"bazil.org/fuse"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s <Mount Point>\n", os.Args[0])
	flag.PrintDefaults()
}

// Serve serves the FUSE connection by making calls to the methods
// of fs and the Nodes and Handles it makes available.  It returns only
// when the connection has been closed or an unexpected error occurs.

func ServeOxygen(c *fuse.Conn) error {
	fs := OxygenFS{
		requests: map[fuse.RequestID]*serveRequest{},
	}

	for {
		req, err := c.ReadRequest()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		go fs.processRequest(req)
	}

	return nil
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 1 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = ServeOxygen(c)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
