package sshsession

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"slatessh/backend/internal/models"
)

// Dial 用于建立 SSH 客户端连接。
// 输入参数：connection 表示连接配置。
// 输出参数：返回 *ssh.Client, error；error 表示执行失败原因。
func Dial(connection *models.DecryptedConnection) (*ssh.Client, error) {
	auth, err := authMethod(connection)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            connection.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(connection.Host, strconv.Itoa(connection.Port)), cfg)
}

// authMethod 用于根据连接配置构造 SSH 认证方式。
// 输入参数：connection 表示连接配置。
// 输出参数：返回 ssh.AuthMethod, error；error 表示执行失败原因。
func authMethod(connection *models.DecryptedConnection) (ssh.AuthMethod, error) {
	switch connection.AuthMethod {
	case "password":
		return ssh.Password(connection.Password), nil
	case "key":
		if connection.Passphrase != "" {
			signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(connection.PrivateKey), []byte(connection.Passphrase))
			if err == nil {
				return ssh.PublicKeys(signer), nil
			}
		}
		signer, err := ssh.ParsePrivateKey([]byte(connection.PrivateKey))
		if err != nil {
			return nil, err
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", connection.AuthMethod)
	}
}

// OpenShell 用于打开带 PTY 的 SSH Shell 会话。
// 输入参数：client 表示SSH 客户端；cols 表示终端列数；rows 表示终端行数。
// 输出参数：返回 io.ReadWriteCloser, io.WriteCloser, error；error 表示执行失败原因。
func OpenShell(client *ssh.Client, cols, rows int) (io.ReadWriteCloser, io.WriteCloser, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = session.Close()
		return nil, nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, nil, err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		return nil, nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, nil, err
	}

	if err := session.Shell(); err != nil {
		_ = session.Close()
		return nil, nil, err
	}

	shell := &shellStream{session: session, reader: io.MultiReader(stdout, stderr)}
	return shell, stdin, nil
}

type shellStream struct {
	session *ssh.Session
	reader  io.Reader
}

// Read 用于从 Shell 输出流读取数据。
// 输入参数：p 表示数据缓冲区。
// 输出参数：返回 int, error；error 表示执行失败原因。
func (s *shellStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

// Write 用于处理对 Shell 包装流的写入请求。
// 输入参数：无。
// 输出参数：返回 int, error；error 表示执行失败原因。
func (s *shellStream) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("write not supported")
}

// Resize 用于同步远端 SSH PTY 窗口尺寸。
// 输入参数：cols 表示终端列数；rows 表示终端行数。
// 输出参数：返回 error；error 表示执行失败原因。
func (s *shellStream) Resize(cols, rows int) error {
	return s.session.WindowChange(rows, cols)
}

// Close 用于关闭底层 Shell 会话。
// 输入参数：无。
// 输出参数：返回 error；error 表示执行失败原因。
func (s *shellStream) Close() error {
	return s.session.Close()
}
