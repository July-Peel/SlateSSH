package sshsession

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"slatessh/backend/internal/models"
)

// Dial 用于建立 SSH 客户端连接。
// 输入参数：connection 表示连接配置。
// 输出参数：返回 *ssh.Client, error；error 表示执行失败原因。
func Dial(connection *models.DecryptedConnection) (*ssh.Client, error) {
	auths, err := authMethods(connection)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            connection.Username,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
	host := strings.TrimSpace(strings.Trim(connection.Host, "[]"))
	return ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(connection.Port)), cfg)
}

// authMethods builds SSH auth methods from a decrypted connection.
// Password auth includes keyboard-interactive for servers that reject pure "password".
func authMethods(connection *models.DecryptedConnection) ([]ssh.AuthMethod, error) {
	method := strings.ToLower(strings.TrimSpace(connection.AuthMethod))
	if method == "" {
		method = "password"
	}
	switch method {
	case "password":
		if connection.Password == "" {
			return nil, fmt.Errorf("password is empty")
		}
		pw := connection.Password
		return []ssh.AuthMethod{
			ssh.Password(pw),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = pw
				}
				return answers, nil
			}),
		}, nil
	case "key":
		var signer ssh.Signer
		var err error
		if connection.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(connection.PrivateKey), []byte(connection.Passphrase))
		}
		if signer == nil {
			signer, err = ssh.ParsePrivateKey([]byte(connection.PrivateKey))
		}
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", connection.AuthMethod)
	}
}

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
