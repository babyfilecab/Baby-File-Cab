package main

import (
 "testing"
 "time"
)

func TestAutoLockChoicesPersistAndExpire(t *testing.T) {
 t.Setenv("HOME", t.TempDir())
 a, err := NewApp()
 if err != nil { t.Fatal(err) }
 if a.GetAutoLockMinutes() != 15 { t.Fatal("default must be 15 minutes") }
 a.currentUser = &UserProfile{}
 for _, minutes := range []int{15, 30, 45} {
  a.lastActivity = time.Now()
  if err := a.SetAutoLockMinutes(minutes); err != nil { t.Fatal(err) }
  restarted, err := NewApp()
  if err != nil { t.Fatal(err) }
  if restarted.GetAutoLockMinutes() != minutes { t.Fatal("choice not persisted") }
  a.lastActivity = time.Now().Add(-time.Duration(minutes-1)*time.Minute)
  if !a.SessionActive() { t.Fatal("session expired early") }
  a.lastActivity = time.Now().Add(-time.Duration(minutes+1)*time.Minute)
  if a.SessionActive() { t.Fatal("session did not expire") }
 }
 a.lastActivity = time.Now()
 if a.SetAutoLockMinutes(0) == nil || a.SetAutoLockMinutes(60) == nil { t.Fatal("invalid timeout accepted") }
 if a.GetAutoLockMinutes() != 45 { t.Fatal("invalid choice changed timeout") }
 a.lastActivity = time.Now().Add(-46*time.Minute)
 if a.SetAutoLockMinutes(45) == nil { t.Fatal("expired session revived through settings") }
}
