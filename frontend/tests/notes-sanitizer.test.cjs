const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'../dist/app.js'),'utf8');
const chunk=source.slice(source.indexOf('function sanitizeNotesHTML(input)'),source.indexOf('function notesEditorPlainText'));
const escape=s=>String(s).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('"','&quot;');
class Element {
 constructor(tag,attrs={},children=[],namespace='http://www.w3.org/1999/xhtml'){this.nodeType=1;this.tagName=tag.toUpperCase();this.attrs={...attrs};this.childNodes=children;this.namespaceURI=namespace;}
 appendChild(child){this.childNodes.push(child);return child;}
 getAttribute(key){return this.attrs[key]??null;}
 setAttribute(key,value){this.attrs[key]=value;}
 get innerHTML(){return this.childNodes.map(c=>c.nodeType===3?escape(c.textContent):`<${c.tagName.toLowerCase()}${Object.entries(c.attrs).map(([k,v])=>` ${k}="${escape(v)}"`).join('')}>${c.innerHTML}</${c.tagName.toLowerCase()}>`).join('');}
}
const text=value=>({nodeType:3,textContent:value});
let parsed;
const document={createTextNode:text,createElement(tag){if(tag==='template'){return {content:{childNodes:parsed},set innerHTML(value){}};}return new Element(tag);}};
const box={document,Node:{ELEMENT_NODE:1,TEXT_NODE:3}};vm.createContext(box);vm.runInContext(chunk,box);
// Explicit parser-tree fixtures test traversal and attribute policy. Browser
// parsing and round-trip cases are covered separately by the Playwright script.
parsed=[new Element('section',{},[new Element('aside',{},[
 new Element('div',{onclick:'attack()',id:'evil',style:'color:red;position:fixed;font-family:url(https://evil)'},[
 new Element('img',{src:'evil',onerror:'attack()'}),new Element('b',{},[text('safe')])])])])];
let clean=box.sanitizeNotesHTML('fixture');
assert.match(clean,/<b>safe<\/b>/);assert.match(clean,/color: red/);
for(const forbidden of ['onclick','onerror','id=','<img','position:','url('])assert.equal(clean.includes(forbidden),false,clean);
parsed=[new Element('svg',{},[new Element('div',{},[text('bad')])],'http://www.w3.org/2000/svg'),new Element('template',{},[new Element('script',{},[text('bad')])]),new Element('font',{face:'Arial',color:'#172a47',onmouseover:'attack()'},[text('safe')])];
clean=box.sanitizeNotesHTML('fixture');assert.match(clean,/face="Arial"/);assert.match(clean,/color="#172a47"/);assert.equal(clean.includes('bad'),false);assert.equal(clean.includes('onmouseover'),false);
parsed=[new Element('div',{style:'font-weight:bold;font-style:italic;text-decoration:underline;color:var(--evil);font-family:Arial'},[text('<script>literal text</script>')])];
clean=box.sanitizeNotesHTML('fixture');assert.match(clean,/font-weight: bold/);assert.match(clean,/&lt;script>/);assert.equal(clean.includes('var('),false);
assert.equal(box.sanitizeNotesHTML('x'.repeat(2*1024*1024+1)),'');
let nested=new Element('b',{},[text('safe')]);for(let i=0;i<130;i++)nested=new Element('section',{},[nested]);parsed=[nested];assert.throws(()=>box.sanitizeNotesHTML('fixture'),/nesting/);
console.log('PASS: nested-wrapper traversal, strict attributes/CSS, namespaces, text escaping and limits');
