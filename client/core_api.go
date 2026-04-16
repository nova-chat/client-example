package client

import (
	"context"
	"fmt"
	"novaclient/client/dto"

	"github.com/nova-chat/novaproto"
	"github.com/nova-chat/novaproto/dhellman"
)

func (c *Client) DHHandshake() error {
	kp, err := dhellman.GenerateKeyPair()
	if err != nil {
		return err
	}
	c.dhPrivate = kp
	return Send(c, ClientDHPublic, dto.DHPublic{PublicKey: kp.PublicKey()})
}

func (c *Client) dhCl(ctx context.Context, _ *Client, header novaproto.PacketHeader, serverHello dto.DHPublic) error {
	if c.dhPrivate == nil {
		return fmt.Errorf("dh handshake not initiated")
	}
	shared, err := c.dhPrivate.ComputeShared(serverHello.PublicKey)
	if err != nil {
		return err
	}

	clientPub := c.dhPrivate.PublicKey()
	salt := append(append([]byte{}, clientPub[:]...), serverHello.PublicKey[:]...)
	key, err := dhellman.DeriveKey(shared, salt, []byte("novaproto/dhellman"))
	if err != nil {
		return err
	}
	// Set frame encryption
	err = c.WireStream.SetKey(key)
	if err != nil {
		return err
	}

	// We got encryption key, send enc ping
	return Send(c, ClientEncPing, dto.EncHello{
		Text: "NOVA",
	})
}

func (c *Client) encPong(ctx context.Context, _ *Client, header novaproto.PacketHeader, serverHello dto.EncHello) error {
	if serverHello.Text != "NOVA" {
		return fmt.Errorf("failed to decode message")
	}
	return nil
}
