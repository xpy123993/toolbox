package endpoint_test

import (
	"encoding/gob"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/xpy123993/toolbox/pkg/endpoint"
	"github.com/xpy123993/toolbox/proto"
	"google.golang.org/grpc"
)

func echoLoop(t *testing.T, conn net.Conn) {
	type TestStruct struct {
		Data string
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gob.NewEncoder(conn).Encode(TestStruct{Data: "helloworld"}); err != nil {
			t.Error(err)
		}
	}()
	res := TestStruct{}
	if err := gob.NewDecoder(conn).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Data != "helloworld" {
		t.Errorf("data mismatched")
	}
	wg.Wait()
}

func TestEndpointProxy(t *testing.T) {
	server := grpc.NewServer()
	proto.RegisterEndpointServer(server, &endpoint.EndpointServer{})
	serverLis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer serverLis.Close()
	go server.Serve(serverLis)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		go io.Copy(conn, conn)
		wg.Done()
	}()

	clientConn, err := grpc.Dial(serverLis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()
	client := endpoint.NewEndpointClient(proto.NewEndpointClient(clientConn))
	peerA, peerB := net.Pipe()
	go client.Proxy("tcp", lis.Addr().String(), peerA)
	echoLoop(t, peerB)
	wg.Wait()
}
