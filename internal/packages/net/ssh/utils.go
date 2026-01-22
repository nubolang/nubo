package ssh

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

func publicKeyFile(file string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read key file: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}
