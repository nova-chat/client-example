package client

import (
	"context"
	"hash/fnv"
	"io"
	"novaclient/client/dto"
	"reflect"

	"github.com/nova-chat/novaproto"
	"github.com/nova-chat/novaproto/serializer"
)

type ApiMethod[T any] struct {
	value uint64
	raw   bool
}

type RawPacketHandler func(ctx context.Context, client *Client, header novaproto.PacketHeader, stream io.Reader) error
type TypePacketHandler[T any] func(ctx context.Context, client *Client, header novaproto.PacketHeader, data T) error

type iClient interface {
	addRoute(method uint64, handler RawPacketHandler)
}

func (m *ApiMethod[T]) Value() uint64 {
	return m.value
}

func (m *ApiMethod[T]) ClientRegister(client iClient, handler TypePacketHandler[T]) {
	if m.raw {
		client.addRoute(m.value, func(ctx context.Context, c *Client, header novaproto.PacketHeader, stream io.Reader) error {
			data := any(stream).(T)
			return handler(ctx, c, header, data)
		})
	} else {
		client.addRoute(m.value, func(ctx context.Context, c *Client, header novaproto.PacketHeader, stream io.Reader) error {
			var data T
			bytes, err := io.ReadAll(stream)
			if err != nil {
				return err
			}
			if err := serializer.Unmarshal(bytes, &data); err != nil {
				return err
			}
			return handler(ctx, c, header, data)
		})
	}
}

type MethodKind uint8

func IsValueMethodKind(value uint64, kind MethodKind) bool {
	return MethodKind(value>>0x38) == kind
}

const (
	KindClient2Server = MethodKind(0xF0)
	KindServer2Client = MethodKind(0xF1)
	KindClient2Client = MethodKind(0xF2)
)

type MethodAccess uint8

func IsValueMethodAccess(value uint64, access MethodAccess) bool {
	return MethodAccess(value>>0x30)&access != 0
}

const (
	AccessPublic    = MethodAccess(1 << 0)
	AccessProtected = MethodAccess(1 << 1)
)

func Method[T any](name string, kind MethodKind, access MethodAccess) ApiMethod[T] {
	hash := fnv.New64a()
	hash.Write([]byte(name))
	r := hash.Sum64()
	r = r &^ 0xFFFF_0000_0000_0000
	r |= uint64(byte(kind)) << 0x38
	r |= uint64(byte(access)) << 0x30

	method := ApiMethod[T]{
		value: r,
	}

	dtoType := reflect.TypeFor[T]()
	if dtoType.Kind() != reflect.Struct {
		if dtoType == reflect.TypeFor[io.Reader]() {
			method.raw = true
		} else {
			panic("invalid dto struct")
		}
	}

	return method
}

// List of mandatory methods mirroring server routes.
var (
	ClientDHPublic = Method[dto.DHPublic]("dh_pubic", KindClient2Server, AccessPublic)
	ServerDHPublic = Method[dto.DHPublic]("dh_pubic", KindServer2Client, AccessProtected)

	ClientEncPing = Method[dto.EncHello]("enc_hello", KindClient2Server, AccessPublic)
	ServerEncPong = Method[dto.EncHello]("enc_hello", KindServer2Client, AccessProtected)
)
