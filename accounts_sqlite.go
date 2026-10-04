package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const accountsDBName = ".accounts.sqlite"
const accountsReadyName = ".accounts-ready"

func openAccountsDB(root string) (*sql.DB, error) {
	path := filepath.Join(root, accountsDBName)
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
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;CREATE TABLE IF NOT EXISTS auth_users(username TEXT PRIMARY KEY,record BLOB NOT NULL);CREATE TABLE IF NOT EXISTS auth_companies(id TEXT PRIMARY KEY,record BLOB NOT NULL);`); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func (a *App) accountsReady() bool {
	_, err := os.Stat(filepath.Join(a.storageRoot, accountsReadyName))
	return err == nil
}
func (a *App) loadUsersSQLite() ([]storedUser, error) {
	db, err := openAccountsDB(a.storageRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT record FROM auth_users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedUser{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var u storedUser
		if err := json.Unmarshal(b, &u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (a *App) saveUsersSQLite(users []storedUser) error {
	db, err := openAccountsDB(a.storageRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM auth_users`); err != nil {
		return err
	}
	for _, u := range users {
		b, err := json.Marshal(u)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO auth_users(username,record) VALUES(?,?)`, u.Username, b); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (a *App) loadCompaniesSQLite() ([]CompanyProfile, error) {
	db, err := openAccountsDB(a.storageRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT record FROM auth_companies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CompanyProfile{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var c CompanyProfile
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (a *App) saveCompaniesSQLite(companies []CompanyProfile) error {
	db, err := openAccountsDB(a.storageRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM auth_companies`); err != nil {
		return err
	}
	for _, c := range companies {
		b, err := json.Marshal(c)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO auth_companies(id,record) VALUES(?,?)`, c.ID, b); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Switch to SQLite only once every legacy company has encrypted its user profiles.
// Until then, the old account files remain the authoritative recovery source.
func (a *App) maybeMigrateAuthIndex(users []storedUser) error {
	if a.accountsReady() {
		return nil
	}
	for _, u := range users {
		if u.UserProfile != redactedUser(u).UserProfile {
			return nil
		}
	}
	companies, err := a.loadCompaniesUnlocked()
	if err != nil {
		return err
	}
	db, err := openAccountsDB(a.storageRoot)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM auth_users`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM auth_companies`); err != nil {
		return err
	}
	for _, u := range users {
		b, err := json.Marshal(u)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO auth_users(username,record) VALUES(?,?)`, u.Username, b); err != nil {
			return err
		}
	}
	for _, c := range companies {
		b, err := json.Marshal(c)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO auth_companies(id,record) VALUES(?,?)`, c.ID, b); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	var usersCount, companiesCount int
	if err := db.QueryRow(`SELECT count(*) FROM auth_users`).Scan(&usersCount); err != nil {
		return err
	}
	if err := db.QueryRow(`SELECT count(*) FROM auth_companies`).Scan(&companiesCount); err != nil {
		return err
	}
	if usersCount != len(users) || companiesCount != len(companies) {
		return fmt.Errorf("account migration count mismatch")
	}
	if err := writeAtomicPrivate(filepath.Join(a.storageRoot, accountsReadyName), []byte("SQLite accounts ready\n")); err != nil {
		return err
	}
	_ = os.Remove(a.usersPath())
	_ = os.Remove(a.companiesPath())
	return nil
}
