package connections

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	crypto2 "slatessh/backend/internal/crypto"
	"slatessh/backend/internal/models"

	"golang.org/x/crypto/ssh"
)

type Service struct {
	repo   *Repository
	crypto *crypto2.Service
}

type UpsertInput struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthMethod string `json:"auth_method"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
	Notes      string `json:"notes"`
}

func NewService(repo *Repository, crypto *crypto2.Service) *Service {
	return &Service{repo: repo, crypto: crypto}
}

func (s *Service) List() ([]models.Connection, error)        { return s.repo.List() }
func (s *Service) Find(id int64) (*models.Connection, error) { return s.repo.Find(id) }
func (s *Service) Delete(id int64) error                     { return s.repo.Delete(id) }

func (s *Service) Create(input UpsertInput) (*models.Connection, error) {
	connection, err := s.toConnection(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(connection)
}

func (s *Service) Update(id int64, input UpsertInput) (*models.Connection, error) {
	connection, err := s.toConnection(input)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.Find(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errBadRequest("连接未找到。")
	}
	if input.Password == "" {
		connection.EncryptedPassword = existing.EncryptedPassword
	}
	if input.PrivateKey == "" {
		connection.EncryptedKey = existing.EncryptedKey
	}
	if input.Passphrase == "" {
		connection.EncryptedPhrase = existing.EncryptedPhrase
	}
	return s.repo.Update(id, connection)
}

func (s *Service) GetDecrypted(id int64) (*models.DecryptedConnection, error) {
	connection, err := s.repo.Find(id)
	if err != nil || connection == nil {
		return nil, err
	}
	return s.decryptConnection(*connection)
}

func (s *Service) Test(ctx context.Context, input UpsertInput) (int64, error) {
	host := strings.TrimSpace(strings.Trim(input.Host, "[]"))
	if host == "" || input.Username == "" || input.Port <= 0 {
		return 0, errBadRequest("缺少必要的连接信息 (host, port, username, auth_method)。")
	}

	authMethod := strings.ToLower(strings.TrimSpace(input.AuthMethod))
	var auth ssh.AuthMethod
	switch authMethod {
	case "password":
		auth = ssh.Password(input.Password)
	case "key":
		signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(input.PrivateKey), []byte(input.Passphrase))
		if err != nil {
			signer, err = ssh.ParsePrivateKey([]byte(input.PrivateKey))
			if err != nil {
				return 0, err
			}
		}
		auth = ssh.PublicKeys(signer)
	default:
		return 0, errBadRequest("无效的认证方式。")
	}

	started := time.Now()
	config := &ssh.ClientConfig{
		User:            input.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	address := net.JoinHostPort(host, strconv.Itoa(input.Port))
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return 0, err
	}
	_ = client.Close()
	return time.Since(started).Milliseconds(), nil
}

func (s *Service) TouchLastConnected(id int64) error { return s.repo.TouchLastConnected(id) }

func (s *Service) toConnection(input UpsertInput) (models.Connection, error) {
	host := strings.TrimSpace(strings.Trim(input.Host, "[]"))
	if host == "" || input.Username == "" || input.Port <= 0 {
		return models.Connection{}, errBadRequest("缺少必要的连接信息 (host, port, username, auth_method)。")
	}
	authMethod := strings.ToLower(strings.TrimSpace(input.AuthMethod))
	if authMethod != "password" && authMethod != "key" {
		return models.Connection{}, errBadRequest("无效的认证方式。")
	}

	encryptedPassword, err := s.crypto.Encrypt(input.Password)
	if err != nil {
		return models.Connection{}, err
	}
	encryptedKey, err := s.crypto.Encrypt(input.PrivateKey)
	if err != nil {
		return models.Connection{}, err
	}
	encryptedPhrase, err := s.crypto.Encrypt(input.Passphrase)
	if err != nil {
		return models.Connection{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = host
	}

	return models.Connection{
		Name:              name,
		Type:              "SSH",
		Host:              host,
		Port:              input.Port,
		Username:          strings.TrimSpace(input.Username),
		AuthMethod:        authMethod,
		EncryptedPassword: encryptedPassword,
		EncryptedKey:      encryptedKey,
		EncryptedPhrase:   encryptedPhrase,
		Notes:             strings.TrimSpace(input.Notes),
	}, nil
}

func (s *Service) decryptConnection(connection models.Connection) (*models.DecryptedConnection, error) {
	password, err := s.crypto.Decrypt(connection.EncryptedPassword)
	if err != nil {
		return nil, err
	}
	privateKey, err := s.crypto.Decrypt(connection.EncryptedKey)
	if err != nil {
		return nil, err
	}
	passphrase, err := s.crypto.Decrypt(connection.EncryptedPhrase)
	if err != nil {
		return nil, err
	}
	return &models.DecryptedConnection{Connection: connection, Password: password, PrivateKey: privateKey, Passphrase: passphrase}, nil
}
