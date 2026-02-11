package encryption

import (
	"crypto/rand"
	"fmt"

	pb "github.com/golang/protobuf/proto" //nolint
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const nonceSize = 24

// EncryptMessage encrypts a body of the given protobuf Message
func EncryptMessage(remotePubKey wgtypes.Key, ourPrivateKey wgtypes.Key, message pb.Message) ([]byte, error) {
	byteResp, err := pb.Marshal(message)
	if err != nil {
		return nil, err
	}

	encryptedBytes, err := Encrypt(byteResp, remotePubKey, ourPrivateKey)
	if err != nil {
		return nil, err
	}

	return encryptedBytes, nil
}

// DecryptMessage decrypts an encrypted message into given protobuf Message
func DecryptMessage(remotePubKey wgtypes.Key, ourPrivateKey wgtypes.Key, encryptedMessage []byte, message pb.Message) error {
	decrypted, err := Decrypt(encryptedMessage, remotePubKey, ourPrivateKey)
	if err != nil {
		return err
	}

	err = pb.Unmarshal(decrypted, message)
	if err != nil {
		return err
	}
	return nil
}

// Encrypt encrypts a message using local Wireguard private key and remote peer's public key.
func Encrypt(msg []byte, peerPublicKey wgtypes.Key, privateKey wgtypes.Key) ([]byte, error) {
	nonce, err := genNonce()
	if err != nil {
		return nil, err
	}
	return box.Seal(nonce[:], msg, nonce, toByte32(peerPublicKey), toByte32(privateKey)), nil
}

// Decrypt decrypts a message that has been encrypted by the remote peer using Wireguard private key and remote peer's public key.
func Decrypt(encryptedMsg []byte, peerPublicKey wgtypes.Key, privateKey wgtypes.Key) ([]byte, error) {
	nonce, err := genNonce()
	if err != nil {
		return nil, err
	}
	if len(encryptedMsg) < nonceSize {
		return nil, fmt.Errorf("invalid encrypted message length")
	}
	copy(nonce[:], encryptedMsg[:nonceSize])
	opened, ok := box.Open(nil, encryptedMsg[nonceSize:], nonce, toByte32(peerPublicKey), toByte32(privateKey))
	if !ok {
		return nil, fmt.Errorf("failed to decrypt message from peer %s", peerPublicKey.String())
	}

	return opened, nil
}

// Generates nonce of size 24
func genNonce() (*[nonceSize]byte, error) {
	var nonce [nonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	return &nonce, nil
}

// Converts Wireguard key to byte array of size 32 (a format used by the golang crypto package)
func toByte32(key wgtypes.Key) *[32]byte {
	return (*[32]byte)(&key)
}
