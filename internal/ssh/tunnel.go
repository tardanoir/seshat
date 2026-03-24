package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Tunnel struct {
	client   *ssh.Client
	listener net.Listener
}

type TunnelConfig struct {
	Host     string
	Port     int
	User     string
	KeyPath  string
	Password string

	// Remote endpoint the tunnel connects to
	RemoteHost string
	RemotePort int
}

// Open establishes an SSH connection and starts a local TCP listener that forwards traffic to remoteHost:remotePort through the SSH server.
func Open(ctx context.Context, cfg TunnelConfig) (*Tunnel, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}

	authMethods, err := buildAuth(cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh auth: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", sshAddr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", sshAddr, err)
	}

	// Bind a random local port for the tunnel.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("local listen: %w", err)
	}

	remoteAddr := fmt.Sprintf("%s:%d", cfg.RemoteHost, cfg.RemotePort)

	go func() {
		for {
			local, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			remote, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				local.Close()
				continue
			}
			go forward(local, remote)
		}
	}()

	return &Tunnel{client: client, listener: listener}, nil
}

// LocalAddr returns the address of the local end of the tunnel (e.g. 127.0.0.1:54321).
func (t *Tunnel) LocalAddr() string {
	return t.listener.Addr().String()
}

// Close tears down the tunnel.
func (t *Tunnel) Close() error {
	t.listener.Close()
	return t.client.Close()
}

func forward(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		buf := make([]byte, 32*1024)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				if _, wErr := dst.Write(buf[:n]); wErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	a.Close()
	b.Close()
}

func buildAuth(cfg TunnelConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if cfg.KeyPath != "" {
		keyPath := expandPath(cfg.KeyPath)
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("reading key %s: %w", keyPath, err)
		}
		var signer ssh.Signer
		if cfg.Password != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(cfg.Password))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("parsing key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if cfg.Password != "" && cfg.KeyPath == "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	if len(methods) == 0 {
		// Try the SSH agent or default keys as a fallback.
		for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			path := home + "/.ssh/" + name
			keyBytes, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			signer, err := ssh.ParsePrivateKey(keyBytes)
			if err != nil {
				continue
			}
			methods = append(methods, ssh.PublicKeys(signer))
			break
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method available")
	}

	return methods, nil
}

func expandPath(s string) string {
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			s = home + s[1:]
		}
	}
	return s
}
