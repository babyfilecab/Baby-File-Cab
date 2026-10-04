const assert=require('node:assert/strict');const fs=require('node:fs');const path=require('node:path');const vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'../dist/app.js'),'utf8');
const chunk=source.slice(source.indexOf('let firmCalendarPreparers ='),source.indexOf('function renderCalendarPreparerMenu'));
const box={};vm.createContext(box);vm.runInContext(chunk,box);
vm.runInContext("firmCalendarPreparers=[{username:'alice',firstName:'Alice',lastName:'Smith'},{username:'bob',firstName:'Bob',lastName:'Jones'}]",box);
assert.equal(box.calendarPreparerName('ALICE'),'Alice Smith');assert.equal(box.calendarPreparerInitials('alice'),'AS');assert.equal(box.calendarPreparerName(''),'Unassigned');assert.equal(box.calendarPreparerName('former'),'former');
const items=[{clientId:'1',preparerUsername:'alice'},{clientId:'2',preparerUsername:'Bob'},{clientId:'3'}];
assert.equal(box.calendarAssignmentsForPreparer(items,'*').length,3);assert.equal(box.calendarAssignmentsForPreparer(items,'ALICE')[0].clientId,'1');assert.equal(box.calendarAssignmentsForPreparer(items,'bob')[0].clientId,'2');assert.equal(box.calendarAssignmentsForPreparer(items,'')[0].clientId,'3');assert.equal(box.calendarAssignmentsForPreparer(items,'missing').length,0);
