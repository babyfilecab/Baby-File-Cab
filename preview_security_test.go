package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestNotesSanitizerRejectsNestedActiveContent(t *testing.T) {
 inputs := []string{
  `<section><div onclick="alert(1)"><img src=x onerror=alert(1)><b>safe</b></div></section>`,
  `<svg><foreignObject><div onmouseover="alert(1)">bad</div></foreignObject></svg><p>safe</p>`,
  `<math><mtext><img src=x onerror=alert(1)></mtext></math><p>safe</p>`,
  `<template><script>alert(1)</script></template><p>safe</p>`,
  `<div style="color:red; font-family:url(https://evil); position:fixed" id=x><a href="javascript:alert(1)">safe</a></div>`,
  `<font face="Arial" color="red" onclick="alert(1)">safe</font><iframe srcdoc="bad"></iframe>`,
 }
 for _, input := range inputs {
  clean, err := sanitizeNotesHTML(input)
  if err != nil { t.Fatal(err) }
  for _, bad := range []string{"onclick","onerror","onmouseover","<img","<script","<svg","<math","<template","<iframe","href=","url(","position:","id="} {
   if strings.Contains(clean,bad) { t.Fatalf("unsafe output: %s",clean) }
  }
  if !strings.Contains(clean,"safe") { t.Fatalf("lost safe text: %s",clean) }
  again, err := sanitizeNotesHTML(clean)
  if err != nil || again != clean { t.Fatalf("unstable sanitization: %s -> %s",clean,again) }
 }
 clean, err := sanitizeNotesHTML(`<p><b>Bold</b><i>Italic</i><u>Underline</u></p><ul><li>Item</li></ul><span style="color: #172a47; font-family: Arial">Color</span>`)
 if err != nil || !strings.Contains(clean,"<b>Bold</b>") || !strings.Contains(clean,"font-family: Arial") { t.Fatal("formatting lost",clean,err) }
 if _, err := sanitizeNotesHTML(strings.Repeat("x",2*1024*1024+1)); err == nil { t.Fatal("oversized notes accepted") }
}

func TestLoginMissingKeyNeverCreatesReplacement(t *testing.T) {
 for _, versioned := range []bool{false,true} {
  for _, keyDirectoryExists := range []bool{false,true} {
   a := &App{storageRoot:t.TempDir(),autoLockMinutes:15}
   company := CompanyProfile{ID:"company",Name:"Company"}
   user, _, err := buildStoredUser(UserRegistration{Username:"admin",Password:"test-password",Role:"administrator"},company)
   if err != nil { t.Fatal(err) }
   if versioned { user.KeyWrapID="missing-version" }
   if err := a.saveUsersUnlocked([]storedUser{user}); err != nil { t.Fatal(err) }
   if err := a.saveCompaniesUnlocked([]CompanyProfile{company}); err != nil { t.Fatal(err) }
   keyDir := filepath.Dir(a.wrappedKeyPath(company.ID,user.Username))
   if keyDirectoryExists { if err:=os.MkdirAll(keyDir,0700);err!=nil{t.Fatal(err)} }
   // A correct password with a missing key must fail before any migration/write.
   before, err := os.ReadFile(a.usersPath())
   if err != nil { t.Fatal(err) }
   if _, err := a.Login("admin","test-password"); err == nil || !strings.Contains(err.Error(),"no replacement key") { t.Fatal("missing key did not fail closed",err) }
   after, err := os.ReadFile(a.usersPath())
   if err != nil || string(before)!=string(after) { t.Fatal("failed login changed account") }
   entries, err := os.ReadDir(keyDir)
   if err != nil && !os.IsNotExist(err) { t.Fatal(err) }
   if len(entries)!=0 || a.currentUser!=nil || len(a.vaultKey)!=0 { t.Fatal("failed login created a key/session") }
  }
 }
}
