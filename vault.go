package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

const vaultKeySize = 32
const vaultNonceSize = 12
const vaultFileMagic = "BFCENC2\x00"
const keyWrapVersion = 1

type wrappedVaultKey struct {
	Version    int    `json:"version"`
	CompanyID  string `json:"companyId"`
	Username   string `json:"username"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	return b, err
}

func sealWithKey(key, plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	return append(nonce, gcm.Seal(nil, nonce, plaintext, aad)...), nil
}

func openWithKey(key, data, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("encrypted data is damaged")
	}
	return gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], aad)
}

func (a *App) wrappedKeyPath(companyID, username string) string {
	// Account names never become path components, even if legacy data is malformed.
	companyPart := sha256.Sum256([]byte(companyID))
	userPart := sha256.Sum256([]byte(strings.ToLower(username)))
	return filepath.Join(a.storageRoot, ".vault-keys", fmt.Sprintf("%x", companyPart), fmt.Sprintf("%x.json", userPart))
}

func (a *App) writeWrappedKey(companyID, username, password string, vaultKey []byte) error {
	return a.writeWrappedKeyAt(a.wrappedKeyPath(companyID, username), companyID, username, password, vaultKey)
}

func (a *App) writeWrappedKeyAt(path, companyID, username, password string, vaultKey []byte) error {
	if len(vaultKey) != vaultKeySize {
		return errors.New("invalid vault key")
	}
	salt, err := randomBytes(24)
	if err != nil {
		return err
	}
	wrappingKey := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, vaultKeySize)
	aad := []byte("BabyFileCab-vault-key-v1:" + companyID + ":" + strings.ToLower(username))
	encrypted, err := sealWithKey(wrappingKey, vaultKey, aad)
	for i := range wrappingKey {
		wrappingKey[i] = 0
	}
	if err != nil {
		return err
	}
	record := wrappedVaultKey{Version: keyWrapVersion, CompanyID: companyID, Username: strings.ToLower(username), Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(encrypted[:vaultNonceSize]), Ciphertext: base64.StdEncoding.EncodeToString(encrypted[vaultNonceSize:])}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writeAtomicPrivate(path, data)
}

func (a *App) unwrapKey(companyID, username, password string) ([]byte, error) {
	path, err := a.activeWrappedKeyPath(companyID, username)
	if err != nil { return nil, err }
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record wrappedVaultKey
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, errors.New("vault key record is damaged")
	}
	if record.Version != keyWrapVersion || record.CompanyID != companyID || record.Username != strings.ToLower(username) {
		return nil, errors.New("vault key record does not match account")
	}
	salt, err := base64.StdEncoding.DecodeString(record.Salt)
	if err != nil || len(salt) != 24 {
		return nil, errors.New("vault key salt is damaged")
	}
	nonce, err := base64.StdEncoding.DecodeString(record.Nonce)
	if err != nil || len(nonce) != vaultNonceSize {
		return nil, errors.New("vault key nonce is damaged")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(record.Ciphertext)
	if err != nil {
		return nil, errors.New("vault key ciphertext is damaged")
	}
	wrappingKey := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, vaultKeySize)
	aad := []byte("BabyFileCab-vault-key-v1:" + companyID + ":" + strings.ToLower(username))
	key, err := openWithKey(wrappingKey, append(nonce, ciphertext...), aad)
	for i := range wrappingKey {
		wrappingKey[i] = 0
	}
	if err != nil || len(key) != vaultKeySize {
		return nil, errors.New("could not unlock company vault")
	}
	return key, nil
}

func writeAtomicPrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".bfc-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (a *App) vaultKeyCopy() ([]byte, error) {
	a.vaultMu.RLock()
	defer a.vaultMu.RUnlock()
	if len(a.vaultKey) != vaultKeySize {
		return nil, errors.New("company vault is locked")
	}
	return append([]byte(nil), a.vaultKey...), nil
}

func (a *App) vaultEncrypt(data []byte) ([]byte, error) {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	blob, err := sealWithKey(key, data, []byte("BabyFileCab-content-v2"))
	if err != nil {
		return nil, err
	}
	return append([]byte(vaultFileMagic), blob...), nil
}

func (a *App) vaultDecrypt(data []byte) ([]byte, error) {
	if len(data) < len(vaultFileMagic) || subtle.ConstantTimeCompare(data[:len(vaultFileMagic)], []byte(vaultFileMagic)) != 1 {
		return nil, errors.New("file is not encrypted with BabyFileCab")
	}
	key, err := a.vaultKeyCopy()
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	return openWithKey(key, data[len(vaultFileMagic):], []byte("BabyFileCab-content-v2"))
}
