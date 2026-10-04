const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '../dist/app.js'), 'utf8');
const chunk = source.slice(source.indexOf('function installAutomaticLock()'), source.indexOf('let clientCommunicationsRows'));
(async () => {
 for (const minutes of [15,30,45]) {
  let now = 0, locks = 0, tick; const listeners = {};
  const box = {Date:{now:()=>now},currentUser:{username:'test'},lockingSession:false,lastDesktopInput:0,lastActivityPing:0,AUTO_LOCK_MS:minutes*60000,lockSession:async()=>locks++,backend:()=>({TouchActivity:async()=>true,SessionActive:async()=>true}),document:{addEventListener:(n,f)=>listeners[n]=f},window:{addEventListener:(n,f)=>listeners[n]=f},setInterval:f=>tick=f};
  vm.createContext(box); vm.runInContext(chunk+';installAutomaticLock();',box);
  now=15000; await tick(); assert.equal(locks,0);
  now=minutes*60000-1; await tick(); assert.equal(locks,0);
  now=minutes*60000; await tick(); assert.equal(locks,1);
  assert.equal(listeners.blur,undefined); assert.equal(listeners.visibilitychange,undefined);
 }
 console.log('PASS: 15/30/45 minute expiry and no short focus/visibility logout');
})().catch(e=>{console.error(e);process.exitCode=1;});
