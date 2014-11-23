package main

import (
	"github.com/atomosio/oxygen-fuse"

	"flag"
	"fmt"
	"os"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s <Endpoint> <Mount Point> <Token>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Example:\n")
	fmt.Fprintf(os.Stderr, "  %s https://oxygen.atomos.io /home/user/oxygen AB32ABXX\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 3 {
		Usage()
		os.Exit(2)
	}

	endpoint := flag.Arg(0)
	mountpoint := flag.Arg(1)
	token := flag.Arg(2)

	if err := oxygenfuse.MountAndServeOxygen(mountpoint, endpoint, token, nil); err != nil {
		fmt.Println(err)
		panic(err)
	}
}
