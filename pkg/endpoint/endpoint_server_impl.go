package endpoint

import (
	"context"
	"net"

	"github.com/xpy123993/toolbox/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EndpointServer struct {
	proto.UnimplementedEndpointServer
}

type BiStream interface {
	Context() context.Context
	Send(*proto.DataTrunk) error
	Recv() (*proto.DataTrunk, error)
}

func copyStream(conn net.Conn, gs BiStream) {
	copyCtx, cancelFn := context.WithCancel(gs.Context())
	go func() {
		defer cancelFn()
		for {
			in, err := gs.Recv()
			if err != nil {
				return
			}
			if _, err := conn.Write(in.GetData()); err != nil {
				return
			}
		}
	}()
	go func() {
		defer cancelFn()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			if err := gs.Send(&proto.DataTrunk{Data: buf[:n]}); err != nil {
				return
			}
		}
	}()
	<-copyCtx.Done()
}

func (s *EndpointServer) Proxy(gs proto.Endpoint_ProxyServer) error {
	req, err := gs.Recv()
	if err != nil {
		return err
	}
	if req.GetMetadata() == nil {
		return status.Errorf(codes.InvalidArgument, "metadata must be included at the streaming head")
	}
	conn, err := net.Dial(req.Metadata.Method, req.Metadata.Address)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	defer conn.Close()
	copyStream(conn, gs)
	return nil
}
