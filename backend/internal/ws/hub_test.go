package ws

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/pkg/sftp"
)

func TestContentDispositionEncodesUnicodeFilename(t *testing.T) {
	got := contentDisposition("报告 (final)'s.txt")
	want := `attachment; filename="__ (final)'s.txt"; filename*=UTF-8''%E6%8A%A5%E5%91%8A%20%28final%29%27s.txt`
	if got != want {
		t.Fatalf("unexpected header:\n got: %s\nwant: %s", got, want)
	}
}

// newMemorySFTPClient 启动一个内存 SFTP 服务端并返回连接到它的客户端。
func newMemorySFTPClient(t *testing.T) *sftp.Client {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	server := sftp.NewRequestServer(serverConn, sftp.InMemHandler())
	go func() { _ = server.Serve() }()
	client, err := sftp.NewClientPipe(clientConn, clientConn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return client
}

func TestCopyRemoteFileCopiesBinaryAndRefusesOverwrite(t *testing.T) {
	client := newMemorySFTPClient(t)
	payload := []byte{0, 1, 2, 0xff, 0xfe, 0}
	source, err := client.Create("/source.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = source.Close()

	written, err := copyRemoteFile(client, "/source.bin", "/target.bin")
	if err != nil {
		t.Fatal(err)
	}
	if written != int64(len(payload)) {
		t.Fatalf("written = %d, want %d", written, len(payload))
	}
	target, err := client.Open("/target.bin")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(target)
	_ = target.Close()
	if !bytes.Equal(got, payload) {
		t.Fatalf("copied content = %v, want %v", got, payload)
	}

	if _, err := copyRemoteFile(client, "/source.bin", "/target.bin"); err == nil {
		t.Fatal("expected copying onto an existing file to fail")
	}
}
