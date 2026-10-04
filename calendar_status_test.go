package main

import (
 "encoding/json"
 "testing"
)

func TestCalendarStatusUpdatesOnlySelectedAssignment(t *testing.T) {
 items := []storedFirmCalendarAssignment{{Date:"2026-01-01",ClientID:"1"},{Date:"2026-02-01",ClientID:"1"},{Date:"2026-01-01",ClientID:"2"}}
 for _, status := range []string{"drafting","waiting-on-reports","8879-sent","waiting-for-signature","signed","ready-to-e-file","completed"} {
  if err:=updateCalendarAssignmentStatus(items,"2026-01-01","1",status);err!=nil{t.Fatal(err)}
  if items[0].Status!=status || items[1].Status!="" || items[2].Status!="" {t.Fatal("wrong assignment changed")}
 }
 if updateCalendarAssignmentStatus(items,"2026-01-01","1","invalid")==nil {t.Fatal("invalid status accepted")}
 if updateCalendarAssignmentStatus(items,"2026-03-01","1","completed")==nil {t.Fatal("missing assignment accepted")}
 if err:=updateCalendarAssignmentStatus(items,"2026-01-01","1","");err!=nil{t.Fatal(err)}
 if items[0].Status!=""{t.Fatal("clear did not remove status")}
 if err:=updateCalendarAssignmentStatus(items,"2026-01-01","1","completed");err!=nil{t.Fatal(err)}
 data,err:=json.Marshal(items);if err!=nil{t.Fatal(err)}
 var restored []storedFirmCalendarAssignment
 if err:=json.Unmarshal(data,&restored);err!=nil{t.Fatal(err)}
 if restored[0].Status!="completed" {t.Fatal("status lost in serialization")}
 var legacy storedFirmCalendarAssignment
 if err:=json.Unmarshal([]byte(`{"date":"2026-01-01","clientId":"1"}`),&legacy);err!=nil{t.Fatal(err)}
 if legacy.Status!="" {t.Fatal("legacy assignment changed")}
}
