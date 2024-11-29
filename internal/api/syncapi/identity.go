package syncengine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

var (
	curve = elliptic.P256() // ed25519
)

type Identity struct {
	InstanceID     string
	credentialFile string

	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

func NewIdentity(instanceID, credentialFile string) (*Identity, error) {
	i := &Identity{
		InstanceID:     instanceID,
		credentialFile: credentialFile,
	}
	if err := i.loadOrGenerateKey(); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *Identity) loadOrGenerateKey() error {
	privKeyBytes, errpriv := os.ReadFile(i.credentialFile)
	pubKeyBytes, errpub := os.ReadFile(i.credentialFile + ".pub")
	if errpriv != nil || errpub != nil {
		if os.IsNotExist(errpriv) || os.IsNotExist(errpub) {
			return i.generateKeys()
		}
		if errpriv != nil {
			return fmt.Errorf("open private key: %w", errpriv)
		}
		if errpub != nil {
			return fmt.Errorf("open public key: %w", errpub)
		}
	}

	privKeyBlock, _ := pem.Decode(privKeyBytes)
	if privKeyBlock == nil {
		return errors.New("no private key found in pem")
	}
	privKey, err := x509.ParseECPrivateKey(privKeyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	pubKeyBlock, _ := pem.Decode(pubKeyBytes)
	if pubKeyBlock == nil {
		return errors.New("no public key found in pem")
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	i.privateKey = privKey
	i.publicKey = pubKey.(*ecdsa.PublicKey)

	return nil
}

func (i *Identity) generateKeys() error {
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}

	i.privateKey = privKey
	i.publicKey = &privKey.PublicKey

	privateKeyBytes, err := x509.MarshalECPrivateKey(i.privateKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	pemPrivateKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE", Bytes: privateKeyBytes})
	if err := os.WriteFile(i.credentialFile, pemPrivateKeyBytes, 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&i.privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}
	pemPublicKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PUBLIC", Bytes: publicKeyBytes})
	if err := os.WriteFile(i.credentialFile+".pub", pemPublicKeyBytes, 0600); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}

func (i *Identity) SignMessage(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)

	sig, err := ecdsa.SignASN1(rand.Reader, i.privateKey, hash[:])
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (i *Identity) VerifySignature(message, sig []byte) error {
	hash := sha256.Sum256(message)
	if !ecdsa.VerifyASN1(i.publicKey, hash[:], sig) {
		return errors.New("signature verification failed")
	}
	return nil
}
