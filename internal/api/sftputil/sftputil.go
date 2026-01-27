package sftputil

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// AddHostKey adds the SFTP host key to the known_hosts file.
// It uses ssh-keyscan to fetch the key.
func AddHostKey(host, port string, sshDir string) error {
	hostSpec := host
	if port != "" && port != "22" {
		hostSpec = fmt.Sprintf("[%s]:%s", host, port)
	}

	// Check if already known
	checkCmd := exec.Command("ssh-keygen", "-F", hostSpec)
	if checkCmd.Run() == nil {
		zap.S().Debugf("SFTP host %s already in known_hosts", hostSpec)
		return nil
	}

	knownHostsPath := path.Join(sshDir, "known_hosts")
	if err := os.MkdirAll(path.Dir(knownHostsPath), 0700); err != nil {
		return fmt.Errorf("failed to create ssh dir: %w", err)
	}

	keyscanArgs := []string{"-H"}
	if port != "" {
		keyscanArgs = append(keyscanArgs, "-p", port)
	}
	keyscanArgs = append(keyscanArgs, host)

	keyscanCmd := exec.Command("ssh-keyscan", keyscanArgs...)
	keyOutput, err := keyscanCmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan for host %s failed: %w", host, err)
	}

	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(keyOutput); err != nil {
		return fmt.Errorf("failed to write to known_hosts file: %w", err)
	}

	zap.S().Infof("Added SFTP host %s to known_hosts file at %s", hostSpec, knownHostsPath)
	return nil
}

// GenerateKey generates an Ed25519 key pair and saves it to the specified directory.
// Returns the private key in OpenSSH PEM format, public key in SSH format, and the full path to the private key file.
func GenerateKey(host string, sshDir string) ([]byte, []byte, string, error) {
	zap.S().Debugf("Generating ED25519 key for host %s", host)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Marshal private key to OpenSSH PEM format (requires "golang.org/x/crypto/ssh")
	// Note: ssh.MarshalPrivateKey returns a PEM block since Go 1.16+ for Ed25519?
	// Actually ssh.MarshalPrivateKey returns an *pem.Block.
	privBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(privBlock)

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create public key: %w", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)

	// Save to file
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create ssh dir: %w", err)
	}
	keyPath := path.Join(sshDir, fmt.Sprintf("id_ed25519_%s_%d", host, time.Now().Unix()))
	if err := os.WriteFile(keyPath, privPEM, 0600); err != nil {
		return nil, nil, "", fmt.Errorf("failed to write private key: %w", err)
	}
	if err := os.WriteFile(keyPath+".pub", pubBytes, 0644); err != nil {
		zap.S().Warnf("failed to write public key: %v", err)
	}

	return privPEM, pubBytes, keyPath, nil
}

// InstallKey connects to the SFTP server using a password and appends the public key to authorized_keys.
func InstallKey(host, port, user, password string, pubBytes []byte) error {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Verification assumed done via AddHostKey
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with password: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create sftp client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure .ssh directory exists
	if _, err := sftpClient.Stat(".ssh"); errors.Is(err, os.ErrNotExist) {
		if err := sftpClient.Mkdir(".ssh"); err != nil {
			return fmt.Errorf("failed to create .ssh directory: %w", err)
		}
		if err := sftpClient.Chmod(".ssh", 0700); err != nil {
			zap.S().Warnf("failed to chmod .ssh: %v", err)
		}
	}

	f, err := sftpClient.OpenFile(".ssh/authorized_keys", os.O_APPEND|os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("failed to open authorized_keys: %w", err)
	}
	defer f.Close()

	if err := f.Chmod(0600); err != nil {
		zap.S().Warnf("failed to chmod authorized_keys: %v", err)
	}

	if _, err := f.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if _, err := f.Write(pubBytes); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if _, err := f.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}

// VerifyConnection attempts to connect using the provided private key.
func VerifyConnection(host, port, user string, privPEM []byte) error {
	signer, err := ssh.ParsePrivateKey(privPEM)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), clientConfig)
	if err != nil {
		return fmt.Errorf("verification connection failed: %w", err)
	}
	defer conn.Close()

	return nil
}
