package dto

import (
	"github.com/nova-chat/novaproto/dhellman"
)

type DHPublic struct {
	PublicKey [dhellman.PublicKeySize]byte
}
