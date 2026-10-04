const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {chromium} = require(process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES ? path.join(process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES, 'playwright') : 'playwright');
(async()=>{
 const browser=await chromium.launch({headless:true,args:['--no-sandbox']});
 try {
  const page=await browser.newPage();
  await page.setContent('<div id="output"></div>');
  const source=fs.readFileSync(path.join(__dirname,'../dist/app.js'),'utf8');
  const sanitizer=source.slice(source.indexOf('function sanitizeNotesHTML(input)'),source.indexOf('function notesEditorPlainText'));
  await page.addScriptTag({content:sanitizer});
  const cases=[
   '<section><div onclick="window.compromised=1"><img src=x onerror="window.compromised=1"><b>safe</b></div></section>',
   '<svg><foreignObject><div onmouseover="alert(1)">bad</div></foreignObject></svg><p>safe</p>',
   '<math><mtext><img src=x onerror=alert(1)></mtext></math><p>safe</p>',
   '<template><script>window.compromised=1</script></template><p>safe</p>',
   '<div style="color:red; font-family:url(https://evil); position:fixed" id=x><a href="javascript:alert(1)">safe</a></div>',
   '<font face="Arial" color="red" onclick="alert(1)">safe</font><iframe srcdoc="bad"></iframe>',
   '<section><section><span onfocus="alert(1)" tabindex="0">safe</span></section></section>',
  ];
  for(const input of cases){
   const result=await page.evaluate(input=>{
    const clean=sanitizeNotesHTML(input);
    document.querySelector('#output').innerHTML=clean;
    return {clean,again:sanitizeNotesHTML(clean),bad:document.querySelector('#output').querySelector('script,img,svg,math,iframe,template,[onclick],[onerror],[onfocus],[tabindex],[href],[id]')!==null,compromised:!!window.compromised};
   },input);
   assert.equal(result.bad,false,result.clean);assert.equal(result.compromised,false);assert.equal(result.again,result.clean);assert.match(result.clean,/safe/);
  }
  const formatted=await page.evaluate(()=>sanitizeNotesHTML('<p><b>Bold</b><i>Italic</i></p><ul><li>Item</li></ul><span style="color:#172a47;font-family:Arial">Color</span>'));
  assert.match(formatted,/<b>Bold<\/b>/);assert.match(formatted,/font-family: Arial/);
  console.log('PASS: nested attacks, foreign namespaces, active attributes, CSS, stable output, preserved formatting');
 } finally {await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
