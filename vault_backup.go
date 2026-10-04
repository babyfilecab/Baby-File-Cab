package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const backupManifestName = "BFC-MANIFEST.json"
const maxBackupBytes = 512 << 20

type vaultBackupManifest struct {
	Version      int               `json:"version"`
	CompanyID    string            `json:"companyId"`
	ClientID     string            `json:"clientId"`
	Profile      ClientProfile     `json:"profile"`
	Descriptions map[string]string `json:"descriptions"`
}

func (a *App) encryptedClientBackup(clientPath, dest string) error {
	props, err := a.GetClientProperties(clientPath)
	if err != nil {
		return err
	}
	profile, err := a.readClientProfile(clientPath)
	if err != nil {
		return err
	}
	meta, err := a.readDocumentMetadata()
	if err != nil {
		return err
	}
	prefix := a.documentMetadataKey(clientPath) + "/"
	descriptions := map[string]string{}
	for k, v := range meta {
		if strings.HasPrefix(k, prefix) {
			descriptions[strings.TrimPrefix(k, prefix)] = v
		}
	}
	manifest := vaultBackupManifest{Version: 1, CompanyID: a.currentCompany.ID, ClientID: props.ClientID, Profile: profile, Descriptions: descriptions}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	w, err := zw.Create(backupManifestName)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	err = filepath.WalkDir(clientPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("backup cannot include symbolic links")
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(clientPath, path)
		if err != nil {
			return err
		}
		if rel == backupManifestName {
			return errors.New("reserved backup file name")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := a.secureReadFile(path); err != nil {
			return fmt.Errorf("backup file verification failed: %s: %w", rel, err)
		}
		if buf.Len()+len(content) > maxBackupBytes {
			return errors.New("client backup exceeds 512 MB limit")
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err
	})
	if err != nil {
		zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if buf.Len() > maxBackupBytes {
		return errors.New("client backup exceeds 512 MB limit")
	}
	encrypted, err := a.vaultEncrypt(buf.Bytes())
	if err != nil {
		return err
	}
	return writeAtomicPrivate(dest, encrypted)
}

func (a *App) restoreEncryptedClientBackup(clientPath, backupPath string) error {
	raw, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	plain, err := a.vaultDecrypt(raw)
	if err != nil {
		return fmt.Errorf("cannot unlock this backup with the current company key: %w", err)
	}
	if len(plain) > maxBackupBytes {
		return errors.New("backup exceeds 512 MB limit")
	}
	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return err
	}
	if len(zr.File) > 10000 {
		return errors.New("backup contains too many files")
	}
	var manifest vaultBackupManifest
	seen := map[string]bool{}
	var totalBytes uint64
	stage, err := os.MkdirTemp(filepath.Dir(clientPath), ".restore-stage-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, zf := range zr.File {
		name := filepath.Clean(filepath.FromSlash(zf.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
			return errors.New("unsafe backup path")
		}
		if zf.Mode()&os.ModeSymlink != 0 {
			return errors.New("backup contains symbolic link")
		}
		if zf.UncompressedSize64 > maxBackupBytes {
			return errors.New("backup entry exceeds size limit")
		}
		if seen[name] {
			return errors.New("duplicate backup entry")
		}
		seen[name] = true
		totalBytes += zf.UncompressedSize64
		if totalBytes > maxBackupBytes {
			return errors.New("backup contents exceed 512 MB limit")
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(io.LimitReader(rc, maxBackupBytes+1))
		closeErr := rc.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if len(content) > maxBackupBytes {
			return errors.New("backup entry exceeds size limit")
		}
		if name == backupManifestName {
			if err := json.Unmarshal(content, &manifest); err != nil {
				return err
			}
			continue
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(filepath.Join(stage, name), 0700); err != nil {
				return err
			}
			continue
		}
		if _, err := a.vaultDecrypt(content); err != nil {
			return fmt.Errorf("backup document is damaged: %s", name)
		}
		target := filepath.Join(stage, name)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := writeAtomicPrivate(target, content); err != nil {
			return err
		}
	}
	clientID, _, ok := parseClientDirName(filepath.Base(clientPath))
	if !ok || manifest.Version != 1 || manifest.CompanyID != a.currentCompany.ID || manifest.ClientID != clientID {
		return errors.New("backup belongs to another company or client")
	}
	for k := range manifest.Descriptions {
		if k == "" || filepath.IsAbs(k) || strings.HasPrefix(filepath.Clean(k), "..") {
			return errors.New("invalid document description path")
		}
	}
	previous, err := a.readClientProfile(clientPath)
	if err != nil {
		return err
	}
	previousMeta, err := a.readDocumentMetadata()
	if err != nil {
		return err
	}
	recoveryDir := filepath.Join(filepath.Dir(clientPath), ".restore-snapshots")
	if err := os.MkdirAll(recoveryDir, 0700); err != nil {
		return err
	}
	oldPath := filepath.Join(recoveryDir, filepath.Base(clientPath))
	if _, err := os.Stat(oldPath); err == nil {
		return errors.New("previous restore must be recovered first")
	}
	if err := os.Rename(clientPath, oldPath); err != nil {
		return err
	}
	if err := os.Rename(stage, clientPath); err != nil {
		_ = os.Rename(oldPath, clientPath)
		return err
	}
	if err := a.writeClientProfile(clientPath, manifest.Profile); err != nil {
		_ = os.Rename(clientPath, stage)
		_ = os.Rename(oldPath, clientPath)
		return err
	}
	meta, err := a.readDocumentMetadata()
	if err != nil {
		_ = os.Rename(clientPath, stage)
		_ = os.Rename(oldPath, clientPath)
		_ = a.writeClientProfile(clientPath, previous)
		return err
	}
	prefix := a.documentMetadataKey(clientPath) + "/"
	for k := range meta {
		if strings.HasPrefix(k, prefix) {
			delete(meta, k)
		}
	}
	for k, v := range manifest.Descriptions {
		meta[prefix+filepath.ToSlash(k)] = v
	}
	if err := a.writeDocumentMetadata(meta); err != nil {
		_ = os.Rename(clientPath, stage)
		_ = os.Rename(oldPath, clientPath)
		_ = a.writeClientProfile(clientPath, previous)
		_ = a.writeDocumentMetadata(previousMeta)
		return err
	}
	return os.RemoveAll(oldPath)
}

// Old BabyFileCab ZIP archives are plaintext. Convert entirely in memory to the
// authenticated backup format before the normal validated restore path.
func (a *App) restoreLegacyClientZIP(clientPath, zipPath string) error {
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	if len(raw) > maxBackupBytes {
		return errors.New("legacy ZIP exceeds 512 MB")
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return err
	}
	if len(zr.File) > 10000 {
		return errors.New("legacy ZIP contains too many files")
	}
	clientID, _, ok := parseClientDirName(filepath.Base(clientPath))
	if !ok {
		return errors.New("invalid client")
	}
	manifest := vaultBackupManifest{Version: 1, CompanyID: a.currentCompany.ID, ClientID: clientID, Descriptions: map[string]string{}}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	seen := map[string]bool{}
	var total uint64
	for _, zf := range zr.File {
		name := filepath.Clean(filepath.FromSlash(zf.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == backupManifestName {
			return errors.New("unsafe legacy ZIP entry")
		}
		if seen[name] || zf.Mode()&os.ModeSymlink != 0 {
			return errors.New("duplicate or linked legacy ZIP entry")
		}
		seen[name] = true
		total += zf.UncompressedSize64
		if total > maxBackupBytes {
			return errors.New("legacy ZIP contents exceed 512 MB")
		}
		if zf.FileInfo().IsDir() {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxBackupBytes+1))
		rc.Close()
		if err != nil || len(data) > maxBackupBytes {
			return errors.New("legacy ZIP entry too large or damaged")
		}
		if name == clientProfileFile {
			if err := json.Unmarshal(data, &manifest.Profile); err != nil {
				return err
			}
		}
		sealed, err := a.vaultEncrypt(data)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(name))
		if err != nil {
			return err
		}
		if _, err := w.Write(sealed); err != nil {
			return err
		}
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	w, err := zw.Create(backupManifestName)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	sealed, err := a.vaultEncrypt(buf.Bytes())
	if err != nil {
		return err
	}
	tempDir := filepath.Join(a.storageRoot, ".plaintext-preview")
	f, err := os.CreateTemp(tempDir, "converted-backup-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	f.Close()
	_ = os.Remove(tmp)
	defer os.Remove(tmp)
	if err := writeAtomicPrivate(tmp, sealed); err != nil {
		return err
	}
	return a.restoreEncryptedClientBackup(clientPath, tmp)
}
