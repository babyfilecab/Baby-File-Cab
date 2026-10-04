package main

import (
 "testing"
 "time"
 "encoding/json"
)

func TestCalendarPreparerScopeAndReassignment(t *testing.T) {
 a:=&App{storageRoot:t.TempDir(),autoLockMinutes:15,currentUser:&UserProfile{Username:"alice",CompanyID:"firm-a"},lastActivity:time.Now()}
 users:=[]storedUser{{UserProfile:UserProfile{Username:"Alice",CompanyID:"firm-a"}},{UserProfile:UserProfile{Username:"Bob",CompanyID:"firm-a"}},{UserProfile:UserProfile{Username:"Other",CompanyID:"firm-b"}}}
 if err:=a.saveUsersUnlocked(users);err!=nil{t.Fatal(err)}
 username,err:=a.validateCalendarPreparer(" bob ");if err!=nil || username!="Bob"{t.Fatal("same-firm preparer rejected",err)}
 if _,err:=a.validateCalendarPreparer("Other");err==nil{t.Fatal("cross-company assignment accepted")}
 if _,err:=a.validateCalendarPreparer("missing");err==nil{t.Fatal("unknown preparer accepted")}
 if username,err:=a.validateCalendarPreparer("");err!=nil || username!=""{t.Fatal("unassigned rejected")}
 items:=[]storedFirmCalendarAssignment{{Date:"2026-01-01",ClientID:"1",Status:"completed",PreparerUsername:"Alice"},{Date:"2026-02-01",ClientID:"1",PreparerUsername:"Alice"}}
 if err:=updateCalendarPreparer(items,"2026-01-01","1","Bob");err!=nil{t.Fatal(err)}
 if items[0].PreparerUsername!="Bob" || items[0].Status!="completed" || items[1].PreparerUsername!="Alice"{t.Fatal("reassignment changed wrong data")}
 if updateCalendarPreparer(items,"2026-03-01","1","Bob")==nil{t.Fatal("missing assignment accepted")}
 data,err:=json.Marshal(items);if err!=nil{t.Fatal(err)}
 var restored []storedFirmCalendarAssignment
 if err:=json.Unmarshal(data,&restored);err!=nil{t.Fatal(err)}
 if restored[0].PreparerUsername!="Bob"{t.Fatal("preparer lost during serialization")}
 a.lastActivity=time.Now().Add(-16*time.Minute)
 if _,err:=a.validateCalendarPreparer("Bob");err==nil{t.Fatal("expired session accepted")}
}
