package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEncryptedCompanyEnrollmentAndBackup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	a, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	admin := UserRegistration{CompanyName: "Example Tax", Username: "admin", FirstName: "Ada", LastName: "Accountant", Email: "ada@example.test", Phone: "5551234", Password: "long-admin-password", ConfirmPassword: "long-admin-password", Role: "administrator"}
	profile, err := a.RegisterUser(admin)
	if err != nil {
		t.Fatal(err)
	}
	if !a.accountsReady() {
		t.Fatal("account index not migrated to SQLite")
	}
	if _, err := os.Stat(a.usersPath()); !os.IsNotExist(err) {
		t.Fatal("legacy user JSON still exists")
	}
	client, err := a.AddClient("Alice Client", "123 Main St", "5551234", "alice@example.test", "123-45-6789", "1040")
	if err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(client, permanentFolder, notesFile)
	if err := a.SaveNotes(client, "confidential note"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(note)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("confidential note")) {
		t.Fatal("notes stored in plaintext")
	}
	got, err := a.GetNotes(client)
	if err != nil || got != "confidential note" {
		t.Fatal("notes round trip", err)
	}
	dbRaw, err := os.ReadFile(filepath.Join(a.dataRoot, vaultDBName))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(dbRaw, []byte("123-45-6789")) || bytes.Contains(dbRaw, []byte("alice@example.test")) {
		t.Fatal("client fields stored in plaintext")
	}
	backup := filepath.Join(home, "client.bfcbackup")
	if err := a.encryptedClientBackup(client, backup); err != nil {
		t.Fatal(err)
	}
	backupBytes, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(backupBytes, []byte("confidential note")) {
		t.Fatal("backup stored in plaintext")
	}
	tampered := append([]byte(nil), backupBytes...)
	tampered[len(tampered)-1] ^= 1
	tamperPath := filepath.Join(home, "tampered.bfcbackup")
	os.WriteFile(tamperPath, tampered, 0600)
	if err := a.restoreEncryptedClientBackup(client, tamperPath); err == nil {
		t.Fatal("tampered backup accepted")
	}
	if err := a.SaveNotes(client, "modified"); err != nil {
		t.Fatal(err)
	}
	if err := a.restoreEncryptedClientBackup(client, backup); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetNotes(client)
	if err != nil || got != "confidential note" {
		t.Fatal("restore failed", err)
	}
	legacyZip := filepath.Join(home, "old-backup.zip")
	oldFile, err := os.Create(legacyZip)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(oldFile)
	w, err := zw.Create("Permanent Folder/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("restored legacy note"))
	w, err = zw.Create(clientProfileFile)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte(`{"taxID":"444-33-2222"}`))
	zw.Close()
	oldFile.Close()
	if err := a.restoreLegacyClientZIP(client, legacyZip); err != nil {
		t.Fatal("legacy restore", err)
	}
	got, err = a.GetNotes(client)
	if err != nil || got != "restored legacy note" {
		t.Fatal("legacy ZIP restore", err)
	}
	raw, _ = os.ReadFile(note)
	if bytes.Contains(raw, []byte("restored legacy note")) {
		t.Fatal("legacy restore left plaintext in vault")
	}
	staffReq := ExistingCompanyUserRegistration{CompanyID: profile.CompanyID, AdminUsername: "admin", AdminPassword: "long-admin-password", Username: "staff", FirstName: "Sam", LastName: "Staff", Email: "sam@example.test", Phone: "5554321", Password: "long-staff-password", ConfirmPassword: "long-staff-password", Role: "staff"}
	if _, err := a.CreateUserUnderExistingCompany(staffReq); err != nil {
		t.Fatal(err)
	}
	a.Logout()
	if _, err := a.Login("staff", "long-staff-password"); err != nil {
		t.Fatal("staff vault unlock", err)
	}
	if got, err := a.GetNotes(client); err != nil || got != "restored legacy note" {
		t.Fatal("staff document access", err)
	}
	if _, err := a.Login("staff", "wrong-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	a.authMu.Lock()
	a.lastActivity = time.Now().Add(-16 * time.Minute)
	a.authMu.Unlock()
	if a.TouchActivity() {
		t.Fatal("expired vault session revived")
	}
	if _, err := a.GetNotes(client); err == nil {
		t.Fatal("expired session read document")
	}
	a.Logout()
	reopened, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Login("staff", "long-staff-password"); err != nil {
		t.Fatal("SQLite account reopen", err)
	}
	if got, err := reopened.GetNotes(client); err != nil || got != "restored legacy note" {
		t.Fatal("reopened vault document", err)
	}
	reopened.Logout()
}

func TestInterruptedMigrationResumesWithoutPlaintext(t *testing.T) {
	root := t.TempDir()
	company := filepath.Join(root, "companies", "company-a")
	client := filepath.Join(company, "00001 - Client", "Permanent Folder")
	if err := os.MkdirAll(client, 0700); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(client, "notes.txt")
	if err := os.WriteFile(note, []byte("private note"), 0600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(company, "zzz-link")
	if err := os.Symlink(note, symlink); err != nil {
		t.Skip("symlinks unavailable")
	}
	key, _ := randomBytes(32)
	if err := migrateCompanyData(company, root, "company-a", key); err == nil {
		t.Fatal("symlink migration was allowed")
	}
	if _, err := os.Stat(filepath.Join(company, vaultReadyName)); !os.IsNotExist(err) {
		t.Fatal("incomplete migration marked ready")
	}
	os.Remove(symlink)
	if err := migrateCompanyData(company, root, "company-a", key); err != nil {
		t.Fatal("resume failed", err)
	}
	raw, _ := os.ReadFile(note)
	if bytes.Contains(raw, []byte("private note")) {
		t.Fatal("plaintext remained")
	}
}

func TestLegacyCompanyRequiresUserEnrollment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root := filepath.Join(home, "BabyFileCabData")
	client := filepath.Join(root, "00001 - Existing Client")
	if err := os.MkdirAll(filepath.Join(client, permanentFolder), 0700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(client, clientProfileFile), []byte(`{"taxID":"987-65-4321"}`), 0600)
	os.WriteFile(filepath.Join(client, permanentFolder, notesFile), []byte("legacy secret"), 0600)
	legacyUser := func(username, password, role string) storedUser {
		salt := bytes.Repeat([]byte{7}, 24)
		hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, passwordKeyLength)
		return storedUser{UserProfile: UserProfile{Username: username, FirstName: username, LastName: "Person", Email: username + "@example.test", Phone: "5550000", Role: role}, PasswordSalt: base64.StdEncoding.EncodeToString(salt), PasswordHash: base64.StdEncoding.EncodeToString(hash), Iterations: passwordIterations}
	}
	users := []storedUser{legacyUser("admin", "legacy-admin-pass", "administrator"), legacyUser("staff", "legacy-staff-pass", "staff")}
	b, _ := json.Marshal(users)
	os.WriteFile(filepath.Join(root, ".users.json"), b, 0600)
	a, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	// Existing accounts without key wrappers must fail closed, including legacy
	// administrators. Login must not bootstrap a replacement vault key.
	for _, credentials := range [][2]string{{"admin", "legacy-admin-pass"}, {"staff", "legacy-staff-pass"}} {
		if _, err := a.Login(credentials[0], credentials[1]); err == nil || !strings.Contains(err.Error(), "no replacement key") {
			t.Fatal("legacy login did not fail closed", credentials[0], err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(client, permanentFolder, notesFile))
	if err != nil || string(raw) != "legacy secret" {
		t.Fatal("failed login changed legacy notes", err)
	}
}
