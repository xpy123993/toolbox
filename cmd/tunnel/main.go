package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/armon/go-socks5"
)

var (
	ca  = flag.String("ca", "ca.crt", "")
	crt = flag.String("crt", "cert.crt", "")
	key = flag.String("key", "cert.key", "")
)

type request struct {
	Method  string
	Address string
}

type response struct {
	Success bool
	Payload string
}

func loadCertificate(caFile, certFile, keyFile string) (*tls.Certificate, *x509.CertPool, error) {
	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, nil, err
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caBytes) {
		return nil, nil, fmt.Errorf("cannot parse certificate from %s", caFile)
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}
	return &certificate, caPool, nil
}

func handleProxyServer(address string, certificate *tls.Certificate, caPool *x509.CertPool) error {
	listener, err := tls.Listen("tcp", address, &tls.Config{
		RootCAs:      caPool,
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{*certificate},
	})
	if err != nil {
		return err
	}
	log.Printf("Serving on address %s", listener.Addr().String())
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func(clientconn net.Conn) {
			defer clientconn.Close()
			req := request{}
			if gob.NewDecoder(clientconn).Decode(&req) != nil {
				gob.NewEncoder(clientconn).Encode(response{Success: false, Payload: "invaild format"})
				return
			}
			remoteConn, err := net.Dial(req.Method, req.Address)
			if err != nil {
				gob.NewEncoder(clientconn).Encode(response{Success: false, Payload: err.Error()})
				return
			}
			defer remoteConn.Close()
			if gob.NewEncoder(clientconn).Encode(response{Success: true, Payload: remoteConn.LocalAddr().String()}) != nil {
				return
			}
			ctx, cancelFn := context.WithCancel(context.Background())
			go func() { io.Copy(clientconn, remoteConn); cancelFn() }()
			go func() { io.Copy(remoteConn, clientconn); cancelFn() }()
			<-ctx.Done()
		}(conn)
	}
}

type proxyDialer struct {
	remoteAddress string
	tlsConfig     *tls.Config
}

func (client *proxyDialer) Dial(context context.Context, network, address string) (net.Conn, error) {
	conn, err := tls.Dial("tcp", client.remoteAddress, client.tlsConfig)
	handshakeSuccess := false
	defer func() {
		if !handshakeSuccess {
			conn.Close()
		}
	}()
	if err != nil {
		return nil, err
	}
	if err := gob.NewEncoder(conn).Encode(request{Method: network, Address: address}); err != nil {
		return nil, err
	}
	resp := response{}
	if err := gob.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("remote error: %s", resp.Payload)
	}
	handshakeSuccess = true
	return conn, nil
}

func handleProxyClient(localAddress string, proxyServerAddress string, certificate *tls.Certificate, caPool *x509.CertPool) error {
	lis, err := net.Listen("tcp", localAddress)
	if err != nil {
		return err
	}
	defer lis.Close()
	remoteDialer := proxyDialer{
		remoteAddress: proxyServerAddress,
		tlsConfig: &tls.Config{
			RootCAs:            caPool,
			Certificates:       []tls.Certificate{*certificate},
			InsecureSkipVerify: true,
		},
	}
	server, err := socks5.New(&socks5.Config{
		Dial: remoteDialer.Dial,
	})
	if err != nil {
		return err
	}
	log.Printf("Serving on local address %s", lis.Addr().String())
	return server.Serve(lis)
}

func main() {
	flag.Parse()

	cert, ca, err := loadCertificate(*ca, *crt, *key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load tokens: %s\n", err.Error())
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Printf("Usage:\n tunnel [server|client] [args]\n")
		os.Exit(0)
	}

	switch flag.Arg(0) {
	case "server":
		if len(os.Args) != 2 {
			fmt.Printf("Usage:\n tunnel server [listen address]\n")
			os.Exit(1)
		}
		err = handleProxyServer(flag.Arg(1), cert, ca)
	case "client":
		if len(os.Args) != 3 {
			fmt.Printf("Usage:\n tunnel client [listen address] [remote address]\n")
			os.Exit(1)
		}
		err = handleProxyClient(flag.Arg(1), flag.Arg(2), cert, ca)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Finished with error status: %s\n", err.Error())
		os.Exit(1)
	}
}
