package main

import (
	"fmt"
	"io"
	"net"

	"github.com/nova-chat/novaproto"
	"github.com/nova-chat/novaproto/dhellman"
	"github.com/nova-chat/novaproto/serializer"
)

type ApiMethod = uint64

const (
	MessageRelay   ApiMethod = 0xF100000000000000 // Client to server, asking to relay message
	MessageRelayed ApiMethod = 0xF200000000000001 // Server to client, sending relayed message

	MessageDHSv ApiMethod = 0x0000000000000010 // Client to server with public key
	MessageDHCl ApiMethod = 0x0000000000000021 // Server to client with public key
)

func main() {
	conn, err := net.Dial("tcp", "localhost:7777")
	if err != nil {
		panic(err)
	}
	wireStream := novaproto.NewNovaWireStreamCipher(novaproto.NewNovaWireStream(conn))
	packetStream := novaproto.NewRoutedPacketStream(novaproto.NewPacketStream(wireStream))

	clientPublic, err := dhellman.GenerateKeyPair()
	if err != nil {
		panic(err)
	}
	publicMsg, err := serializer.Marshal(dhellman.NewHelloMessage(clientPublic))
	if err != nil {
		panic(err)
	}

	ps, err := packetStream.SendPacket(novaproto.PacketHeader{
		Kind: MessageDHSv,
	})
	if err != nil {
		panic(err)
	}
	_, err = ps.Write(publicMsg)
	if err != nil {
		panic(err)
	}
	ps.Close()

	header, stream, err := packetStream.ReceivePacket()
	if err != nil {
		panic(err)
	}
	if header.Kind == MessageDHCl {
		data, err := io.ReadAll(stream)
		if err != nil {
			panic(err)
		}
		var serverHello dhellman.HelloMessage
		err = serializer.Unmarshal(data, &serverHello)
		if err != nil {
			panic(err)
		}

		shared, err := clientPublic.ComputeShared(serverHello.PublicKey)
		if err != nil {
			panic(err)
		}

		spk := clientPublic.PublicKey()
		salt := append(append([]byte{}, spk[:]...), serverHello.PublicKey[:]...)
		key, err := dhellman.DeriveKey(shared, salt, []byte("novaproto/dhellman"))
		if err != nil {
			panic(err)
		}
		fmt.Println(key)
	}

	/*
	   clientPublic, err := dhellman.GenerateKeyPair()

	   	if err != nil {
	   		panic(err)
	   	}

	   publicMsg, err := serializer.Marshal(dhellman.NewHelloMessage(clientPublic))

	   	if err != nil {
	   		panic(err)
	   	}

	   	wireData, err := c2s.EncodePlain(&c2s.NovaServerPacket{
	   		Meta: c2s.Metadata{
	   			MessageType: 0,
	   		},
	   		Payload: publicMsg,
	   	})

	   	if err != nil {
	   		panic(err)
	   	}

	   _, err = conn.Write(wireData)

	   	if err != nil {
	   		panic(err)
	   	}

	   var frame []byte

	   _, err = conn.Read(frame)

	   	if err != nil {
	   		panic(err)
	   	}

	   packet, err := c2s.DecodePlain(frame)

	   	if err != nil {
	   		panic(err)
	   	}

	   var serverHello dhellman.HelloMessage
	   err = serializer.Unmarshal(packet.Payload, &serverHello)

	   	if err != nil {
	   		panic(err)
	   	}

	   // Compute shared key
	   shared, err := clientPublic.ComputeShared(serverHello.PublicKey)

	   	if err != nil {
	   		fmt.Printf("failed to compute shared: %v", err)
	   		return
	   	}

	   clientPublicKey := clientPublic.PublicKey()

	   salt := append(append([]byte{}, serverHello.PublicKey[:]...), clientPublicKey[:]...)
	   info := []byte("novaproto/dhellman/test")

	   sharedKey, err := dhellman.DeriveKey(shared, salt, info)

	   	if err != nil {
	   		fmt.Printf("failed to compute secret: %v", err)
	   		return
	   	}

	   fmt.Println(sharedKey)
	*/
}
