package client

import (
	"strings"
	"sync"

	"github.com/kr/fs"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type (
	SFTPClient interface {
		Connect() error
		Close() error
		StatVFS(path string) (*sftp.StatVFS, error)
		Walk(root string) *fs.Walker
		SupportsStatVFS() bool
	}

	sftpClient struct {
		client    *sftp.Client
		sshClient *ssh.Client
		mu        sync.Mutex  // Mutex to protect access to the client
	}
)

func (s *sftpClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil || s.sshClient == nil {
		return nil // Already closed or never connected
	}

	if err := s.client.Close(); err != nil {
		log.WithField("when", "closing SFTP connection").Error(err)
		return err
	}
	if err := s.sshClient.Close(); err != nil {
		log.WithField("when", "closing SSH connection").Error(err)
		return err
	}

	// Set clients to nil after closing
	s.client = nil
	s.sshClient = nil
	return nil
}

func (s *sftpClient) Connect() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sshClient != nil && s.client != nil {
		return nil // Already connected
	}

	s.sshClient, err = NewSSHClient()
	if err != nil {
		return err
	}

	s.client, err = sftp.NewClient(s.sshClient)
	if err != nil {
		s.sshClient.Close()
		log.WithField("when", "opening SFTP connection").Error(err)
		return err
	}

	return nil
}

func (s *sftpClient) StatVFS(path string) (*sftp.StatVFS, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, sftp.ErrSSHFxNoConnection
	}
	return s.client.StatVFS(path)
}

func (s *sftpClient) Walk(root string) *fs.Walker {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fs.WalkFS(root, s.client)
}

func (s *sftpClient) SupportsStatVFS() bool {
	_, err := s.StatVFS("/")
	if err != nil {
		if strings.Contains(err.Error(), "SSH_FX_OP_UNSUPPORTED") {
			return false
		}
	}
	return true
}

func NewSFTPClient() SFTPClient {
	return &sftpClient{}
}
