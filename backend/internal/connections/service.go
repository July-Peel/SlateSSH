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
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthMethod string `json:"auth_method"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
	Notes      string `json:"notes"`
}

// NewService 用于创建业务服务实例。
// 输入参数：repo 表示repo 参数；crypto 表示crypto 参数。
// 输出参数：返回 *Service。
func NewService(repo *Repository, crypto *crypto2.Service) *Service {
	return &Service{repo: repo, crypto: crypto}
}

// List 用于返回列表数据。
// 输入参数：无。
// 输出参数：返回 []models.Connection, error；error 表示执行失败原因。
func (s *Service) List() ([]models.Connection, error) { return s.repo.List() }

// Find 用于根据标识查询数据记录。
// 输入参数：id 表示标识符。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
func (s *Service) Find(id int64) (*models.Connection, error) { return s.repo.Find(id) }

// Delete 用于删除指定数据记录。
// 输入参数：id 表示标识符。
// 输出参数：返回 error；error 表示执行失败原因。
func (s *Service) Delete(id int64) error { return s.repo.Delete(id) }

// Create 用于创建新数据记录。
// 输入参数：input 表示连接输入参数。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
func (s *Service) Create(input UpsertInput) (*models.Connection, error) {
	connection, err := s.toConnection(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(connection)
}

// Update 用于更新已有数据记录。
// 输入参数：id 表示标识符；input 表示连接输入参数。
// 输出参数：返回 *models.Connection, error；error 表示执行失败原因。
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

// GetDecrypted 用于读取并解密连接配置。
// 输入参数：id 表示标识符。
// 输出参数：返回 *models.DecryptedConnection, error；error 表示执行失败原因。
func (s *Service) GetDecrypted(id int64) (*models.DecryptedConnection, error) {
	connection, err := s.repo.Find(id)
	if err != nil || connection == nil {
		return nil, err
	}
	return s.decryptConnection(*connection)
}

// Test 用于测试连接配置的可达性。
// 输入参数：ctx 表示上下文对象；input 表示连接输入参数。
// 输出参数：返回 int64, error；error 表示执行失败原因。
func (s *Service) Test(ctx context.Context, input UpsertInput) (int64, error) {
	input, err := s.resolveTestInput(input)
	if err != nil {
		return 0, err
	}

	host := strings.TrimSpace(strings.Trim(input.Host, "[]"))
	if host == "" || strings.TrimSpace(input.Username) == "" || input.Port <= 0 {
		return 0, errBadRequest("缺少必要的连接信息 (host, port, username)。")
	}

	connectionType := normalizeConnectionType(input.Type)
	if connectionType == "RDP" {
		started := time.Now()
		dialer := net.Dialer{Timeout: 15 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(input.Port)))
		if err != nil {
			return 0, err
		}
		_ = conn.Close()
		return time.Since(started).Milliseconds(), nil
	}

	authMethods, err := buildSSHAuthMethods(input)
	if err != nil {
		return 0, err
	}

	started := time.Now()
	config := &ssh.ClientConfig{
		User:            strings.TrimSpace(input.Username),
		Auth:            authMethods,
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

// resolveTestInput fills missing secrets from the database when testing a saved connection.
// This allows "test" on existing servers (and edit forms with blank password fields) to use
// the encrypted credentials already stored in the database.
func (s *Service) resolveTestInput(input UpsertInput) (UpsertInput, error) {
	if input.ID <= 0 {
		return input, nil
	}
	saved, err := s.GetDecrypted(input.ID)
	if err != nil {
		return input, err
	}
	if saved == nil {
		return input, errBadRequest("连接未找到。")
	}

	// Prefer explicit form values; fall back to DB for empty secrets / blank core fields.
	if strings.TrimSpace(input.Type) == "" {
		input.Type = saved.Type
	}
	if strings.TrimSpace(input.Host) == "" {
		input.Host = saved.Host
	}
	if input.Port <= 0 {
		input.Port = saved.Port
	}
	if strings.TrimSpace(input.Username) == "" {
		input.Username = saved.Username
	}
	if strings.TrimSpace(input.AuthMethod) == "" {
		input.AuthMethod = saved.AuthMethod
	}
	if input.Password == "" {
		input.Password = saved.Password
	}
	if strings.TrimSpace(input.PrivateKey) == "" {
		input.PrivateKey = saved.PrivateKey
	}
	if input.Passphrase == "" {
		input.Passphrase = saved.Passphrase
	}
	return input, nil
}

func buildSSHAuthMethods(input UpsertInput) ([]ssh.AuthMethod, error) {
	authMethod := strings.ToLower(strings.TrimSpace(input.AuthMethod))
	if authMethod == "" {
		authMethod = "password"
	}
	switch authMethod {
	case "password":
		if input.Password == "" {
			return nil, errBadRequest("未找到可用于测试的密码，请填写密码或确认数据库中已保存密码。")
		}
		return passwordAuthMethods(input.Password), nil
	case "key":
		if strings.TrimSpace(input.PrivateKey) == "" {
			return nil, errBadRequest("未找到可用于测试的私钥，请填写私钥或确认数据库中已保存私钥。")
		}
		var signer ssh.Signer
		var err error
		if input.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(input.PrivateKey), []byte(input.Passphrase))
		}
		if signer == nil {
			signer, err = ssh.ParsePrivateKey([]byte(input.PrivateKey))
		}
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, errBadRequest("无效的认证方式。")
	}
}

// passwordAuthMethods returns both password and keyboard-interactive methods.
// Many cloud SSH servers only accept keyboard-interactive for password logins.
func passwordAuthMethods(password string) []ssh.AuthMethod {
	pw := password
	return []ssh.AuthMethod{
		ssh.Password(pw),
		ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = pw
			}
			return answers, nil
		}),
	}
}

// TouchLastConnected 用于更新连接最近使用时间。
// 输入参数：id 表示标识符。
// 输出参数：返回 error；error 表示执行失败原因。
func (s *Service) TouchLastConnected(id int64) error { return s.repo.TouchLastConnected(id) }

// toConnection 用于校验输入并转换为连接模型。
// 输入参数：input 表示连接输入参数。
// 输出参数：返回 models.Connection, error；error 表示执行失败原因。
func (s *Service) toConnection(input UpsertInput) (models.Connection, error) {
	host := strings.TrimSpace(strings.Trim(input.Host, "[]"))
	if host == "" || input.Username == "" || input.Port <= 0 {
		return models.Connection{}, errBadRequest("缺少必要的连接信息 (host, port, username, auth_method)。")
	}
	connectionType := normalizeConnectionType(input.Type)
	authMethod := strings.ToLower(strings.TrimSpace(input.AuthMethod))
	if connectionType == "RDP" && authMethod == "" {
		authMethod = "password"
	}
	if connectionType == "RDP" && authMethod != "password" {
		return models.Connection{}, errBadRequest("RDP 连接目前仅支持密码认证。")
	}
	if connectionType == "SSH" && authMethod != "password" && authMethod != "key" {
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
		Type:              connectionType,
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

// decryptConnection 用于解密连接中的敏感认证字段。
// 输入参数：connection 表示连接配置。
// 输出参数：返回 *models.DecryptedConnection, error；error 表示执行失败原因。
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

// normalizeConnectionType 用于规范化连接类型字段。
// 输入参数：value 表示输入值。
// 输出参数：返回 string。
func normalizeConnectionType(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "RDP":
		return "RDP"
	case "SSH", "":
		return "SSH"
	default:
		return "SSH"
	}
}
