package main

import (
 "errors"
 "strings"
)

func updateCalendarAssignmentStatus(items []storedFirmCalendarAssignment, date, clientID, status string) error {
 if status != "" && status != "drafting" && status != "waiting-on-reports" && status != "completed" && status != "8879-sent" && status != "waiting-for-signature" && status != "signed" && status != "ready-to-e-file" { return errors.New("invalid calendar status") }
 for i := range items {
  if items[i].Date == date && items[i].ClientID == clientID { items[i].Status = status; return nil }
 }
 return errors.New("calendar assignment could not be found")
}

// Status belongs to one scheduled assignment and follows it when rescheduled.
// Calendar metadata is encrypted with the signed-in company's vault key.
func (a *App) SetFirmCalendarStatus2026(date, clientID, status string) ([]FirmCalendarAssignment, error) {
 if err := a.requireSignedIn(); err != nil { return nil, err }
 if err := validateFirmCalendarDate(date); err != nil { return nil, err }
 clientID = strings.TrimSpace(clientID)
 if clientID == "" { return nil, errors.New("client ID is required") }
 a.calendarMu.Lock()
 defer a.calendarMu.Unlock()
 items, err := a.loadFirmCalendarUnlocked()
 if err != nil { return nil, err }
 if err := updateCalendarAssignmentStatus(items,date,clientID,status); err != nil { return nil, err }
 if err := a.saveFirmCalendarUnlocked(items); err != nil { return nil, err }
 return a.resolveFirmCalendarAssignmentsUnlocked(items)
}
