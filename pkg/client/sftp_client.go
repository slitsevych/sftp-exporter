package client

import (
	"strings"

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
    SupportsStatVFS() bool  // Add this line
	}

	sftpClient struct {
		*sftp.Client
		sshClient *ssh.Client
	}
)

func (s *sftpClient) Close() error {
	if err := s.Client.Close(); err != nil {
		log.WithField("when", "closing SFTP connection").Error(err)
		return err
	}
	if err := s.sshClient.Close(); err != nil {
		log.WithField("when", "closing SSH connection").Error(err)
		return err
	}
	return nil
}

func (s *sftpClient) Connect() (err error) {
	s.sshClient, err = NewSSHClient()
	if err != nil {
		return err
	}

	s.Client, err = sftp.NewClient(s.sshClient)
	if err != nil {
		if err := s.sshClient.Close(); err != nil {
			log.WithField("when", "opening SFTP connection").Error(err)
		}
		return err
	}
	return nil
}

func (c *sftpClient) SupportsStatVFS() bool {
	// Try a StatVFS on a known path and see if it returns an error
	_, err := c.StatVFS("/")
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
