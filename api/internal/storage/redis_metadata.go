package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"visual-kyc/api/internal/domain"
)

type RedisMetadataStore struct {
	addr     string
	password string
	timeout  time.Duration
}

func NewRedisMetadataStore(addr, password string) *RedisMetadataStore {
	return &RedisMetadataStore{addr: addr, password: password, timeout: 3 * time.Second}
}

func (s *RedisMetadataStore) SaveIdentity(meta domain.IdentityMetadata) error {
	now := time.Now().UTC()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.UpdatedAt = now
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return s.set("identity:"+meta.IdentityID, string(b))
}

func (s *RedisMetadataStore) GetIdentity(identityID string) (*domain.IdentityMetadata, error) {
	val, err := s.get("identity:" + identityID)
	if err != nil {
		return nil, err
	}
	var meta domain.IdentityMetadata
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *RedisMetadataStore) SaveStatus(record domain.StatusRecord) error {
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.set("status:"+record.TransactionID, string(b))
}

func (s *RedisMetadataStore) GetStatus(transactionID string) (*domain.StatusRecord, error) {
	val, err := s.get("status:" + transactionID)
	if err != nil {
		return nil, err
	}
	var rec domain.StatusRecord
	if err := json.Unmarshal([]byte(val), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *RedisMetadataStore) set(key, value string) error {
	_, err := s.command("SET", key, value)
	return err
}

func (s *RedisMetadataStore) get(key string) (string, error) {
	resp, err := s.command("GET", key)
	if err != nil {
		return "", err
	}
	if resp.kind == '$' && resp.nilBulk {
		return "", errNotExist()
	}
	return resp.value, nil
}

type redisResp struct {
	kind    byte
	value   string
	nilBulk bool
}

func (s *RedisMetadataStore) command(args ...string) (redisResp, error) {
	conn, err := net.DialTimeout("tcp", s.addr, s.timeout)
	if err != nil {
		return redisResp{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
	reader := bufio.NewReader(conn)
	if s.password != "" {
		if err := writeRESP(conn, []string{"AUTH", s.password}); err != nil {
			return redisResp{}, err
		}
		if _, err := readRESP(reader); err != nil {
			return redisResp{}, err
		}
	}
	if err := writeRESP(conn, args); err != nil {
		return redisResp{}, err
	}
	return readRESP(reader)
}

func writeRESP(conn net.Conn, args []string) error {
	var buf bytes.Buffer
	buf.WriteString("*")
	buf.WriteString(strconv.Itoa(len(args)))
	buf.WriteString("\r\n")
	for _, a := range args {
		buf.WriteString("$")
		buf.WriteString(strconv.Itoa(len(a)))
		buf.WriteString("\r\n")
		buf.WriteString(a)
		buf.WriteString("\r\n")
	}
	_, err := conn.Write(buf.Bytes())
	return err
}

func readRESP(r *bufio.Reader) (redisResp, error) {
	kind, err := r.ReadByte()
	if err != nil {
		return redisResp{}, err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return redisResp{}, err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch kind {
	case '+':
		return redisResp{kind: kind, value: line}, nil
	case '-':
		return redisResp{}, fmt.Errorf("redis error: %s", line)
	case ':':
		return redisResp{kind: kind, value: line}, nil
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return redisResp{}, err
		}
		if n == -1 {
			return redisResp{kind: kind, nilBulk: true}, nil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return redisResp{}, err
		}
		return redisResp{kind: kind, value: string(buf[:n])}, nil
	default:
		return redisResp{}, fmt.Errorf("unsupported redis response kind %q", kind)
	}
}

func errNotExist() error { return os.ErrNotExist }
