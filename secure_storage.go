package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const vaultDBName = ".vault.sqlite"
const vaultReadyName = ".vault-ready"

func openVaultDB(root string) (*sql.DB, error) {
	path := filepath.Join(root, vaultDBName)
	if f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600); err == nil {
		f.Close()
	} else if !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS secure_metadata (
        kind TEXT NOT NULL, item_key TEXT NOT NULL, ciphertext BLOB NOT NULL,
        PRIMARY KEY (kind, item_key)
    );`); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func metadataAAD(kind, itemKey string) []byte {
	return []byte("BabyFileCab-meta-v1:" + kind + ":" + itemKey)
}
func metadataIndex(key []byte, kind, itemKey string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(metadataAAD(kind, itemKey))
	return hex.EncodeToString(mac.Sum(nil))
}

func metadataPut(db *sql.DB, key []byte, kind, itemKey string, value []byte) error {
	blob, err := sealWithKey(key, value, metadataAAD(kind, itemKey))
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO secure_metadata(kind,item_key,ciphertext) VALUES(?,?,?)
		ON CONFLICT(kind,item_key) DO UPDATE SET ciphertext=excluded.ciphertext`, kind, metadataIndex(key, kind, itemKey), blob)
	return err
}

func metadataGet(db *sql.DB, key []byte, kind, itemKey string) ([]byte, error) {
	var blob []byte
	if err := db.QueryRow(`SELECT ciphertext FROM secure_metadata WHERE kind=? AND item_key=?`, kind, metadataIndex(key, kind, itemKey)).Scan(&blob); err != nil {
		return nil, err
	}
	return openWithKey(key, blob, metadataAAD(kind, itemKey))
}

func (a *App) encryptedMetadataGet(kind, itemKey string) ([]byte, error) {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)
	db, err := openVaultDB(a.dataRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return metadataGet(db, key, kind, itemKey)
}
func (a *App) encryptedMetadataPut(kind, itemKey string, value []byte) error {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return err
	}
	defer zeroBytes(key)
	db, err := openVaultDB(a.dataRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	return metadataPut(db, key, kind, itemKey, value)
}
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func encryptedFileBytes(key, plaintext []byte) ([]byte, error) {
	blob, err := sealWithKey(key, plaintext, []byte("BabyFileCab-content-v2"))
	if err != nil {
		return nil, err
	}
	return append([]byte(vaultFileMagic), blob...), nil
}
func decryptedFileBytes(key, content []byte) ([]byte, error) {
	if !bytes.HasPrefix(content, []byte(vaultFileMagic)) {
		return nil, errors.New("unencrypted file in company vault")
	}
	return openWithKey(key, content[len(vaultFileMagic):], []byte("BabyFileCab-content-v2"))
}
func (a *App) secureReadFile(path string) ([]byte, error) {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decryptedFileBytes(key, content)
}
func (a *App) secureWriteFile(path string, data []byte) error {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return err
	}
	defer zeroBytes(key)
	content, err := encryptedFileBytes(key, data)
	if err != nil {
		return err
	}
	return writeAtomicPrivate(path, content)
}
func (a *App) secureCopyFile(src, dst string) error {
	var plain []byte
	var err error
	if a.isInsideVault(src) {
		if _, err = a.safePath(src); err != nil {
			return err
		}
		plain, err = a.secureReadFile(src)
	} else {
		plain, err = os.ReadFile(src)
	}
	if err != nil {
		return err
	}
	return a.secureWriteFile(dst, plain)
}

