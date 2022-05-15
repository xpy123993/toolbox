package endpoint

import (
	"context"
	"net"

	"github.com/xpy123993/toolbox/proto"
)

type EndpointClient struct {
	client proto.EndpointClient
}

func NewEndpointClient(c proto.EndpointClient) *EndpointClient {
	return &EndpointClient{client: c}
}

func (c *EndpointClient) Proxy(method string, address string, conn net.Conn) error {
	gc, err := c.client.Proxy(context.Background())
	if err != nil {
		return err
	}
	if err := gc.Send(&proto.DataTrunk{Metadata: &proto.ProxyMetadata{
		Method:  method,
		Address: address,
	}}); err != nil {
		return err
	}
	copyStream(conn, gc)
	return nil
}
