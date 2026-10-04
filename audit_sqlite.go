package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func auditIndex(key []byte, clientID string) string {
	return metadataIndex(key, "audit-client", clientID)
}
func auditAAD(scope, clientIndex string) []byte {
	return []byte("BabyFileCab-audit-v1:" + scope + ":" + clientIndex)
}
func initAuditTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS secure_audit (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 scope TEXT NOT NULL, client_index TEXT NOT NULL,
 source_key TEXT NOT NULL UNIQUE, ciphertext BLOB NOT NULL
 );CREATE INDEX IF NOT EXISTS audit_scope_client ON secure_audit(scope,client_index,id);`)
	return err
}
func insertAudit(db *sql.DB, key []byte, scope, clientID, sourceKey string, entry interface{}) error {
	if err := initAuditTable(db); err != nil {
		return err
	}
	idx := auditIndex(key, clientID)
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	sealed, err := sealWithKey(key, b, auditAAD(scope, idx))
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT OR IGNORE INTO secure_audit(scope,client_index,source_key,ciphertext) VALUES(?,?,?,?)`, scope, idx, sourceKey, sealed)
	return err
}
func (a *App) appendAuditSQLite(scope, clientID string, entry interface{}) error {
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
	id, err := randomBytes(16)
	if err != nil {
		return err
	}
	return insertAudit(db, key, scope, clientID, hex.EncodeToString(id), entry)
}
func (a *App) loadAuditSQLite(scope, clientID string, decode func([]byte) error) error {
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
	if err := initAuditTable(db); err != nil {
		return err
	}
	idx := auditIndex(key, clientID)
	rows, err := db.Query(`SELECT ciphertext FROM secure_audit WHERE scope=? AND client_index=? ORDER BY id DESC`, scope, idx)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var blob []byte
		if err := rows.Scan(&blob); err != nil {
			return err
		}
		b, err := openWithKey(key, blob, auditAAD(scope, idx))
		if err != nil {
			return err
		}
		if err := decode(b); err != nil {
			return err
		}
	}
	return rows.Err()
}

func migrationSourceKey(path string, line int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("migration:%s:%d", path, line)))
	return hex.EncodeToString(h[:])
}
func migrateAuditToSQLite(root, storageRoot string, key []byte) error {
	db, err := openVaultDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := initAuditTable(db); err != nil {
		return err
	}
	if _, err := metadataGet(db, key, "migration-state", "audit-v1"); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	appPath := filepath.Join(root, appAuditFile)
	_, appErr := os.Stat(appPath)
	hasAppLog := appErr == nil
	if appErr != nil && !errors.Is(appErr, os.ErrNotExist) {
		return appErr
	}
	var migratedPaths []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if root == storageRoot && filepath.Dir(path) == root && (d.Name() == "companies" || d.Name() == ".migration-backups" || d.Name() == ".vault-keys" || d.Name() == ".plaintext-preview") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != auditFile && path != appPath {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		clear, err := decryptedFileBytes(key, raw)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		clientID := ""
		clientName := ""
		if path != appPath {
			for _, part := range splitRel(rel) {
				if id, name, ok := parseClientDirName(part); ok {
					clientID = id
					clientName = name
					break
				}
			}
			if clientID == "" {
				return nil
			}
		}
		scanner := bufio.NewScanner(bytes.NewReader(clear))
		scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
		line := 0
		for scanner.Scan() {
			line++
			source := migrationSourceKey(rel, line)
			if path == appPath {
				var entry AppAuditEntry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					continue
				}
				if err := insertAudit(db, key, "app", "", source, entry); err != nil {
					return err
				}
			}
			if path != appPath {
				var entry AuditEntry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					continue
				}
				if err := insertAudit(db, key, "client", clientID, source, entry); err != nil {
					return err
				}
				if !hasAppLog {
					global := AppAuditEntry{Timestamp: entry.Timestamp, ClientID: clientID, ClientName: clientName, Action: entry.Action, Details: entry.Details}
					if err := insertAudit(db, key, "app", "", source+"-app", global); err != nil {
						return err
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		migratedPaths = append(migratedPaths, path)
		return nil
	})
	if err != nil {
		return err
	}
	if err := metadataPut(db, key, "migration-state", "audit-v1", []byte("done")); err != nil {
		return err
	}
	// Encrypted source files are redundant after successful import. The encrypted
	// migration recovery copies remain available for repair.
	for _, path := range migratedPaths {
		_ = os.Remove(path)
	}
	return nil
}
func sortAppAudit(entries []AppAuditEntry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Timestamp > entries[j].Timestamp })
}
func sortClientAudit(entries []AuditEntry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Timestamp > entries[j].Timestamp })
}
func isAuditPath(path string) bool {
	return strings.HasSuffix(path, auditFile) || strings.HasSuffix(path, appAuditFile)
}
