package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"oxygen-go"
)

import (
	//"code.google.com/p/rsc/fuse"
	"oxygen-fuse-fs"
)

const (
	//OxygenEndpoint = "https://oxygen.atomos.io" //TODO: Change back to normal
	OxygenEndpoint = "http://localhost:9000"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s <Mount Point> <Token>\n", os.Args[0])
	flag.PrintDefaults()
}

// Serve serves the FUSE connection by making calls to the methods
// of fs and the Nodes and Handles it makes available.  It returns only
// when the connection has been closed or an unexpected error occurs.

func ServeOxygen(endpoint, token string, log bool, c *fuse.Conn) error {
	client := oxygen.NewHttpClient(endpoint, token)
	if log {
		client.StartLogging()
	}
	fs := NewOxygenFS(client, log)

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

	if flag.NArg() != 2 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	token := flag.Arg(1)
	err = ServeOxygen(OxygenEndpoint, token, false, c)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
