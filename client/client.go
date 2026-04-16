package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/nova-chat/novaproto"
	"github.com/nova-chat/novaproto/dhellman"
	"github.com/nova-chat/novaproto/serializer"
)

type Client struct {
	Id uuid.UUID

	dhPrivate *dhellman.KeyPair

	conn         net.Conn
	WireStream   *novaproto.NovaWireStreamCipher
	PacketStream *novaproto.RoutedPacketStream

	routes    map[uint64]RawPacketHandler
	routesMut sync.RWMutex
}

func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	cli := &Client{
		Id:     uuid.New(),
		conn:   conn,
		routes: make(map[uint64]RawPacketHandler),
	}
	cli.WireStream = novaproto.NewNovaWireStreamCipher(novaproto.NewNovaWireStream(conn))
	cli.PacketStream = novaproto.NewRoutedPacketStream(novaproto.NewPacketStream(cli.WireStream))

	ServerDHPublic.ClientRegister(cli, cli.dhCl)
	ServerEncPong.ClientRegister(cli, cli.encPong)

	return cli, nil
}

func (c *Client) addRoute(method uint64, handler RawPacketHandler) {
	c.routesMut.Lock()
	defer c.routesMut.Unlock()
	if _, ex := c.routes[method]; ex {
		log.Fatalf("route: %d already exist", method)
	}
	c.routes[method] = handler
}

func (c *Client) GetHandler(header novaproto.PacketHeader) (RawPacketHandler, bool) {
	c.routesMut.RLock()
	defer c.routesMut.RUnlock()
	handler, ex := c.routes[header.Kind]
	return handler, ex
}

func (c *Client) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			header, dataStream, err := c.PacketStream.ReceivePacket()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("failed to recv packet: %v", err)
				continue
			}
			if err := c.handleRequest(ctx, header, dataStream); err != nil {
				log.Printf("failed to call method %x: %v", header.Kind, err)
				continue
			}
		}
	}()
}

func (c *Client) handleRequest(ctx context.Context, header novaproto.PacketHeader, dataStream io.Reader) error {
	if IsValueMethodKind(header.Kind, KindServer2Client) || IsValueMethodKind(header.Kind, KindClient2Client) {
		handler, ex := c.GetHandler(header)
		if !ex {
			io.Copy(io.Discard, dataStream)
			return fmt.Errorf("handler for method: %x not found", header.Kind)
		}
		return handler(ctx, c, header, dataStream)
	}
	io.Copy(io.Discard, dataStream)
	return fmt.Errorf("unknown method kind signature")
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Send serializes data and sends it as a packet with the given method value.
func Send[T any](c *Client, method ApiMethod[T], data T) error {
	payload, err := serializer.Marshal(data)
	if err != nil {
		return err
	}
	ps, err := c.PacketStream.SendPacket(novaproto.PacketHeader{
		SourceID: c.Id,
		Kind:     method.Value(),
	})
	if err != nil {
		return err
	}
	if _, err := ps.Write(payload); err != nil {
		return err
	}
	return ps.Close()
}
