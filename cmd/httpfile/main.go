package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/xpy123993/toolbox/pkg/httpfile"
)

var (
	serveFolder   = flag.String("f", ".", "Serving folder")
	listenAddress = flag.String("l", "localhost:8080", "Serving address of the HTTP file server.")
)

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot bind address %s: %v", *listenAddress, err)
		os.Exit(1)
	}
	log.Printf("serving directory `%s` on `%s`", *serveFolder, lis.Addr().String())
	http.Serve(lis, httpfile.CreateHTTPServiceMux(*serveFolder))
}
