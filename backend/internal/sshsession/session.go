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

func (s *shellStream) Read(p []byte) (int, error) {
    return s.reader.Read(p)
}

func (s *shellStream) Write(_ []byte) (int, error) {
    return 0, fmt.Errorf("write not supported")
}

func (s *shellStream) Resize(cols, rows int) error {
    return s.session.WindowChange(rows, cols)
}

func (s *shellStream) Close() error {
    return s.session.Close()
}