// Migration is resumable: each original is replaced atomically only after an
// encrypted backup and round-trip check. The ready marker is written last.
func migrateCompanyData(root, storageRoot, companyID string, key []byte) error {
	if len(key) != vaultKeySize {
		return errors.New("invalid company key")
	}
	if err:=os.Chmod(root,0700);err!=nil{return err}
	if _, err := os.Stat(filepath.Join(root, vaultReadyName)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	companyPart := sha256.Sum256([]byte(companyID))
	backupRoot := filepath.Join(storageRoot, ".migration-backups", fmt.Sprintf("%x", companyPart))
	if err := os.MkdirAll(backupRoot, 0700); err != nil {
		return err
	}
	db, err := openVaultDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	// Move JSON metadata into encrypted SQLite entries. Legacy files are only
	// removed once the corresponding encrypted row can be read back.
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in company data: %s", path)
		}
		if d.IsDir() {
			if err:=os.Chmod(path,0700);err!=nil{return err}
			if path == root {
				return nil
			}
			if root == storageRoot && filepath.Dir(path) == root && (d.Name() == "companies" || d.Name() == ".vault-keys" || d.Name() == ".migration-backups" || d.Name() == ".plaintext-preview") {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if path == filepath.Join(root, vaultDBName) || name == vaultDBName+"-journal" || name == vaultDBName+"-wal" || name == vaultDBName+"-shm" || name == vaultReadyName || (root == storageRoot && (name == accountsDBName || name == accountsReadyName || strings.HasPrefix(name, accountsDBName+"-"))) {
			return nil
		}
		if root == storageRoot && (name == ".users.json" || name == ".companies.json") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		kind := ""
		itemKey := ""
		switch name {
		case clientProfileFile:
			kind = "client-profile"
			itemKey = filepath.ToSlash(filepath.Dir(rel))
		case documentMetadataFile:
			kind = "document-metadata"
			itemKey = "all"
		case firmCalendarFile:
			kind = "firm-calendar"
			itemKey = "all"
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if kind != "" {
			if _, err := metadataGet(db, key, kind, itemKey); errors.Is(err, sql.ErrNoRows) {
				if !json.Valid(raw) {
					return fmt.Errorf("invalid legacy JSON: %s", path)
				}
				if err := metadataPut(db, key, kind, itemKey, raw); err != nil {
					return err
				}
				check, err := metadataGet(db, key, kind, itemKey)
				if err != nil || !bytes.Equal(check, raw) {
					return fmt.Errorf("metadata migration verification failed: %s", path)
				}
			} else if err != nil {
				return err
			}
		}
		if bytes.HasPrefix(raw, []byte(vaultFileMagic)) {
			_, err := decryptedFileBytes(key, raw)
			return err
		}
		backupPath := filepath.Join(backupRoot, rel) + ".bfc"
		if err := os.MkdirAll(filepath.Dir(backupPath), 0700); err != nil {
			return err
		}
		backup, err := encryptedFileBytes(key, raw)
		if err != nil {
			return err
		}
		if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
			if err := writeAtomicPrivate(backupPath, backup); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		check, err := os.ReadFile(backupPath)
		if err != nil {
			return err
		}
		plain, err := decryptedFileBytes(key, check)
		if err != nil || !bytes.Equal(plain, raw) {
			return fmt.Errorf("migration backup verification failed: %s", path)
		}
		if err := writeAtomicPrivate(path, backup); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return writeAtomicPrivate(filepath.Join(root, vaultReadyName), []byte("BabyFileCab vault migration complete\n"))
}

func (a *App) metadataJSONGet(kind, itemKey string, out interface{}) error {
	data, err := a.encryptedMetadataGet(kind, itemKey)
	if errors.Is(err, sql.ErrNoRows) {
		return os.ErrNotExist
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
func (a *App) metadataJSONPut(kind, itemKey string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return a.encryptedMetadataPut(kind, itemKey, data)
}
func (a *App) metadataDelete(kind, itemKey string) error {
	key, err := a.vaultKeyCopy()
	if err != nil {
		return err
	}
	defer zeroBytes(key)
	db, err := openVaultDB(a.dataRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`DELETE FROM secure_metadata WHERE kind=? AND item_key=?`, kind, metadataIndex(key, kind, itemKey))
	return err
}
func (a *App) metadataRenamePrefix(kind, oldPrefix, newPrefix string) error {
	data, err := a.encryptedMetadataGet(kind, oldPrefix)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := a.encryptedMetadataPut(kind, newPrefix, data); err != nil {
		return err
	}
	return a.metadataDelete(kind, oldPrefix)
}

// Legacy preview programs require a pathname. The temporary file is private
// and removed as soon as the conversion finishes. OpenFile keeps it until lock.
func (a *App) decryptedTemp(path string) (string, func(), error) {
	data, err := a.secureReadFile(path)
	if err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp(filepath.Join(a.storageRoot, ".plaintext-preview"), "preview-*")
	if err != nil {
		return "", nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	name := filepath.Join(dir, filepath.Base(path))
	if err := os.WriteFile(name, data, 0600); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	a.tempMu.Lock()
	if a.tempDirs == nil {
		a.tempDirs = make(map[string]struct{})
	}
	a.tempDirs[dir] = struct{}{}
	a.tempMu.Unlock()
	cleanup := func() { a.tempMu.Lock(); delete(a.tempDirs, dir); a.tempMu.Unlock(); _ = os.RemoveAll(dir) }
	return name, cleanup, nil
}
func (a *App) removeDecryptedTemps() {
	a.tempMu.Lock()
	defer a.tempMu.Unlock()
	for dir := range a.tempDirs {
		_ = os.RemoveAll(dir)
		delete(a.tempDirs, dir)
	}
}

func (a *App) isInsideVault(path string) bool {
	root, _ := filepath.Abs(a.dataRoot)
	abs, _ := filepath.Abs(path)
	return abs == root || strings.HasPrefix(abs, root+string(os.PathSeparator))
}
func (a *App) generateOutput(path string, write func(string) error) error {
	if !a.isInsideVault(path) {
		return write(path)
	}
	dir, err := os.MkdirTemp(filepath.Join(a.storageRoot, ".plaintext-preview"), "generate-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	temp := filepath.Join(dir, filepath.Base(path))
	if err := write(temp); err != nil {
		return err
	}
	data, err := os.ReadFile(temp)
	if err != nil {
		return err
	}
	return a.secureWriteFile(path, data)
}
func (a *App) openGeneratedOutput(path string) error {
	if a.isInsideVault(path) {
		return a.OpenFile(path)
	}
	return openPath(path)
}

func (a *App) appendEncryptedLog(path string, line []byte) error {
	existing, err := a.secureReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		existing = nil
	} else if err != nil {
		return err
	}
	existing = append(existing, line...)
	return a.secureWriteFile(path, existing)
}

func redactedUser(user storedUser) storedUser {
	user.UserProfile = UserProfile{Username: user.Username, Role: user.Role, CompanyID: user.CompanyID, CreatedAt: user.CreatedAt}
	return user
}
func (a *App) migrateCompanyUserProfiles(users []storedUser, company CompanyProfile, root string, key []byte) ([]storedUser, error) {
	db, err := openVaultDB(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	changed := false
	for i := range users {
		if users[i].CompanyID != company.ID {
			continue
		}
		itemKey := strings.ToLower(users[i].Username)
		_, getErr := metadataGet(db, key, "user-profile", itemKey)
		if errors.Is(getErr, sql.ErrNoRows) {
			if users[i].FirstName == "" && users[i].LastName == "" && users[i].Email == "" {
				return nil, fmt.Errorf("user profile missing for @%s", users[i].Username)
			}
			data, err := json.Marshal(users[i].UserProfile)
			if err != nil {
				return nil, err
			}
			if err := metadataPut(db, key, "user-profile", itemKey, data); err != nil {
				return nil, err
			}
			got, err := metadataGet(db, key, "user-profile", itemKey)
			if err != nil || !bytes.Equal(got, data) {
				return nil, errors.New("user profile migration verification failed")
			}
		} else if getErr != nil {
			return nil, getErr
		}
		if users[i].UserProfile != redactedUser(users[i]).UserProfile {
			users[i] = redactedUser(users[i])
			changed = true
		}
	}
	if changed {
		if err := a.saveUsersUnlocked(users); err != nil {
			return nil, err
		}
	}
	return users, nil
}
