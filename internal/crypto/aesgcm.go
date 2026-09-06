// Package crypto содержит сервисы шифрования приватных данных.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
)

const (
	streamMagic     = "GPK1"
	streamChunkSize = 64 * 1024
)

// ErrInvalidCiphertext означает, что ciphertext не удалось расшифровать.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// AESGCMEncryptor шифрует данные через AES-GCM.
type AESGCMEncryptor struct {
	aead cipher.AEAD
}

// NewAESGCMEncryptor создает AESGCMEncryptor из master key.
func NewAESGCMEncryptor(masterKey string) (*AESGCMEncryptor, error) {
	key := sha256.Sum256([]byte(masterKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &AESGCMEncryptor{aead: aead}, nil
}

// Encrypt шифрует plaintext и возвращает ciphertext с nonce.
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	return e.aead.Seal(nil, nonce, plaintext, nil), nonce, nil
}

// Decrypt расшифровывает ciphertext по nonce.
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte, nonce []byte) ([]byte, error) {
	if len(nonce) != e.aead.NonceSize() {
		return nil, ErrInvalidCiphertext
	}
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
	}
	return plaintext, nil
}

// EncryptStream шифрует stream чанками и возвращает размер и sha256 исходных данных.
func (e *AESGCMEncryptor) EncryptStream(src io.Reader, dst io.Writer) (int64, string, error) {
	checksum := sha256.New()
	if _, err := dst.Write([]byte(streamMagic)); err != nil {
		return 0, "", fmt.Errorf("write stream magic: %w", err)
	}

	buffer := make([]byte, streamChunkSize)
	var size int64
	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			size += int64(n)
			if _, err := checksum.Write(chunk); err != nil {
				return 0, "", fmt.Errorf("write checksum: %w", err)
			}
			if err := e.encryptChunk(dst, chunk); err != nil {
				return 0, "", err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return 0, "", fmt.Errorf("read plaintext stream: %w", readErr)
		}
	}
	return size, hexChecksum(checksum), nil
}

// DecryptStream расшифровывает stream, записанный методом EncryptStream.
func (e *AESGCMEncryptor) DecryptStream(src io.Reader, dst io.Writer) error {
	magic := make([]byte, len(streamMagic))
	if _, err := io.ReadFull(src, magic); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
	}
	if string(magic) != streamMagic {
		return ErrInvalidCiphertext
	}

	for {
		var nonceLen uint32
		if err := binary.Read(src, binary.BigEndian, &nonceLen); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
		}
		nonce := make([]byte, nonceLen)
		if _, err := io.ReadFull(src, nonce); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
		}

		var ciphertextLen uint32
		if err := binary.Read(src, binary.BigEndian, &ciphertextLen); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
		}
		ciphertext := make([]byte, ciphertextLen)
		if _, err := io.ReadFull(src, ciphertext); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
		}
		plaintext, err := e.Decrypt(ciphertext, nonce)
		if err != nil {
			return err
		}
		if _, err = dst.Write(plaintext); err != nil {
			return fmt.Errorf("write plaintext stream: %w", err)
		}
	}
}

func (e *AESGCMEncryptor) encryptChunk(dst io.Writer, plaintext []byte) error {
	ciphertext, nonce, err := e.Encrypt(plaintext)
	if err != nil {
		return err
	}
	if err = binary.Write(dst, binary.BigEndian, uint32(len(nonce))); err != nil {
		return fmt.Errorf("write nonce size: %w", err)
	}
	if _, err = dst.Write(nonce); err != nil {
		return fmt.Errorf("write nonce: %w", err)
	}
	if err = binary.Write(dst, binary.BigEndian, uint32(len(ciphertext))); err != nil {
		return fmt.Errorf("write ciphertext size: %w", err)
	}
	if _, err = dst.Write(ciphertext); err != nil {
		return fmt.Errorf("write ciphertext: %w", err)
	}
	return nil
}

func hexChecksum(checksum hash.Hash) string {
	return fmt.Sprintf("%x", checksum.Sum(nil))
}
