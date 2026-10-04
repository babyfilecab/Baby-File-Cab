package main

import (
 "os"
 "path/filepath"
 "testing"
 "time"
)

func TestSectionsStayIndependentAndRequireEmptyDeletion(t *testing.T) {
 t.Setenv("HOME", t.TempDir())
 a, err := NewApp()
 if err != nil { t.Fatal(err) }
 _, err = a.RegisterUser(UserRegistration{CompanyName: "Section Test", Username: "sectionadmin", FirstName: "Test", LastName: "User", Email: "test@example.test", Phone: "5551234", Password: "test-admin-password", ConfirmPassword: "test-admin-password", Role: "administrator"})
 if err != nil { t.Fatal(err) }
 client, err := a.AddClient("Section Client", "", "", "", "", "1040")
 if err != nil { t.Fatal(err) }
 original, err := a.AddSection(client, "Original")
 if err != nil { t.Fatal(err) }
 file := filepath.Join(original, "document.txt")
 if err := a.secureWriteFile(file, []byte("keep in original")); err != nil { t.Fatal(err) }
 created, err := a.AddSection(original, "New Section")
 if err != nil { t.Fatal(err) }
 if filepath.Dir(created) != client { t.Fatalf("new section is nested: %s", created) }
 entries, err := os.ReadDir(created)
 if err != nil || len(entries) != 0 { t.Fatalf("new section not empty: %v %v", entries, err) }
 if _, err := os.Stat(file); err != nil { t.Fatalf("original document moved: %v", err) }
 if err := a.DeleteSection(original); err == nil { t.Fatal("nonempty section deleted") }
 if _, err := os.Stat(file); err != nil { t.Fatal("rejected deletion lost document") }
 if err := a.DeleteSection(created); err != nil { t.Fatal(err) }
 if err := os.Remove(file); err != nil { t.Fatal(err) }
 if err := a.DeleteSection(original); err != nil { t.Fatal(err) }
 a.authMu.Lock()
 a.lastActivity = time.Now().Add(-14 * time.Minute)
 a.authMu.Unlock()
 if !a.SessionActive() { t.Fatal("session expires before fifteen minutes") }
 a.authMu.Lock()
 a.lastActivity = time.Now().Add(-16 * time.Minute)
 a.authMu.Unlock()
 if a.SessionActive() { t.Fatal("session remains active after fifteen minutes") }
}
