package main

import (
 "errors"
 "strings"
 "time"
)

// Empty means unassigned. Resolve only against this signed-in company's users.
func (a *App) validateCalendarPreparer(username string) (string,error) {
 a.authMu.RLock()
 defer a.authMu.RUnlock()
 if a.currentUser == nil || time.Since(a.lastActivity) >= a.idleTimeoutLocked() { return "", errors.New("sign in to BabyFileCab first") }
 username = strings.TrimSpace(username)
 if username == "" { return "",nil }
 users,err := a.loadUsersUnlocked()
 if err != nil { return "",err }
 for _,user := range users {
  if user.CompanyID == a.currentUser.CompanyID && strings.EqualFold(user.Username,username) { return user.Username,nil }
 }
 return "",errors.New("choose a preparer registered with this company")
}

func updateCalendarPreparer(items []storedFirmCalendarAssignment,date,clientID,username string) error {
 for i := range items {
  if items[i].Date==date && items[i].ClientID==clientID { items[i].PreparerUsername=username;return nil }
 }
 return errors.New("calendar assignment could not be found")
}

func (a *App) ReassignFirmCalendarPreparer2026(date,clientID,username string) ([]FirmCalendarAssignment,error) {
 if err:=validateFirmCalendarDate(date);err!=nil{return nil,err}
 username,err:=a.validateCalendarPreparer(username)
 if err!=nil{return nil,err}
 a.calendarMu.Lock()
 defer a.calendarMu.Unlock()
 items,err:=a.loadFirmCalendarUnlocked()
 if err!=nil{return nil,err}
 if err:=updateCalendarPreparer(items,date,strings.TrimSpace(clientID),username);err!=nil{return nil,err}
 if err:=a.saveFirmCalendarUnlocked(items);err!=nil{return nil,err}
 return a.resolveFirmCalendarAssignmentsUnlocked(items)
}
