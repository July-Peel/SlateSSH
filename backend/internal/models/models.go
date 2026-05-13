package models

import "time"

type User struct {
    ID             int64     `json:"id"`
    Username       string    `json:"username"`
    HashedPassword string    `json:"-"`
    CreatedAt      time.Time `json:"createdAt"`
    UpdatedAt      time.Time `json:"updatedAt"`
}

type Connection struct {
    ID                int64      `json:"id"`
    Name              string     `json:"name"`
    Type              string     `json:"type"`
    Host              string     `json:"host"`
    Port              int        `json:"port"`
    Username          string     `json:"username"`
    AuthMethod        string     `json:"auth_method"`
    EncryptedPassword string     `json:"-"`
    EncryptedKey      string     `json:"-"`
    EncryptedPhrase   string     `json:"-"`
    Notes             string     `json:"notes"`
    LastConnectedAt   *time.Time `json:"last_connected_at,omitempty"`
    CreatedAt         time.Time  `json:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at"`
}

type DecryptedConnection struct {
    Connection
    Password   string `json:"password,omitempty"`
    PrivateKey string `json:"private_key,omitempty"`
    Passphrase string `json:"passphrase,omitempty"`
}

type Setting struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type ServerStatus struct {
    CPUPercent  float64   `json:"cpuPercent"`
    MemPercent  float64   `json:"memPercent"`
    MemUsedMB   float64   `json:"memUsed"`
    MemTotalMB  float64   `json:"memTotal"`
    SwapPercent float64   `json:"swapPercent"`
    SwapUsedMB  float64   `json:"swapUsed"`
    SwapTotalMB float64   `json:"swapTotal"`
    DiskPercent float64   `json:"diskPercent"`
    DiskUsedKB  float64   `json:"diskUsed"`
    DiskTotalKB float64   `json:"diskTotal"`
    CPUModel    string    `json:"cpuModel"`
    NetRxRate   float64   `json:"netRxRate"`
    NetTxRate   float64   `json:"netTxRate"`
    ServerIP    string    `json:"serverIp"`
    OSName      string    `json:"osName"`
    Timestamp   time.Time `json:"timestamp"`
}
