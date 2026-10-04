package main

import (
 "bytes"
 "os"
 "path/filepath"
 "testing"
 "time"
)

func TestPasswordChangePreservesVaultAndOtherUsers(t *testing.T) {
 for _, sqlite := range []bool{false, true} {
  name := "legacy"
  if sqlite { name = "sqlite" }
  t.Run(name, func(t *testing.T) {
   root := t.TempDir()
   a := &App{storageRoot: root, autoLockMinutes: 15}
   if sqlite {
    if err := os.WriteFile(filepath.Join(root, accountsReadyName), []byte("ready"), 0600); err != nil { t.Fatal(err) }
   }
   company := CompanyProfile{ID: "company"}
   user, profile, err := buildStoredUser(UserRegistration{Username: "alice", Password: "old-password"}, company)
   if err != nil { t.Fatal(err) }
   other, _, err := buildStoredUser(UserRegistration{Username: "bob", Password: "bob-password"}, company)
   if err != nil { t.Fatal(err) }
   if err := a.saveUsersUnlocked([]storedUser{user, other}); err != nil { t.Fatal(err) }
   key := bytes.Repeat([]byte{42}, vaultKeySize)
   for _, credentials := range [][2]string{{"alice", "old-password"}, {"bob", "bob-password"}} {
    if err := a.writeWrappedKey(company.ID, credentials[0], credentials[1], key); err != nil { t.Fatal(err) }
   }
   ciphertext, err := sealWithKey(key, []byte("customer document"), []byte("test"))
   if err != nil { t.Fatal(err) }
   a.currentUser = &profile
   a.lastActivity = time.Now()
   for _, input := range [][3]string{{"wrong", "new-password", "new-password"}, {"old-password", "short", "short"}, {"old-password", "new-password", "mismatch"}} {
    if a.ChangePassword(input[0], input[1], input[2]) == nil { t.Fatal("invalid change accepted") }
   }
   if err := a.ChangePassword("old-password", "new-password", "new-password"); err != nil { t.Fatal(err) }
   restarted := &App{storageRoot: root}
   users, err := restarted.loadUsersUnlocked()
   if err != nil { t.Fatal(err) }
   for _, u := range users {
    if u.Username == "alice" {
     ok, err := verifyStoredPassword(u, "new-password")
     if err != nil || !ok { t.Fatal("new password rejected") }
     ok, _ = verifyStoredPassword(u, "old-password")
     if ok { t.Fatal("old password accepted") }
    } else if u != other { t.Fatal("other account changed") }
   }
   restoredKey, err := restarted.unwrapKey(company.ID, "alice", "new-password")
   if err != nil || !bytes.Equal(restoredKey, key) { t.Fatal("vault key changed or inaccessible", err) }
   plaintext, err := openWithKey(restoredKey, ciphertext, []byte("test"))
   if err != nil || string(plaintext) != "customer document" { t.Fatal("document inaccessible") }
   if _, err := restarted.unwrapKey(company.ID, "alice", "old-password"); err == nil { t.Fatal("old password unwraps key") }
   if _, err := restarted.unwrapKey(company.ID, "bob", "bob-password"); err != nil { t.Fatal("other user lost access", err) }
   // An uncommitted wrapper must not change authoritative credentials.
   if err := restarted.writeWrappedKeyAt(restarted.keyVersionPath(company.ID, "alice", "uncommitted"), company.ID, "alice", "unused-password", key); err != nil { t.Fatal(err) }
   if _, err := restarted.unwrapKey(company.ID, "alice", "new-password"); err != nil { t.Fatal("staged wrapper changed active key", err) }
   a.lastActivity = time.Now().Add(-16*time.Minute)
   if a.ChangePassword("new-password", "next-password", "next-password") == nil { t.Fatal("expired session changed password") }
  })
 }
}
