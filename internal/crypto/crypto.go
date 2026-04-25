// Package crypto 提供加密工具
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// AESEncryptor AES 加密器
type AESEncryptor struct {
	key []byte
	mu  sync.Mutex
}

// NewAESEncryptor 创建 AES 加密器
func NewAESEncryptor() *AESEncryptor {
	return &AESEncryptor{
		key: deriveKey(),
	}
}

// deriveKey 从机器信息派生加密密钥
func deriveKey() []byte {
	// 获取机器标识信息
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	// 组合生成唯一标识
	unique := hostname + homeDir + "magic-frpc-encryption-key"

	// 使用 SHA256 派生 32 字节密钥
	hash := sha256.Sum256([]byte(unique))
	return hash[:]
}

// Encrypt 使用 AES-GCM 加密
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 AES-GCM 解密
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文数据太短")
	}

	// 提取 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEncrypted 检查字符串是否已加密（简单判断）
func IsEncrypted(s string) bool {
	// 加密后的字符串是 Base64 编码，长度通常较长且包含特定模式
	if len(s) < 20 {
		return false
	}
	// 尝试 Base64 解码
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
