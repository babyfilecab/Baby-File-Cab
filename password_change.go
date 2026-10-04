package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// A fresh wrapper is written and synced before the account record switches to it.
// The account store's atomic commit is the sole commit point. Interrupted changes
// leave either the old credentials or the new credentials fully usable.
func (a *App) keyVersionPath(companyID, username, id string) string {
	return a.wrappedKeyPath(companyID, username) + "." + fmt.Sprintf("%x", []byte(id))
}

func (a *App) activeWrappedKeyPath(companyID, username string) (string, error) {
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if u.CompanyID == companyID && strings.EqualFold(u.Username, username) && u.KeyWrapID != "" {
			return a.keyVersionPath(companyID, username, u.KeyWrapID), nil
		}
	}
	return a.wrappedKeyPath(companyID, username), nil
}

func (a *App) ChangePassword(currentPassword, newPassword, confirmation string) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() {
		return errors.New("sign in to BabyFileCab first")
	}
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if newPassword != confirmation {
		return errors.New("passwords do not match")
	}
	if newPassword == currentPassword {
		return errors.New("choose a different new password")
	}
	users, err := a.loadUsersUnlocked()
	if err != nil {
		return err
	}
	for i, u := range users {
		if !strings.EqualFold(u.Username, a.currentUser.Username) || u.CompanyID != a.currentUser.CompanyID {
			continue
		}
		ok, err := verifyStoredPassword(u, currentPassword)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("current password is incorrect")
		}
		key, err := a.unwrapKey(u.CompanyID, u.Username, currentPassword)
		if err != nil {
			return err
		}
		defer zeroBytes(key)
		oldPath, err := a.activeWrappedKeyPath(u.CompanyID, u.Username)
		if err != nil {
			return err
		}
		salt, err := randomBytes(passwordSaltLength)
		if err != nil {
			return err
		}
		id, err := randomBytes(24)
		if err != nil {
			return err
		}
		u.KeyWrapID = base64.RawURLEncoding.EncodeToString(id)
		newPath := a.keyVersionPath(u.CompanyID, u.Username, u.KeyWrapID)
		if err := a.writeWrappedKeyAt(newPath, u.CompanyID, u.Username, newPassword, key); err != nil {
			return err
		}
		// Persist the directory entry before committing the credential reference.
		if err := syncVaultDirectory(filepath.Dir(newPath)); err != nil {
			return err
		}
		hash := argon2.IDKey([]byte(newPassword), salt, 3, 64*1024, 4, passwordKeyLength)
		u.PasswordSalt = base64.StdEncoding.EncodeToString(salt)
		u.PasswordHash = base64.StdEncoding.EncodeToString(hash)
		zeroBytes(hash)
		u.PasswordKDF = "argon2id-v1"
		u.Iterations = 0
		users[i] = u
		// Never remove the new wrapper on a commit error: SQLite commit outcomes can
		// be ambiguous. Both encrypted files stay available for recovery in that case.
		if err := a.saveUsersUnlocked(users); err != nil {
			return err
		}
		// Old wrappers contain encrypted key material only. Best-effort cleanup;
		// failed cleanup must not report an already committed change as unsuccessful.
		_ = os.Remove(oldPath)
		a.lastActivity = time.Now()
		return nil
	}
	return errors.New("account could not be found")
}
