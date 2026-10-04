const SHORTCUT_ACTIONS = [
  {id:'help',label:'Help / User manual',keys:['F1']},
  {id:'global-find',label:'Global Find',keys:['Ctrl+Shift+F']},
  {id:'pdf-find',label:'Find in PDF',keys:['Ctrl+F','Shift+M']},
  {id:'sign-out',label:'Sign out / Lock',keys:['Ctrl+Shift+L']},
  {id:'exit',label:'Exit BabyFileCab',keys:['Ctrl+Q']},
  {id:'backup-client',label:'Back up client',keys:['Ctrl+B']},
  {id:'refresh',label:'Refresh workspace',keys:['F5']},
  {id:'rename-file',label:'Rename document',keys:['F2','Shift+F']},
  {id:'upload',label:'Upload files',keys:['F']},
  {id:'add-client',label:'Add client',keys:['C']},
  {id:'edit-client',label:'Edit client',keys:['Shift+C']},
  {id:'add-year',label:'Add tax year',keys:['Y']},
  {id:'edit-year',label:'Edit tax year',keys:['Shift+Y']},
  {id:'add-section',label:'Add section',keys:['S']},
  {id:'edit-section',label:'Edit section',keys:['Shift+S']},
  {id:'delete',label:'Delete selected item(s)',keys:['D','Shift+D']},
  {id:'manage-users',label:'Manage users',keys:['M']},
];
function defaultShortcutBindings() { return Object.fromEntries(SHORTCUT_ACTIONS.map(item => [item.id, [...item.keys]])); }
let shortcutBindings = defaultShortcutBindings();
let shortcutDraft = null;
const SHORTCUT_KEYS = [
  ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', ...'0123456789',
  ...Array.from({length:24}, (_,i) => `F${i+1}`),
  'Space','Enter','Tab','Escape','Backspace','Delete','Insert','Home','End','PageUp','PageDown','ArrowUp','ArrowDown','ArrowLeft','ArrowRight',
  'Backquote','Minus','Equal','BracketLeft','BracketRight','Backslash','Semicolon','Quote','Comma','Period','Slash',
  ...Array.from({length:10},(_,i) => `Numpad${i}`),
  'NumpadAdd','NumpadSubtract','NumpadMultiply','NumpadDivide','NumpadDecimal','NumpadEnter','NumpadEqual',
];
function shortcutFromEvent(event) {
  if (event.isComposing) return '';
  let key = String(event.key || '');
  if (/^Numpad/.test(event.code || '')) key = event.code;
  else if (['Backquote','Minus','Equal','BracketLeft','BracketRight','Backslash','Semicolon','Quote','Comma','Period','Slash'].includes(event.code)) key = event.code;
  else if (/^Digit[0-9]$/.test(event.code || '')) key = event.code.slice(5);
  else if (key === ' ') key = 'Space';
  else if (/^[a-z0-9]$/i.test(key) || /^F[0-9]+$/i.test(key)) key = key.toUpperCase();
  if (!SHORTCUT_KEYS.includes(key)) return '';
  return [(event.ctrlKey || event.metaKey) ? 'Ctrl' : '', event.altKey ? 'Alt' : '', event.shiftKey ? 'Shift' : '', key].filter(Boolean).join('+');
}
function shortcutAllowed(binding) {
  if (typeof binding !== 'string') return false;
  const parts = binding.split('+'); const key = parts.pop();
  const canonical = ['Ctrl','Alt','Shift'].filter(modifier => parts.includes(modifier));
  return SHORTCUT_KEYS.includes(key) && parts.join('+') === canonical.join('+') && !['G','Escape','Ctrl+S','Ctrl+I','Ctrl+U','Alt+F4'].includes(binding);
}
function validateShortcutBindings(bindings) {
  const used = new Map();
  for (const item of SHORTCUT_ACTIONS) {
    const keys = bindings?.[item.id];
    if (!Array.isArray(keys) || keys.length > 4) throw new Error('Invalid shortcut settings.');
    for (const key of keys) {
      if (typeof key !== 'string' || !shortcutAllowed(key)) throw new Error('That shortcut is reserved or unsupported.');
      if (used.has(key)) throw new Error(`${key} is already assigned to ${used.get(key)}.`);
      used.set(key, item.label);
    }
  }
  return bindings;
}
function shortcutStorageKey() { return 'babyfilecab-shortcuts-v1:' + JSON.stringify([currentUser.companyId, currentUser.username.toLowerCase()]); }
function loadShortcutBindings() {
  shortcutBindings = defaultShortcutBindings();
  try {
    const stored = localStorage.getItem(shortcutStorageKey());
    if (stored) shortcutBindings = validateShortcutBindings(JSON.parse(stored));
  } catch (error) { toast('Could not load saved shortcuts; using defaults.', true); }
  updateShortcutHints();
}
function updateShortcutHints() {
  const menuIds = {'help':'user-manual','rename-file':'rename-selected-document','delete':'delete-selected'};
  for (const item of SHORTCUT_ACTIONS) {
    const menu = document.querySelector(`[data-menu-action="${menuIds[item.id] || item.id}"]`);
    if (menu) {
      let hint = menu.querySelector('.menu-shortcut');
      if (!hint) { hint = document.createElement('span'); hint.className = 'menu-shortcut'; menu.appendChild(hint); }
      hint.textContent = shortcutBindings[item.id].join(' / ');
    }
  }
  const grid = $('mappedShortcutHelp');
  if (grid) {
    grid.replaceChildren();
    for (const item of SHORTCUT_ACTIONS) {
      const key = document.createElement('kbd'); key.textContent = shortcutBindings[item.id].join(' / ') || 'Disabled';
      const label = document.createElement('span'); label.textContent = item.label;
      grid.append(key, label);
    }
  }
}
function showShortcutMapper() {
  shortcutDraft = structuredClone(shortcutBindings);
  renderShortcutMapper();
  $('shortcutMapperBackdrop').classList.remove('hidden');
  $('shortcutMapperCancel').focus();
}
let shortcutSelectedRow = null;
let shortcutEditingRow = null;
function hideShortcutMapper() {
  hideShortcutEditor(); $('shortcutMapperBackdrop').classList.add('hidden'); shortcutDraft = null; shortcutSelectedRow = null;
}
function renderShortcutMapper() {
  const body = $('shortcutMapperRows'); body.replaceChildren(); shortcutSelectedRow = null;
  for (const id of ['shortcutMapperModify','shortcutMapperClear']) $(id).disabled = true;
  let number = 0;
  for (const item of SHORTCUT_ACTIONS) {
    const keys = shortcutDraft[item.id].length ? shortcutDraft[item.id] : [''];
    keys.forEach((key,index) => {
      const row = document.createElement('tr'); row.tabIndex = 0; row.setAttribute('aria-selected','false');
      const count = document.createElement('td'); count.textContent = String(++number);
      const label = document.createElement('td'); label.textContent = item.label;
      const binding = document.createElement('td'); binding.textContent = key || 'Disabled';
      row.append(count,label,binding);
      const select = () => {
        body.querySelectorAll('tr').forEach(other => { other.classList.remove('is-selected'); other.setAttribute('aria-selected','false'); });
        row.classList.add('is-selected'); row.setAttribute('aria-selected','true');
        shortcutSelectedRow = {id:item.id,index};
        for (const id of ['shortcutMapperModify','shortcutMapperClear']) $(id).disabled = false;
      };
      row.addEventListener('click',select);
      row.addEventListener('keydown',e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); select(); } });
      row.addEventListener('dblclick', () => { select(); showShortcutEditor(false); });
      body.appendChild(row);
    });
  }
}
function showShortcutEditor() {
  if (!shortcutSelectedRow || !shortcutDraft) return;
  shortcutEditingRow = {...shortcutSelectedRow};
  const item = SHORTCUT_ACTIONS.find(item => item.id === shortcutEditingRow.id);
  $('shortcutEditName').textContent = item.label;
  const binding = shortcutDraft[item.id][shortcutEditingRow.index] || 'A';
  const parts = binding.split('+'); const key = parts.pop();
  for (const modifier of ['Ctrl','Alt','Shift']) $(`shortcutEdit${modifier}`).checked = parts.includes(modifier);
  const select = $('shortcutEditKey'); select.replaceChildren();
  for (const key of SHORTCUT_KEYS) { const option = document.createElement('option'); option.value = key; option.textContent = key.startsWith('Numpad') ? key.replace('Numpad','Numpad ') : key; select.appendChild(option); }
  select.value = key;
  $('shortcutEditError').textContent = ''; updateShortcutEditorPreview();
  $('shortcutEditBackdrop').classList.remove('hidden'); select.focus();
}
function shortcutEditorBinding() {
  return [...['Ctrl','Alt','Shift'].filter(modifier => $(`shortcutEdit${modifier}`).checked), $('shortcutEditKey').value].join('+');
}
function updateShortcutEditorPreview() { $('shortcutEditPreview').textContent = shortcutEditorBinding(); $('shortcutEditError').textContent = ''; }
function hideShortcutEditor() { $('shortcutEditBackdrop').classList.add('hidden'); shortcutEditingRow = null; }
function applyShortcutEditor() {
  if (!shortcutEditingRow || !shortcutDraft) return;
  const next = structuredClone(shortcutDraft);
  next[shortcutEditingRow.id][shortcutEditingRow.index] = shortcutEditorBinding();
  try { validateShortcutBindings(next); shortcutDraft = next; hideShortcutEditor(); renderShortcutMapper(); $('shortcutMapperSave').focus(); }
  catch (error) { $('shortcutEditError').textContent = error.message; }
}
function saveShortcutMapper() {
  try {
    validateShortcutBindings(shortcutDraft);
    localStorage.setItem(shortcutStorageKey(), JSON.stringify(shortcutDraft));
    shortcutBindings = structuredClone(shortcutDraft);
    updateShortcutHints(); hideShortcutMapper(); toast('Keyboard shortcuts saved.');
  } catch (error) { showError(error); }
}
async function runMappedShortcut(action) {
  switch (action) {
    case 'help': showUserManual(); break;
    case 'global-find': showGlobalFind(); break;
    case 'pdf-find': focusPDFFind(); break;
    case 'sign-out':
      document.querySelectorAll('.modal-backdrop').forEach(el => el.classList.add('hidden'));
      await signOut(); break;
    case 'refresh': await refreshTree(selectedNode?.path || null); toast('Workspace refreshed.'); break;
    case 'manage-users': await showManageUsers(); break;
    case 'delete': await deleteHighlightedItem(); break;
    case 'exit': case 'backup-client': await handleMenuAction(action); break;
    case 'upload':
      if (selectedNode && ['client','permanent','taxyear','section'].includes(selectedNode.type)) await doAction('upload');
      else toast('Select a client, Permanent Folder, tax year, or section first.', true);
      break;
    default: await doAction(action);
  }
}

let treeData = [];
let selectedNode = null;
let selectedElement = null;
let selectedFilePaths = new Set();
let fileSelectionAnchor = null;
let expanded = new Set();
let pdfPage = 1;
let currentPDF = null;
let pdfZoom = 1.0;
let pdfRotation = 0;
let pdfSearchResults = [];
let pdfSearchIndex = -1;
let pdfSearchTimer = null;
let pdfSearchQuery = '';
let modalResolver = null;
let contextClient = null;
let contextTaxYear = null;
let contextDocument = null;
let contextSection = null;
let documentPropertiesPath = null;
let globalFindTimer = null;
let notesSelectionRange = null;
let clientFormResolver = null;
let internalDraggedPath = '';
let internalDraggedElement = null;
let clientPickerResolver = null;
let clientPickerItems = [];
let clientFormMode = 'new';
let clientFormClient = null;
let currentUser = null;
let sessionGeneration = 0;
let pendingVaultEnrollmentUser = '';
let lastDesktopInput = Date.now();
let lastActivityPing = 0;
let lockingSession = false;
let AUTO_LOCK_MS = 15 * 60 * 1000;

async function lockSession() {
  if (!currentUser || lockingSession) return;
  lockingSession = true;
  const username = currentUser.username;
  try { await signOut(); } finally {
    $('authUsername').value = username || '';
    lockingSession = false;
  }
}

function installAutomaticLock() {
  const noteInput = () => {
    if (!currentUser || lockingSession) return;
    if (Date.now() - lastDesktopInput >= AUTO_LOCK_MS) { lockSession(); return; }
    lastDesktopInput = Date.now();
    if (Date.now() - lastActivityPing > 10000) {
      lastActivityPing = Date.now();
      backend().TouchActivity().then(active => { if (!active) lockSession(); }).catch(() => lockSession());
    }
  };
  for (const event of ['keydown', 'pointerdown', 'pointermove', 'wheel', 'touchstart']) {
    document.addEventListener(event, noteInput, { passive: true, capture: true });
  }
  // Native session monitoring handles sleep and account switching. Losing
  // focus or minimizing the app is not a reason to expire a ten-minute session.
  window.addEventListener('focus', () => {
    if (currentUser && Date.now() - lastDesktopInput >= AUTO_LOCK_MS) lockSession();
  });
  setInterval(async () => {
    if (!currentUser || lockingSession) return;
    if (Date.now() - lastDesktopInput >= AUTO_LOCK_MS) { await lockSession(); return; }
    try { if (!await backend().SessionActive()) await lockSession(); }
    catch (_) { await lockSession(); }
  }, 5000);
}
let clientCommunicationsRows = [];
let currentTheme = 'light';
let shortcutChord = '';
let shortcutChordTimer = null;
let firmCalendarAssignments = [];
let firmCalendarWeekStarts = [];
let firmCalendarWeekIndex = 0;
let firmCalendarSelectedClientID = '';
let firmCalendarContextAssignment = null;
let appAuditRows = [];
let engagementLetterUsers = [];
let engagementLetterClients = [];
let engagementLogoDataURL = '';
let invoiceClients = [];
let invoiceUsers = [];
let invoiceLogoDataURL = '';

const $ = (id) => document.getElementById(id);
const backend = () => window.go && window.go.main && window.go.main.App;


const TAX_YEARS = {
  2025: {
    standardDeduction: {
      single: 15750,
      mfj: 31500,
      mfs: 15750,
      hoh: 23625,
      qss: 31500,
    },
    ordinaryBrackets: {
      single: [[11925, .10], [48475, .12], [103350, .22], [197300, .24], [250525, .32], [626350, .35], [Infinity, .37]],
      mfj: [[23850, .10], [96950, .12], [206700, .22], [394600, .24], [501050, .32], [751600, .35], [Infinity, .37]],
      mfs: [[11925, .10], [48475, .12], [103350, .22], [197300, .24], [250525, .32], [375800, .35], [Infinity, .37]],
      hoh: [[17000, .10], [64850, .12], [103350, .22], [197300, .24], [250500, .32], [626350, .35], [Infinity, .37]],
      qss: [[23850, .10], [96950, .12], [206700, .22], [394600, .24], [501050, .32], [751600, .35], [Infinity, .37]],
    },
    capitalGains: {
      single: { zeroMax: 48350, fifteenMax: 533400 },
      mfj: { zeroMax: 96700, fifteenMax: 600050 },
      mfs: { zeroMax: 48350, fifteenMax: 300000 },
      hoh: { zeroMax: 64750, fifteenMax: 566700 },
      qss: { zeroMax: 96700, fifteenMax: 600050 },
    },
    capitalLossLimit: { single: 3000, mfj: 3000, mfs: 1500, hoh: 3000, qss: 3000 },
    additionalMedicareThreshold: { single: 200000, mfj: 250000, mfs: 125000, hoh: 200000, qss: 200000 },
    socialSecurityWageBase: 176100,
    childTaxCredit: {
      perChild: 2200,
      refundablePerChild: 1700,
      earnedIncomeThreshold: 2500,
      phaseoutThreshold: { single: 200000, mfj: 400000, mfs: 200000, hoh: 200000, qss: 200000 },
    },
    estimatedTaxDueDates: ['April 15, 2025', 'June 16, 2025', 'September 15, 2025', 'January 15, 2026'],
  },
  2026: {
    standardDeduction: {
      single: 16100,
      mfj: 32200,
      mfs: 16100,
      hoh: 24150,
      qss: 32200,
    },
    ordinaryBrackets: {
      single: [[12400, .10], [50400, .12], [105700, .22], [201775, .24], [256225, .32], [640600, .35], [Infinity, .37]],
      mfj: [[24800, .10], [100800, .12], [211400, .22], [403550, .24], [512450, .32], [768700, .35], [Infinity, .37]],
      mfs: [[12400, .10], [50400, .12], [105700, .22], [201775, .24], [256225, .32], [384350, .35], [Infinity, .37]],
      hoh: [[17700, .10], [67450, .12], [105700, .22], [201750, .24], [256200, .32], [640600, .35], [Infinity, .37]],
      qss: [[24800, .10], [100800, .12], [211400, .22], [403550, .24], [512450, .32], [768700, .35], [Infinity, .37]],
    },
    capitalGains: {
      single: { zeroMax: 49450, fifteenMax: 545500 },
      mfj: { zeroMax: 98900, fifteenMax: 613700 },
      mfs: { zeroMax: 49450, fifteenMax: 306850 },
      hoh: { zeroMax: 66200, fifteenMax: 579600 },
      qss: { zeroMax: 98900, fifteenMax: 613700 },
    },
    capitalLossLimit: { single: 3000, mfj: 3000, mfs: 1500, hoh: 3000, qss: 3000 },
    additionalMedicareThreshold: { single: 200000, mfj: 250000, mfs: 125000, hoh: 200000, qss: 200000 },
    socialSecurityWageBase: 184500,
    childTaxCredit: {
      perChild: 2200,
      refundablePerChild: 1700,
      earnedIncomeThreshold: 2500,
      phaseoutThreshold: { single: 200000, mfj: 400000, mfs: 200000, hoh: 200000, qss: 200000 },
    },
    estimatedTaxDueDates: ['April 15, 2026', 'June 15, 2026', 'September 15, 2026', 'January 15, 2027'],
  },
};



// IRS yearly-average currency exchange rates published at:
// https://www.irs.gov/individuals/international-taxpayers/yearly-average-currency-exchange-rates
// The IRS table expresses each rate as units of foreign currency per 1 U.S. dollar.
const IRS_YEARLY_EXCHANGE_RATES = [
  { code: 'AFN', country: 'Afghanistan', currency: 'Afghani', rates: { 2025: 69.637, 2024: 70.649, 2023: 82.635, 2022: 90.084, 2021: 83.484 } },
  { code: 'DZD', country: 'Algeria', currency: 'Dinar', rates: { 2025: 131.627, 2024: 134.124, 2023: 135.933, 2022: 142.123, 2021: 135.011 } },
  { code: 'ARS', country: 'Argentina', currency: 'Peso', rates: { 2025: 1243.369, 2024: 915.161, 2023: 296.154, 2022: 130.792, 2021: 95.098 } },
  { code: 'AUD', country: 'Australia', currency: 'Dollar', rates: { 2025: 1.551, 2024: 1.516, 2023: 1.506, 2022: 1.442, 2021: 1.332 } },
  { code: 'BHD', country: 'Bahrain', currency: 'Dinar', rates: { 2025: 0.377, 2024: 0.377, 2023: 0.377, 2022: 0.377, 2021: 0.377 } },
  { code: 'BRL', country: 'Brazil', currency: 'Real', rates: { 2025: 5.593, 2024: 5.392, 2023: 4.994, 2022: 5.165, 2021: 5.395 } },
  { code: 'CAD', country: 'Canada', currency: 'Dollar', rates: { 2025: 1.398, 2024: 1.370, 2023: 1.350, 2022: 1.301, 2021: 1.254 } },
  { code: 'KYD', country: 'Cayman Islands', currency: 'Dollar', rates: { 2025: 0.821, 2024: 0.833, 2023: 0.833, 2022: 0.833, 2021: 0.833 } },
  { code: 'CNY', country: 'China', currency: 'Yuan', rates: { 2025: 7.129, 2024: 7.189, 2023: 7.075, 2022: 6.730, 2021: 6.452 } },
  { code: 'DKK', country: 'Denmark', currency: 'Krone', rates: { 2025: 6.617, 2024: 6.896, 2023: 6.890, 2022: 7.077, 2021: 6.290 } },
  { code: 'EGP', country: 'Egypt', currency: 'Pound', rates: { 2025: 49.233, 2024: 45.345, 2023: 30.651, 2022: 19.208, 2021: 15.697 } },
  { code: 'EUR', country: 'Euro Zone', currency: 'Euro', rates: { 2025: 0.886, 2024: 0.924, 2023: 0.924, 2022: 0.951, 2021: 0.846 } },
  { code: 'HKD', country: 'Hong Kong', currency: 'Dollar', rates: { 2025: 7.796, 2024: 7.803, 2023: 7.829, 2022: 7.831, 2021: 7.773 } },
  { code: 'HUF', country: 'Hungary', currency: 'Forint', rates: { 2025: 352.869, 2024: 365.603, 2023: 353.020, 2022: 372.775, 2021: 303.292 } },
  { code: 'ISK', country: 'Iceland', currency: 'Krona', rates: { 2025: 128.262, 2024: 137.958, 2023: 137.857, 2022: 135.296, 2021: 126.986 } },
  { code: 'INR', country: 'India', currency: 'Rupee', rates: { 2025: 87.133, 2024: 83.677, 2023: 82.572, 2022: 78.598, 2021: 73.936 } },
  { code: 'IQD', country: 'Iraq', currency: 'Dinar', rates: { 2025: 1309.753, 2024: 1309.744, 2023: 1376.529, 2022: 1459.510, 2021: 1460.133 } },
  { code: 'ILS', country: 'Israel', currency: 'New Shekel', rates: { 2025: 3.451, 2024: 3.701, 2023: 3.687, 2022: 3.361, 2021: 3.232 } },
  { code: 'JPY', country: 'Japan', currency: 'Yen', rates: { 2025: 149.632, 2024: 151.353, 2023: 140.511, 2022: 131.454, 2021: 109.817 } },
  { code: 'LBP', country: 'Lebanon', currency: 'Pound', rates: { 2025: 89568.540, 2024: 78958.611, 2023: 13730.988, 2022: 1515.669, 2021: 1519.228 } },
  { code: 'MXN', country: 'Mexico', currency: 'Peso', rates: { 2025: 19.212, 2024: 18.330, 2023: 17.733, 2022: 20.110, 2021: 20.284 } },
  { code: 'MAD', country: 'Morocco', currency: 'Dirham', rates: { 2025: 9.344, 2024: 9.937, 2023: 10.134, 2022: 10.275, 2021: 8.995 } },
  { code: 'NZD', country: 'New Zealand', currency: 'Dollar', rates: { 2025: 1.719, 2024: 1.654, 2023: 1.630, 2022: 1.578, 2021: 1.415 } },
  { code: 'NOK', country: 'Norway', currency: 'Kroner', rates: { 2025: 10.392, 2024: 10.756, 2023: 10.564, 2022: 9.619, 2021: 8.598 } },
  { code: 'QAR', country: 'Qatar', currency: 'Rial', rates: { 2025: 3.643, 2024: 3.643, 2023: 3.643, 2022: 3.644, 2021: 3.644 } },
  { code: 'RUB', country: 'Russia', currency: 'Ruble', rates: { 2025: 83.755, 2024: 92.837, 2023: 85.509, 2022: 69.896, 2021: 73.686 } },
  { code: 'SAR', country: 'Saudi Arabia', currency: 'Riyal', rates: { 2025: 3.751, 2024: 3.752, 2023: 3.752, 2022: 3.755, 2021: 3.751 } },
  { code: 'SGD', country: 'Singapore', currency: 'Dollar', rates: { 2025: 1.307, 2024: 1.336, 2023: 1.343, 2022: 1.379, 2021: 1.344 } },
  { code: 'ZAR', country: 'South Africa', currency: 'Rand', rates: { 2025: 17.884, 2024: 18.326, 2023: 18.457, 2022: 16.377, 2021: 14.789 } },
  { code: 'KRW', country: 'South Korean', currency: 'Won', rates: { 2025: 1421.779, 2024: 1364.153, 2023: 1306.686, 2022: 1291.729, 2021: 1144.883 } },
  { code: 'SEK', country: 'Sweden', currency: 'Krona', rates: { 2025: 9.813, 2024: 10.577, 2023: 10.613, 2022: 10.122, 2021: 8.584 } },
  { code: 'CHF', country: 'Switzerland', currency: 'Franc', rates: { 2025: 0.831, 2024: 0.881, 2023: 0.899, 2022: 0.955, 2021: 0.914 } },
  { code: 'TWD', country: 'Taiwan', currency: 'Dollar', rates: { 2025: 31.167, 2024: 32.117, 2023: 31.160, 2022: 29.813, 2021: 27.932 } },
  { code: 'THB', country: 'Thailand', currency: 'Baht', rates: { 2025: 32.870, 2024: 35.267, 2023: 34.802, 2022: 35.044, 2021: 31.997 } },
  { code: 'TND', country: 'Tunisia', currency: 'Dinar', rates: { 2025: 2.996, 2024: 3.111, 2023: 3.103, 2022: 3.082, 2021: 2.778 } },
  { code: 'TRY', country: 'Turkey', currency: 'New Lira', rates: { 2025: 39.546, 2024: 32.867, 2023: 23.824, 2022: 16.572, 2021: 8.904 } },
  { code: 'AED', country: 'United Arab Emirates', currency: 'Dirham', rates: { 2025: 3.673, 2024: 3.673, 2023: 3.673, 2022: 3.673, 2021: 3.673 } },
  { code: 'GBP', country: 'United Kingdom', currency: 'Pound', rates: { 2025: 0.759, 2024: 0.783, 2023: 0.804, 2022: 0.811, 2021: 0.727 } },
  { code: 'VES', country: 'Venezuela', currency: 'Bolivar (Fuerte)', rates: { 2025: 13057596875331350.0, 2024: 3833558362078.0, 2023: 2863377461538.5, 2022: 666470505836.6, 2021: 232298866894.8 } },
];

window.addEventListener('DOMContentLoaded', async () => {
  installAutomaticLock();
  installPasswordVisibilityToggles();
  wireEvents();
  applyTheme(localStorage.getItem('babyfilecab-theme') || 'light', false);
  await waitForBackend();
  wireNativeFileDrop();
  await initialiseAuth();
});

async function waitForBackend() {
  for (let i = 0; i < 100; i++) {
    if (backend()) return;
    await new Promise(r => setTimeout(r, 50));
  }
  throw new Error('BabyFileCab backend did not start.');
}


async function initialiseAuth() {
  hideAllPasswords();
  const hasUsers = await backend().HasUsers();
  currentUser = null;
  $('appShell').classList.add('hidden');
  $('authScreen').classList.remove('hidden');
  $('authPassword').value = '';
  $('authHelp').textContent = hasUsers
    ? 'Enter your BabyFileCab username and password.'
    : 'No local accounts exist yet. Create a new company and its first Administrator account to get started.';
  setTimeout(() => (hasUsers ? $('authUsername') : $('showCreateAccountBtn')).focus(), 40);
}

async function signIn() {
  const username = $('authUsername').value.trim();
  const password = $('authPassword').value;
  if (!username || !password) {
    toast('Enter your username and password.', true);
    (!username ? $('authUsername') : $('authPassword')).focus();
    return;
  }

  const button = $('signInBtn');
  button.disabled = true;
  button.textContent = 'Signing In…';
  try {
    const user = await backend().Login(username, password);
    $('authPassword').value = '';
    await enterApplication(user);
  } catch (e) {
    showError(e);
    $('authPassword').value = '';
    $('authPassword').focus();
  } finally {
    button.disabled = false;
    button.textContent = 'Sign In';
  }
}

async function enterApplication(user) {
  sessionGeneration++;
  hideChangePassword();
  currentUser = user;
  loadShortcutBindings();
  AUTO_LOCK_MS = await backend().GetAutoLockMinutes() * 60 * 1000;
  updateAutoLockMenu();
  lastDesktopInput = Date.now();
  lastActivityPing = 0;
  $('authScreen').classList.add('hidden');
  $('appShell').classList.remove('hidden');
  const fullName = [user?.firstName, user?.lastName].filter(Boolean).join(' ') || user?.username || 'User';
  $('signedInName').textContent = fullName;
  $('signedInRole').textContent = `${displayRole(user?.role)}${user?.companyName ? ` • ${user.companyName}` : ''} • @${user?.username || ''}`;
  $('dataPath').textContent = `Data: ${await backend().DataRoot()}`;
  await refreshTree();
}

async function signOut() {
  sessionGeneration++;
  hideCalendarScheduleClient();
  hideCalendarPreparerReassign();
  hideShortcutMapper();
  try {
    await backend().Logout();
  } catch (e) {
    showError(e);
    return;
  }
  currentUser = null;
  treeData = [];
  selectedNode = null;
  selectedElement = null;
  selectedFilePaths.clear();
  fileSelectionAnchor = null;
  expanded.clear();
  clearPreview();
  $('tree').innerHTML = '';
  $('treeSearch').value = '';
  $('signedInName').textContent = '';
  $('signedInRole').textContent = '';
  await initialiseAuth();
}

function showCreateAccount() {
  hideAllPasswords();
  resetCreateAccountForm();
  $('createAccountBackdrop').classList.remove('hidden');
  setTimeout(() => $('createCompanyName').focus(), 30);
}

function hideCreateAccount() {
  $('createAccountBackdrop').classList.add('hidden');
  $('createPassword').value = '';
  $('createConfirmPassword').value = '';
}

function resetCreateAccountForm() {
  const ids = [
    'createCompanyName', 'createUsername', 'createFirstName', 'createLastName', 'createEmail', 'createPhone',
    'createPassword', 'createConfirmPassword', 'createStreetAddress', 'createCity',
    'createState', 'createZIP', 'createCAF', 'createPTIN', 'createTelephone', 'createFax'
  ];
  for (const id of ids) $(id).value = '';
  $('createRole').value = 'administrator';
}

async function createAccount() {
  const req = {
    companyName: $('createCompanyName').value.trim(),
    username: $('createUsername').value.trim(),
    firstName: $('createFirstName').value.trim(),
    lastName: $('createLastName').value.trim(),
    email: $('createEmail').value.trim(),
    phone: $('createPhone').value.trim(),
    password: $('createPassword').value,
    confirmPassword: $('createConfirmPassword').value,
    role: $('createRole').value,
    streetAddress: $('createStreetAddress').value.trim(),
    city: $('createCity').value.trim(),
    state: $('createState').value.trim(),
    zip: $('createZIP').value.trim(),
    caf: $('createCAF').value.trim(),
    ptin: $('createPTIN').value.trim(),
    telephone: $('createTelephone').value.trim(),
    fax: $('createFax').value.trim(),
  };

  if (!req.companyName) { $('createCompanyName').focus(); toast('Company name is required.', true); return; }
  if (!req.username) { $('createUsername').focus(); toast('Username is required.', true); return; }
  if (!req.firstName) { $('createFirstName').focus(); toast('First name is required.', true); return; }
  if (!req.lastName) { $('createLastName').focus(); toast('Last name is required.', true); return; }
  if (!req.email) { $('createEmail').focus(); toast('Email address is required.', true); return; }
  if (!req.phone) { $('createPhone').focus(); toast('Phone number is required.', true); return; }
  if (req.password.length < 8) { $('createPassword').focus(); toast('Password must be at least 8 characters.', true); return; }
  if (req.password !== req.confirmPassword) { $('createConfirmPassword').focus(); toast('Passwords do not match.', true); return; }
  if (req.role !== 'administrator') { $('createRole').focus(); toast('The first user for a new company must be an Administrator.', true); return; }

  const button = $('createAccountSave');
  button.disabled = true;
  button.textContent = 'Creating Company…';
  try {
    const user = await backend().RegisterUser(req);
    hideCreateAccount();
    $('authUsername').value = user.username || req.username;
    $('authPassword').value = '';
    toast(`${user.companyName || req.companyName} created.`);
    await enterApplication(user);
  } catch (e) {
    showError(e);
  } finally {
    button.disabled = false;
    button.textContent = 'Create Company & Account';
  }
}

async function showExistingCompanyUser() {
  hideAllPasswords();
  let companies;
  try {
    companies = await backend().ListCompanies();
  } catch (e) {
    showError(e);
    return;
  }
  if (!companies || !companies.length) {
    toast('No existing company is available yet. Create a new company account first.', true);
    return;
  }

  resetExistingCompanyUserForm();
  const select = $('existingCompanySelect');
  select.innerHTML = companies.map(c => `<option value="${escapeHTML(c.id)}">${escapeHTML(c.name)}</option>`).join('');
  if (currentUser?.companyId && companies.some(c => c.id === currentUser.companyId)) {
    select.value = currentUser.companyId;
  }
  if (currentUser?.role?.toLowerCase() === 'administrator') {
    $('existingAdminUsername').value = currentUser.username || '';
  }
  $('existingCompanyUserBackdrop').classList.remove('hidden');
  setTimeout(() => $('existingUsername').focus(), 30);
}

function hideExistingCompanyUser() {
  $('existingCompanyUserBackdrop').classList.add('hidden');
  $('existingPassword').value = '';
  $('existingConfirmPassword').value = '';
  $('existingAdminPassword').value = '';
}

function resetExistingCompanyUserForm() {
  const ids = [
    'existingUsername', 'existingFirstName', 'existingLastName', 'existingEmail', 'existingPhone',
    'existingPassword', 'existingConfirmPassword', 'existingStreetAddress', 'existingCity',
    'existingState', 'existingZIP', 'existingCAF', 'existingPTIN', 'existingTelephone', 'existingFax',
    'existingAdminUsername', 'existingAdminPassword'
  ];
  for (const id of ids) $(id).value = '';
  $('existingRole').value = 'staff';
}

async function createExistingCompanyUser() {
  const req = {
    companyId: $('existingCompanySelect').value,
    username: $('existingUsername').value.trim(),
    firstName: $('existingFirstName').value.trim(),
    lastName: $('existingLastName').value.trim(),
    email: $('existingEmail').value.trim(),
    phone: $('existingPhone').value.trim(),
    password: $('existingPassword').value,
    confirmPassword: $('existingConfirmPassword').value,
    role: $('existingRole').value,
    streetAddress: $('existingStreetAddress').value.trim(),
    city: $('existingCity').value.trim(),
    state: $('existingState').value.trim(),
    zip: $('existingZIP').value.trim(),
    caf: $('existingCAF').value.trim(),
    ptin: $('existingPTIN').value.trim(),
    telephone: $('existingTelephone').value.trim(),
    fax: $('existingFax').value.trim(),
    adminUsername: $('existingAdminUsername').value.trim(),
    adminPassword: $('existingAdminPassword').value,
  };

  if (!req.companyId) { $('existingCompanySelect').focus(); toast('Select a company.', true); return; }
  if (!req.username) { $('existingUsername').focus(); toast('Username is required.', true); return; }
  if (!req.firstName) { $('existingFirstName').focus(); toast('First name is required.', true); return; }
  if (!req.lastName) { $('existingLastName').focus(); toast('Last name is required.', true); return; }
  if (!req.email) { $('existingEmail').focus(); toast('Email address is required.', true); return; }
  if (!req.phone) { $('existingPhone').focus(); toast('Phone number is required.', true); return; }
  if (req.password.length < 8) { $('existingPassword').focus(); toast('Password must be at least 8 characters.', true); return; }
  if (req.password !== req.confirmPassword) { $('existingConfirmPassword').focus(); toast('Passwords do not match.', true); return; }
  if (!req.adminUsername) { $('existingAdminUsername').focus(); toast('Administrator username is required.', true); return; }
  if (!req.adminPassword) { $('existingAdminPassword').focus(); toast('Administrator password is required.', true); return; }

  const button = $('existingCompanyUserSave');
  button.disabled = true;
  button.textContent = 'Creating User…';
  try {
    const user = await backend().CreateUserUnderExistingCompany(req);
    hideExistingCompanyUser();
    if (!currentUser) {
      $('authUsername').value = user.username || req.username;
      $('authPassword').value = '';
      $('authUsername').focus();
    }
    toast(`${user.firstName || user.username} was added to ${user.companyName || 'the company'}.`);
  } catch (e) {
    showError(e);
  } finally {
    button.disabled = false;
    button.textContent = 'Create User';
    $('existingAdminPassword').value = '';
  }
}

function displayRole(role) {
  return String(role || '').toLowerCase() === 'administrator' ? 'Administrator' : 'Staff';
}

function showUserManual() {
  $('userManualBackdrop').classList.remove('hidden');
  $('userManualBackdrop').querySelector('.manual-scroll').scrollTop = 0;
  setTimeout(() => $('userManualCloseTop').focus(), 20);
}

function hideUserManual() {
  $('userManualBackdrop').classList.add('hidden');
}

function showAbout() {
  $('aboutBackdrop').classList.remove('hidden');
  setTimeout(() => $('aboutCloseTop')?.focus(), 20);
}

function hideAbout() {
  $('aboutBackdrop').classList.add('hidden');
}


function showAppAuditTrail() {
  $('appAuditSearch').value = '';
  $('appAuditBackdrop').classList.remove('hidden');
  return refreshAppAuditTrail().then(() => setTimeout(() => $('appAuditSearch').focus(), 20));
}

function hideAppAuditTrail() {
  $('appAuditBackdrop').classList.add('hidden');
}

async function refreshAppAuditTrail() {
  appAuditRows = await backend().GetAppAuditTrail();
  renderAppAuditTrail();
}

function renderAppAuditTrail() {
  const q = $('appAuditSearch').value.trim().toLowerCase();
  const rows = (appAuditRows || []).filter(entry => {
    if (!q) return true;
    return [entry.username, entry.userName, entry.clientName, entry.clientId, entry.action, entry.details]
      .some(value => String(value || '').toLowerCase().includes(q));
  });
  const body = $('appAuditBody');
  body.innerHTML = rows.length ? rows.map(entry => {
    const userLabel = entry.userName
      ? `${escapeHTML(entry.userName)}${entry.username ? ` <small>@${escapeHTML(entry.username)}</small>` : ''}`
      : (entry.username ? `@${escapeHTML(entry.username)}` : '<span class="app-audit-muted">Legacy / unknown</span>');
    const clientLabel = entry.clientName
      ? `${escapeHTML(entry.clientName)}${entry.clientId ? ` <small>#${escapeHTML(entry.clientId)}</small>` : ''}`
      : '<span class="app-audit-muted">—</span>';
    return `<tr>
      <td class="app-audit-time">${escapeHTML(formatAuditTime(entry.timestamp))}</td>
      <td>${userLabel}</td>
      <td>${clientLabel}</td>
      <td><strong>${escapeHTML(entry.action || 'Activity')}</strong></td>
      <td>${entry.details ? escapeHTML(entry.details) : '<span class="app-audit-muted">—</span>'}</td>
    </tr>`;
  }).join('') : '<tr><td colspan="5" class="app-audit-empty">No audit entries match this search.</td></tr>';
  $('appAuditStatus').textContent = `${rows.length} of ${(appAuditRows || []).length} audit entr${(appAuditRows || []).length === 1 ? 'y' : 'ies'}`;
}

const ENGAGEMENT_SERVICES = ['1040', '1041', '1065', '1120S', '1120', 'Bookkeeping', 'Tax Projection'];
const ENGAGEMENT_TAX_RETURN_SERVICES = new Set(['1040', '1041', '1065', '1120S', '1120']);
const ENGAGEMENT_SERVICE_DESCRIPTIONS = {
  '1040': 'Form 1040 U.S. Individual Income Tax Return',
  '1041': 'Form 1041 U.S. Income Tax Return for Estates and Trusts',
  '1065': 'Form 1065 U.S. Return of Partnership Income',
  '1120S': 'Form 1120-S U.S. Income Tax Return for an S Corporation',
  '1120': 'Form 1120 U.S. Corporation Income Tax Return',
  'Bookkeeping': 'bookkeeping services',
  'Tax Projection': 'tax projection services',
};

function engagementFormatFee(feeType, raw) {
  let value = String(raw || '').trim().replace(/^\$\s*/, '');
  if (!value) return '';
  value = '$' + value;
  if (feeType === 'Hourly Rate' && !/(\/\s*(?:hr|hour)|per\s+hour)/i.test(value)) value += '/hr';
  return value;
}

function engagementDisplayDate(value) {
  const raw = String(value || '').trim();
  if (!raw) return new Intl.DateTimeFormat('en-US', { month: 'long', day: 'numeric', year: 'numeric' }).format(new Date());
  const d = new Date(raw + 'T12:00:00');
  if (Number.isNaN(d.getTime())) return raw;
  return new Intl.DateTimeFormat('en-US', { month: 'long', day: 'numeric', year: 'numeric' }).format(d);
}

function engagementLetterSections(data) {
  const company = String(data.companyName || '').trim() || 'the Firm';
  const client = String(data.clientName || '').trim() || 'the Client';
  const service = String(data.service || '').trim();
  const year = String(data.taxYear || '').trim();
  const feeType = String(data.feeType || 'Flat Fee').trim();
  const feeValue = engagementFormatFee(feeType, data.feeValue) || 'Enter fee or rate';
  const description = ENGAGEMENT_SERVICE_DESCRIPTIONS[service] || service || 'professional services';

  const intro = `Thank you for engaging ${company}. This Engagement Letter confirms the terms, scope, and limitations of our engagement with ${client} for ${description} for tax year or period ${year}. Please review this letter carefully because it defines the responsibilities of both the Client and the Firm.`;

  let scope;
  if (ENGAGEMENT_TAX_RETURN_SERVICES.has(service)) {
    scope = `${company} will prepare the federal ${description} identified above from information, books, records, schedules, financial statements, and representations provided by ${client}. State, local, amended, informational, payroll, sales tax, foreign reporting, bookkeeping, notice response, audit, tax planning, tax projection, and other services are outside this engagement unless separately agreed to in writing.`;
  } else if (service === 'Bookkeeping') {
    scope = `${company} will provide bookkeeping services for ${client} using bank data, source documents, account information, classifications, instructions, and other records supplied or approved by the Client. The engagement may include recording transactions, reconciliations, and preparation of management reports or financial statements from those records. Unless separately agreed in writing, this engagement does not include an audit, review, compilation, tax return preparation, payroll, sales tax, or assurance service.`;
  } else {
    scope = `${company} will prepare tax projections for ${client} using the income, deduction, withholding, estimated payment, transaction, and other assumptions supplied by the Client. A projection is an estimate based on information available at the time and is not a guarantee of the Client's final tax liability. Actual results may differ because of later transactions, incomplete information, changes in law, or other events.`;
  }

  const sections = [
    ['Scope of Services', scope],
    ['Client Responsibilities', `${client} is responsible for providing complete, accurate, and timely information and for maintaining the books, records, source documents, and supporting evidence required for the services and by applicable tax or regulatory authorities. The Client must review all returns, reports, projections, or other work product before relying on, signing, or filing it.`],
    ['Our Responsibilities', `${company} will perform the agreed services using information and records supplied by ${client}. Unless separately agreed in writing, our services are not an audit, review, examination, agreed-upon procedures engagement, or other assurance service, and we will not independently verify the information provided to us.`],
  ];

  if (ENGAGEMENT_TAX_RETURN_SERVICES.has(service)) {
    sections.push(
      ['Tax Positions and Electronic Filing', `${company} may request clarification or additional documentation when information appears incomplete or inconsistent. The Firm may decline a tax position that it believes lacks adequate support. If the return is eligible for electronic filing, the Firm will not transmit it until the required signed e-file authorization is received.`]
    );
  }

  sections.push(
    ['Deadlines and Client Action', `${client} agrees to provide requested information sufficiently in advance of applicable deadlines. An extension of time to file is not an extension of time to pay. Tax payments, estimated payments, signatures, approvals, and other Client actions remain the Client's responsibility unless separately agreed in writing.`],
    ['Fees and Additional Services', `The agreed fee arrangement for this engagement is ${feeType}: ${feeValue}. This fee applies only to the service identified in this Engagement Letter. Additional work outside the stated scope may require a separate fee or hourly charge agreed to by the Client and the Firm.`],
    ['Records and Confidentiality', `${company} will handle Client information in accordance with applicable professional and legal obligations. The Client should retain original records and supporting documentation.`],
    ['Termination', `Either ${client} or ${company} may terminate this engagement by written notice. The Client remains responsible for fees earned and costs incurred through the termination date and for all filing, payment, recordkeeping, and other deadlines after termination.`]
  );

  return {
    intro,
    sections,
    closing: `If the terms above correctly describe your understanding of this engagement, please sign below. Your signature confirms that ${client} has read, understands, and accepts this Engagement Letter with ${company}.`,
  };
}

function engagementPreparerName(user) {
  return [user?.firstName, user?.lastName].filter(Boolean).join(' ').trim() || user?.username || '';
}

function engagementPreparerAddress(user) {
  if (!user) return [];
  const cityState = [user.city, user.state].filter(Boolean).join(', ');
  const cityLine = cityState + (user.zip ? `${cityState ? ' ' : ''}${user.zip}` : '');
  return [user.streetAddress, cityLine.trim(), user.telephone || user.phone, user.email].filter(Boolean);
}

function engagementSelectedUser() {
  const username = $('engagementPreparer').value;
  return engagementLetterUsers.find(u => String(u.username || '') === username) || null;
}

function engagementClearClientData() {
  const el = $('engagementClient');
  el.dataset.name = '';
  el.dataset.address = '';
  el.dataset.phone = '';
  el.dataset.email = '';
  el.dataset.clientId = '';
  el.dataset.returnType = '';
}

function engagementAddOption(select, value, label) {
  const option = document.createElement('option');
  option.value = value;
  option.textContent = label;
  select.appendChild(option);
}

async function showEngagementLetter() {
  // Always reload the active client list when the generator opens so the
  // Client dropdown reflects the current BabyFileCab database.
  engagementLetterClients = await backend().GetTree();
  engagementLetterClients = (engagementLetterClients || []).filter(node => node && node.type === 'client');
  if (!engagementLetterClients.length) {
    toast('Create a client before generating an Engagement Letter.', true);
    return;
  }

  engagementLetterUsers = await backend().ListCompanyUsers();
  $('engagementCompanyAccount').textContent = currentUser?.companyName || 'BabyFileCab Company';

  const clientSelect = $('engagementClient');
  clientSelect.replaceChildren();
  engagementAddOption(clientSelect, '', 'Select a client…');
  engagementLetterClients.forEach(client => {
    const id = client.clientId ? ` (#${client.clientId})` : '';
    engagementAddOption(clientSelect, client.path, `${client.name}${id}`);
  });
  clientSelect.value = '';
  engagementClearClientData();

  const preparerSelect = $('engagementPreparer');
  preparerSelect.replaceChildren();
  engagementAddOption(preparerSelect, '', 'Select a preparer…');
  engagementLetterUsers.forEach(user => {
    const role = String(user.role || '').toLowerCase().includes('admin') ? 'Administrator' : 'Staff';
    engagementAddOption(preparerSelect, String(user.username || ''), `${engagementPreparerName(user)} — ${role}`);
  });
  preparerSelect.value = '';

  $('engagementService').value = '';
  $('engagementTaxYear').value = '2026';
  $('engagementFeeType').value = 'Flat Fee';
  $('engagementFeeValue').value = '';
  $('engagementDate').value = new Date().toISOString().slice(0, 10);
  $('engagementClientSummary').textContent = `${engagementLetterClients.length} active client${engagementLetterClients.length === 1 ? '' : 's'} available. Choose one from the dropdown.`;
  $('engagementPreparerSummary').textContent = 'Choose the preparer responsible for this engagement.';

  await refreshEngagementFirmLogo();
  updateEngagementFeeUI();
  renderEngagementPreview();
  $('engagementLetterBackdrop').classList.remove('hidden');
  setTimeout(() => clientSelect.focus(), 30);
}

function hideEngagementLetter() {
  $('engagementLetterBackdrop').classList.add('hidden');
}

function updateEngagementLogoUI(info) {
  engagementLogoDataURL = String(info?.dataURL || '');
  const img = $('engagementFirmLogoPreview');
  const placeholder = $('engagementFirmLogoPlaceholder');
  if (engagementLogoDataURL) {
    img.src = engagementLogoDataURL;
    img.classList.remove('hidden');
    placeholder.classList.add('hidden');
    $('engagementRemoveLogo').disabled = false;
  } else {
    img.removeAttribute('src');
    img.classList.add('hidden');
    placeholder.classList.remove('hidden');
    $('engagementRemoveLogo').disabled = true;
  }
  renderEngagementPreview();
}

async function refreshEngagementFirmLogo() {
  const info = await backend().GetEngagementLetterLogo();
  updateEngagementLogoUI(info || {});
}

async function browseEngagementFirmLogo() {
  const info = await backend().ChooseEngagementLetterLogo();
  if (info?.dataURL) {
    updateEngagementLogoUI(info);
    toast('Firm logo added to the Engagement Letter.');
  }
}

async function removeEngagementFirmLogo() {
  await backend().ClearEngagementLetterLogo();
  updateEngagementLogoUI({});
  toast('Firm logo removed from the Engagement Letter.');
}

async function refreshEngagementClient() {
  const path = $('engagementClient').value;
  if (!path) {
    engagementClearClientData();
    $('engagementClientSummary').textContent = `${engagementLetterClients.length} active client${engagementLetterClients.length === 1 ? '' : 's'} available. Choose one from the dropdown.`;
    $('engagementPreviewClient').textContent = 'Select a client';
    return;
  }
  const props = await backend().GetClientProperties(path);
  const el = $('engagementClient');
  el.dataset.name = props.name || '';
  el.dataset.address = props.address || '';
  el.dataset.phone = props.phone || '';
  el.dataset.email = props.email || '';
  el.dataset.clientId = props.clientId || '';
  el.dataset.returnType = props.returnType || '';

  const details = [props.address, props.phone, props.email].filter(Boolean);
  $('engagementClientSummary').textContent = details.length
    ? details.join('  •  ')
    : 'Client selected. No address, phone number, or email is saved in this client profile.';
  $('engagementPreviewClient').textContent = props.name || 'Client';
}

function refreshEngagementPreparer() {
  const user = engagementSelectedUser();
  if (!user) {
    $('engagementPreparerSummary').textContent = 'Choose the preparer responsible for this engagement.';
    return;
  }
  const details = [engagementPreparerName(user), ...engagementPreparerAddress(user)].filter(Boolean);
  $('engagementPreparerSummary').textContent = details.length ? details.join('  •  ') : 'Preparer selected.';
}

function updateEngagementFeeUI() {
  const hourly = $('engagementFeeType').value === 'Hourly Rate';
  $('engagementFeeLabel').textContent = hourly ? 'Hourly Rate (USD $)' : 'Flat Fee (USD $)';
  $('engagementFeeValue').placeholder = hourly ? '250.00' : '1,500.00';
}

function engagementPreviewData() {
  const clientEl = $('engagementClient');
  const user = engagementSelectedUser();
  const service = $('engagementService').value;
  return {
    companyName: currentUser?.companyName || 'CPA Firm',
    logoDataURL: engagementLogoDataURL,
    clientName: clientEl.dataset.name || '',
    clientAddress: clientEl.dataset.address || '',
    clientPhone: clientEl.dataset.phone || '',
    clientEmail: clientEl.dataset.email || '',
    clientId: clientEl.dataset.clientId || '',
    preparerName: engagementPreparerName(user),
    preparerLines: engagementPreparerAddress(user),
    service,
    serviceDescription: ENGAGEMENT_SERVICE_DESCRIPTIONS[service] || '',
    taxYear: $('engagementTaxYear').value,
    feeType: $('engagementFeeType').value,
    feeValue: $('engagementFeeValue').value.trim(),
    date: engagementDisplayDate($('engagementDate').value),
  };
}

function renderEngagementPreview() {
  const data = engagementPreviewData();
  const fee = engagementFormatFee(data.feeType, data.feeValue);
  $('engagementPreviewClient').textContent = data.clientName || 'Select a client';
  $('engagementPreviewService').textContent = data.service || 'Service';

  const logo = data.logoDataURL
    ? `<div class="letter-logo-wrap"><img class="letter-logo" src="${data.logoDataURL}" alt="${escapeHTML(data.companyName)} logo"></div>`
    : '';
  const preparerLines = data.preparerLines.map(line => `<div class="letter-contact">${escapeHTML(line)}</div>`).join('');
  const clientLines = [data.clientAddress, data.clientPhone, data.clientEmail].filter(Boolean).map(line => `<div class="letter-contact">${escapeHTML(line)}</div>`).join('');
  const ready = Boolean(data.clientName && data.preparerName && data.service && data.taxYear && fee);

  let body;
  if (ready) {
    const text = engagementLetterSections(data);
    const sections = text.sections.map(([title, content]) => `<h4>${escapeHTML(title)}</h4><p>${escapeHTML(content)}</p>`).join('');
    body = `
      <p>Dear <strong>${escapeHTML(data.clientName)}</strong>:</p>
      <p>${escapeHTML(text.intro)}</p>
      ${sections}
      <p>${escapeHTML(text.closing)}</p>
      <div class="letter-signature"><strong>Client acceptance:</strong><br><br>${escapeHTML(data.clientName)} signature: ____________________________________<br>Date: ____________________</div>
      <div class="letter-signature"><strong>Firm acknowledgement:</strong><br><br>${escapeHTML(data.companyName)} / ${escapeHTML(data.preparerName)}: ____________________________________<br>Date: ____________________</div>`;
  } else {
    body = `<div class="engagement-preview-empty"><strong>Complete the form above to build the engagement letter.</strong><span>Select a client, preparer, service, tax year, and enter the fee or hourly rate. The preview updates automatically.</span></div>`;
  }

  $('engagementPreview').innerHTML = `
    ${logo}
    <div class="letter-letterhead">
      <div class="letter-firm">${escapeHTML(data.companyName)}</div>
      <div class="letter-preparer-name">${escapeHTML(data.preparerName || 'Preparer')}</div>
      ${preparerLines}
    </div>
    <div class="letter-date">${escapeHTML(data.date)}</div>
    <div class="letter-client">${escapeHTML(data.clientName || 'Client')}</div>
    ${clientLines}
    <h3>Engagement Letter</h3>
    <div class="letter-summary">
      <strong>Service:</strong> ${escapeHTML(data.serviceDescription || 'Select a service')}<br>
      <strong>Tax Year:</strong> ${escapeHTML(data.taxYear || '—')}<br>
      <strong>${escapeHTML(data.feeType || 'Fee')}:</strong> ${escapeHTML(fee || 'Enter an amount')}
    </div>
    ${body}`;
}

async function engagementSelectionChanged({ client = false, preparer = false } = {}) {
  if (client) await refreshEngagementClient();
  if (preparer) refreshEngagementPreparer();
  updateEngagementFeeUI();
  renderEngagementPreview();
}

function buildEngagementRequest() {
  const clientPath = $('engagementClient').value;
  const preparerUsername = $('engagementPreparer').value;
  const service = $('engagementService').value;
  const taxYear = $('engagementTaxYear').value;
  const feeType = $('engagementFeeType').value;
  const feeValue = $('engagementFeeValue').value.trim();
  if (!clientPath) throw new Error('Choose a client from the Client dropdown.');
  if (!preparerUsername) throw new Error('Choose a preparer from the Preparer dropdown.');
  if (!ENGAGEMENT_SERVICES.includes(service)) throw new Error('Choose a service from the Service dropdown.');
  if (!taxYear) throw new Error('Choose a tax year from 2018 through 2026.');
  if (!feeValue) throw new Error(`Enter the ${feeType === 'Hourly Rate' ? 'hourly rate' : 'flat fee'} in U.S. dollars.`);
  return {
    clientPath,
    preparerUsername,
    service,
    taxYear,
    feeType,
    feeValue,
    letterDate: $('engagementDate').value,
    outputPath: '',
  };
}

async function generateEngagementLetter(kind) {
  const buttons = [$('engagementGeneratePDF'), $('engagementGenerateWord')];
  try {
    const req = buildEngagementRequest();
    const chosen = await backend().ChooseEngagementLetterSavePath(kind, req.clientPath, req.service, req.taxYear);
    if (!chosen) return;
    req.outputPath = chosen;
    buttons.forEach(button => button.disabled = true);
    const paths = await backend().GenerateEngagementLetter(req, kind);
    const list = Array.isArray(paths) ? paths.join(' • ') : '';
    toast(`Engagement Letter created${list ? `: ${list}` : '.'}`);
  } catch (e) {
    showError(e);
  } finally {
    buttons.forEach(button => button.disabled = false);
  }
}


function invoiceSelectedUser() {
  const username = $('invoicePreparer').value;
  return invoiceUsers.find(u => String(u.username || '') === username) || null;
}

function invoiceClearClientData() {
  const el = $('invoiceClient');
  el.dataset.name = '';
  el.dataset.address = '';
  el.dataset.phone = '';
  el.dataset.email = '';
  el.dataset.clientId = '';
}

function invoiceMoneyNumber(raw) {
  const cleaned = String(raw || '').replace(/[$,\s]/g, '');
  const value = Number(cleaned);
  return Number.isFinite(value) && value >= 0 ? value : 0;
}

function invoiceMoney(value) {
  const n = Number(value || 0);
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(Number.isFinite(n) ? n : 0);
}

function invoiceDateDisplay(value) {
  const raw = String(value || '').trim();
  if (!raw) return '—';
  const d = new Date(raw + 'T12:00:00');
  if (Number.isNaN(d.getTime())) return raw;
  return new Intl.DateTimeFormat('en-US', { month: 'long', day: 'numeric', year: 'numeric' }).format(d);
}

function invoiceDefaultNumber() {
  const d = new Date();
  const parts = [d.getFullYear(), String(d.getMonth() + 1).padStart(2, '0'), String(d.getDate()).padStart(2, '0')];
  const time = [String(d.getHours()).padStart(2, '0'), String(d.getMinutes()).padStart(2, '0')].join('');
  return `INV-${parts.join('')}-${time}`;
}

function invoiceAddDaysISO(iso, days) {
  const d = new Date(String(iso || '') + 'T12:00:00');
  if (Number.isNaN(d.getTime())) return '';
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}



async function showInvoiceGenerator() {
  invoiceClients = await backend().GetTree();
  invoiceClients = (invoiceClients || []).filter(node => node && node.type === 'client');
  if (!invoiceClients.length) {
    toast('Create a client before generating an invoice.', true);
    return;
  }
  invoiceUsers = await backend().ListCompanyUsers();
  $('invoiceCompanyAccount').textContent = currentUser?.companyName || 'BabyFileCab Company';

  const clientSelect = $('invoiceClient');
  clientSelect.replaceChildren();
  engagementAddOption(clientSelect, '', 'Select a client…');
  invoiceClients.forEach(client => {
    const id = client.clientId ? ` (#${client.clientId})` : '';
    engagementAddOption(clientSelect, client.path, `${client.name}${id}`);
  });
  clientSelect.value = '';
  invoiceClearClientData();
  $('invoiceClientSummary').textContent = `${invoiceClients.length} active client${invoiceClients.length === 1 ? '' : 's'} available. Choose one from the dropdown.`;

  const preparerSelect = $('invoicePreparer');
  preparerSelect.replaceChildren();
  engagementAddOption(preparerSelect, '', 'Select a preparer…');
  invoiceUsers.forEach(user => {
    const role = String(user.role || '').toLowerCase().includes('admin') ? 'Administrator' : 'Staff';
    engagementAddOption(preparerSelect, String(user.username || ''), `${engagementPreparerName(user)} — ${role}`);
  });
  preparerSelect.value = '';
  $('invoicePreparerSummary').textContent = 'Choose the preparer responsible for this invoice.';

  const today = new Date().toISOString().slice(0, 10);
  $('invoiceService').value = '';
  $('invoiceTaxYear').value = '2026';
  $('invoiceNumber').value = invoiceDefaultNumber();
  $('invoiceDate').value = today;
  $('invoiceDueDate').value = invoiceAddDaysISO(today, 30);
  $('invoiceFeeType').value = 'Flat Fee';
  $('invoiceHours').value = '1.00';
  $('invoiceFeeValue').value = '';
  $('invoiceDescription').value = '';
  $('invoiceNotes').value = 'Thank you for your business.';
  await refreshInvoiceLogo();
  updateInvoiceFeeUI();
  renderInvoicePreview();
  $('invoiceBackdrop').classList.remove('hidden');
  const invoiceScroll = $('invoiceScrollContent');
  if (invoiceScroll) invoiceScroll.scrollTop = 0;
  setTimeout(() => { clientSelect.focus(); syncInvoiceVisibleScrollbar(); }, 30);
}

function hideInvoiceGenerator() {
  $('invoiceBackdrop').classList.add('hidden');
}

function updateInvoiceLogoUI(info) {
  invoiceLogoDataURL = String(info?.dataURL || '');
  const img = $('invoiceFirmLogoPreview');
  const placeholder = $('invoiceFirmLogoPlaceholder');
  if (invoiceLogoDataURL) {
    img.src = invoiceLogoDataURL;
    img.classList.remove('hidden');
    placeholder.classList.add('hidden');
    $('invoiceRemoveLogo').disabled = false;
  } else {
    img.removeAttribute('src');
    img.classList.add('hidden');
    placeholder.classList.remove('hidden');
    $('invoiceRemoveLogo').disabled = true;
  }
  renderInvoicePreview();
}

async function refreshInvoiceLogo() {
  const info = await backend().GetEngagementLetterLogo();
  updateInvoiceLogoUI(info || {});
}

async function browseInvoiceLogo() {
  const info = await backend().ChooseEngagementLetterLogo();
  if (info?.dataURL) {
    updateInvoiceLogoUI(info);
    toast('Firm logo added to the invoice.');
  }
}

async function removeInvoiceLogo() {
  await backend().ClearEngagementLetterLogo();
  updateInvoiceLogoUI({});
  toast('Firm logo removed.');
}

async function refreshInvoiceClient() {
  const path = $('invoiceClient').value;
  if (!path) {
    invoiceClearClientData();
    $('invoiceClientSummary').textContent = `${invoiceClients.length} active client${invoiceClients.length === 1 ? '' : 's'} available. Choose one from the dropdown.`;
    $('invoicePreviewClient').textContent = 'Select a client';
    return;
  }
  const props = await backend().GetClientProperties(path);
  const el = $('invoiceClient');
  el.dataset.name = props.name || '';
  el.dataset.address = props.address || '';
  el.dataset.phone = props.phone || '';
  el.dataset.email = props.email || '';
  el.dataset.clientId = props.clientId || '';
  const details = [props.address, props.phone, props.email].filter(Boolean);
  $('invoiceClientSummary').textContent = details.length ? details.join('  •  ') : 'Client selected. No address, phone number, or email is saved in this client profile.';
  $('invoicePreviewClient').textContent = props.name || 'Client';
}

function refreshInvoicePreparer() {
  const user = invoiceSelectedUser();
  if (!user) {
    $('invoicePreparerSummary').textContent = 'Choose the preparer responsible for this invoice.';
    return;
  }
  const details = [engagementPreparerName(user), ...engagementPreparerAddress(user)].filter(Boolean);
  $('invoicePreparerSummary').textContent = details.length ? details.join('  •  ') : 'Preparer selected.';
}

function updateInvoiceFeeUI() {
  const hourly = $('invoiceFeeType').value === 'Hourly Rate';
  $('invoiceHoursRow').classList.toggle('hidden', !hourly);
  $('invoiceFeeLabel').textContent = hourly ? 'Hourly Rate (USD $)' : 'Flat Fee (USD $)';
  $('invoiceFeeValue').placeholder = hourly ? '250.00' : '1,500.00';
  renderInvoicePreview();
}

function invoicePreviewData() {
  const clientEl = $('invoiceClient');
  const user = invoiceSelectedUser();
  const service = $('invoiceService').value;
  const feeType = $('invoiceFeeType').value;
  const fee = invoiceMoneyNumber($('invoiceFeeValue').value);
  const hours = feeType === 'Hourly Rate' ? Math.max(0, Number($('invoiceHours').value || 0)) : 1;
  const total = feeType === 'Hourly Rate' ? fee * hours : fee;
  const customDesc = $('invoiceDescription').value.trim();
  const serviceDescription = ENGAGEMENT_SERVICE_DESCRIPTIONS[service] || service || '';
  return {
    companyName: currentUser?.companyName || 'CPA Firm',
    logoDataURL: invoiceLogoDataURL,
    clientName: clientEl.dataset.name || '',
    clientAddress: clientEl.dataset.address || '',
    clientPhone: clientEl.dataset.phone || '',
    clientEmail: clientEl.dataset.email || '',
    clientId: clientEl.dataset.clientId || '',
    preparerName: engagementPreparerName(user),
    preparerLines: engagementPreparerAddress(user),
    service,
    serviceDescription,
    taxYear: $('invoiceTaxYear').value,
    invoiceNumber: $('invoiceNumber').value.trim(),
    invoiceDate: $('invoiceDate').value,
    dueDate: $('invoiceDueDate').value,
    feeType,
    fee,
    hours,
    total,
    description: customDesc || (serviceDescription ? `${serviceDescription} — Tax Year ${$('invoiceTaxYear').value}` : ''),
    notes: $('invoiceNotes').value.trim(),
  };
}

function renderInvoicePreview() {
  const data = invoicePreviewData();
  $('invoicePreviewClient').textContent = data.clientName || 'Select a client';
  $('invoicePreviewService').textContent = data.service || 'Service';
  const logo = data.logoDataURL ? `<div class="letter-logo-wrap"><img class="letter-logo" src="${data.logoDataURL}" alt="${escapeHTML(data.companyName)} logo"></div>` : '';
  const preparerLines = data.preparerLines.map(line => `<div class="letter-contact">${escapeHTML(line)}</div>`).join('');
  const clientLines = [data.clientAddress, data.clientPhone, data.clientEmail].filter(Boolean).map(line => `<div class="letter-contact">${escapeHTML(line)}</div>`).join('');
  const hourly = data.feeType === 'Hourly Rate';
  const quantity = hourly ? data.hours.toFixed(2) : '1';
  const rate = invoiceMoney(data.fee);
  const ready = Boolean(data.clientName && data.preparerName && data.service && data.invoiceNumber && data.invoiceDate && data.dueDate && data.fee > 0 && (!hourly || data.hours > 0));
  $('invoicePreview').innerHTML = `
    <div class="invoice-preview-top">
      <div>${logo}<div class="letter-firm">${escapeHTML(data.companyName)}</div><div class="letter-preparer-name">${escapeHTML(data.preparerName || 'Preparer')}</div>${preparerLines}</div>
      <div class="invoice-preview-title"><strong>INVOICE</strong><span># ${escapeHTML(data.invoiceNumber || '—')}</span></div>
    </div>
    <div class="invoice-meta-grid"><div><span>Invoice Date</span><strong>${escapeHTML(invoiceDateDisplay(data.invoiceDate))}</strong></div><div><span>Due Date</span><strong>${escapeHTML(invoiceDateDisplay(data.dueDate))}</strong></div><div><span>Tax Year</span><strong>${escapeHTML(data.taxYear || '—')}</strong></div></div>
    <div class="invoice-billto"><span>BILL TO</span><strong>${escapeHTML(data.clientName || 'Select a client')}</strong>${clientLines}</div>
    <div class="invoice-line-head"><span>Description</span><span>Qty</span><span>Rate</span><span>Amount</span></div>
    <div class="invoice-line-row"><span>${escapeHTML(data.description || 'Select a service')}</span><span>${escapeHTML(quantity)}</span><span>${escapeHTML(rate)}</span><strong>${escapeHTML(invoiceMoney(data.total))}</strong></div>
    <div class="invoice-total"><span>Total Due</span><strong>${escapeHTML(invoiceMoney(data.total))}</strong></div>
    ${data.notes ? `<div class="invoice-note"><strong>Note</strong><p>${escapeHTML(data.notes)}</p></div>` : ''}
    ${ready ? '' : '<div class="engagement-preview-empty"><strong>Complete the invoice form above.</strong><span>Select a client, preparer, service, dates, and enter the fee information before generating the invoice.</span></div>'}`;
}

async function invoiceSelectionChanged({ client = false, preparer = false } = {}) {
  if (client) await refreshInvoiceClient();
  if (preparer) refreshInvoicePreparer();
  updateInvoiceFeeUI();
  renderInvoicePreview();
}

function buildInvoiceRequest() {
  const data = invoicePreviewData();
  const clientPath = $('invoiceClient').value;
  const preparerUsername = $('invoicePreparer').value;
  if (!clientPath) throw new Error('Choose a client from the Client dropdown.');
  if (!preparerUsername) throw new Error('Choose a preparer from the Preparer dropdown.');
  if (!ENGAGEMENT_SERVICES.includes(data.service)) throw new Error('Choose a service from the Service dropdown.');
  if (!data.invoiceNumber) throw new Error('Enter an invoice number.');
  if (!data.invoiceDate) throw new Error('Choose an invoice date.');
  if (!data.dueDate) throw new Error('Choose a due date.');
  if (data.dueDate < data.invoiceDate) throw new Error('The due date cannot be earlier than the invoice date.');
  if (data.fee <= 0) throw new Error(`Enter the ${data.feeType === 'Hourly Rate' ? 'hourly rate' : 'flat fee'} in U.S. dollars.`);
  if (data.feeType === 'Hourly Rate' && data.hours <= 0) throw new Error('Enter the number of hours for the hourly invoice.');
  return {
    clientPath,
    preparerUsername,
    service: data.service,
    taxYear: data.taxYear,
    invoiceNumber: data.invoiceNumber,
    invoiceDate: data.invoiceDate,
    dueDate: data.dueDate,
    feeType: data.feeType,
    feeValue: $('invoiceFeeValue').value.trim(),
    hours: String(data.hours),
    description: $('invoiceDescription').value.trim(),
    notes: $('invoiceNotes').value.trim(),
    outputPath: '',
  };
}

async function generateInvoice(kind) {
  const buttons = [$('invoiceGeneratePDF'), $('invoiceGenerateWord')];
  try {
    const req = buildInvoiceRequest();
    const chosen = await backend().ChooseInvoiceSavePath(kind, req.clientPath, req.invoiceNumber);
    if (!chosen) return;
    req.outputPath = chosen;
    buttons.forEach(button => button.disabled = true);
    const paths = await backend().GenerateInvoice(req, kind);
    const list = Array.isArray(paths) ? paths.join(' • ') : '';
    toast(`Invoice created${list ? `: ${list}` : '.'}`);
  } catch (e) {
    showError(e);
  } finally {
    buttons.forEach(button => button.disabled = false);
  }
}

function selectedTaxYear() {
  const year = Number($('taxYear')?.value || 2025);
  return TAX_YEARS[year] ? year : 2025;
}

function selectedTaxConfig() {
  return TAX_YEARS[selectedTaxYear()];
}

function updateTaxCalculatorYearUI() {
  const year = selectedTaxYear();
  const config = TAX_YEARS[year];
  const sourceTitle = year === 2026
    ? 'References used for this 2026 calculator'
    : 'IRS references used for this 2025 calculator';
  const sourceText = year === 2026
    ? 'Tax Foundation 2026 Tax Brackets • IRS Revenue Procedure 2025-32 • IRS Child Tax Credit guidance • IRS 2026 Form 1040-ES payment dates • IRS capital-gain thresholds • Social Security Administration 2026 contribution and benefit base.'
    : 'Federal income tax rates and brackets • Publication 17 (2025) • IRS Schedule 8812 / Child Tax Credit guidance • IRS 2025 Form 1040-ES payment dates • Topic No. 409, Capital Gains and Losses • 2025 Schedule SE guidance.';

  $('taxCalculatorEyebrow').textContent = `TOOLS • TAX YEAR ${year}`;
  $('taxCalculatorTitle').textContent = `${year} Federal Tax Calculator`;
  $('taxCalculatorDescription').textContent = `Estimate ${year} federal income tax using the standard deduction, ${year} tax brackets, capital-gain rates, and self-employment tax rules.`;
  $('taxIncomeHeading').textContent = `${year} income`;
  $('taxCalculatorCalculate').textContent = `Calculate ${year} Estimate`;
  $('taxResultYearLabel').textContent = `Estimated ${year} federal tax`;
  $('taxSourcesTitle').textContent = sourceTitle;
  $('taxSourcesText').textContent = sourceText;
  $('taxSourcesNote').textContent = `Standard deduction: ${taxCurrency(config.standardDeduction.single)} Single/MFS; ${taxCurrency(config.standardDeduction.mfj)} MFJ/QSS; ${taxCurrency(config.standardDeduction.hoh)} Head of Household. Child Tax Credit: up to ${taxCurrency(config.childTaxCredit.perChild)} per qualifying child, with up to ${taxCurrency(config.childTaxCredit.refundablePerChild)} potentially refundable through the ACTC. Estimated-tax due dates: ${config.estimatedTaxDueDates.join(' • ')}. Social Security wage base used for SE tax: ${taxCurrency(config.socialSecurityWageBase)}.`;
  $('taxCalculatorResults').classList.add('hidden');
}

function showTaxCalculator() {
  updateTaxCalculatorYearUI();
  $('taxCalculatorBackdrop').classList.remove('hidden');
  setTimeout(() => $('taxYear')?.focus(), 20);
}

function hideTaxCalculator() {
  $('taxCalculatorBackdrop').classList.add('hidden');
}


function firmCalendarUTCDate(year, monthIndex, day) {
  return new Date(Date.UTC(year, monthIndex, day));
}

function firmCalendarISO(date) {
  return date.toISOString().slice(0, 10);
}

function firmCalendarDateFromISO(value) {
  const [year, month, day] = String(value || '').split('-').map(Number);
  return firmCalendarUTCDate(year, month - 1, day);
}

function firmCalendarFormatDate(date, options = {}) {
  return new Intl.DateTimeFormat('en-US', { timeZone: 'UTC', ...options }).format(date);
}

function buildFirmCalendarWeekStarts() {
  const starts = [];
  const start = firmCalendarUTCDate(2025, 11, 29); // Monday of the week containing Jan. 1, 2026.
  const last = firmCalendarUTCDate(2026, 11, 28);
  for (let cursor = new Date(start); cursor <= last; cursor.setUTCDate(cursor.getUTCDate() + 7)) {
    starts.push(new Date(cursor));
  }
  return starts;
}

function firmCalendarWeekEnd(start) {
  const end = new Date(start);
  end.setUTCDate(end.getUTCDate() + 6);
  return end;
}

function firmCalendarWeekLabel(start, index) {
  const end = firmCalendarWeekEnd(start);
  const startText = firmCalendarFormatDate(start, { month: 'short', day: 'numeric', year: start.getUTCFullYear() === 2026 ? undefined : 'numeric' });
  const endText = firmCalendarFormatDate(end, { month: 'short', day: 'numeric', year: 'numeric' });
  return `Week ${index + 1} · ${startText} – ${endText}`;
}

function populateFirmCalendarWeekSelect() {
  const select = $('firmCalendarWeekSelect');
  select.innerHTML = firmCalendarWeekStarts.map((start, index) =>
    `<option value="${index}">${escapeHTML(firmCalendarWeekLabel(start, index))}</option>`
  ).join('');
  select.value = String(firmCalendarWeekIndex);
}

function firmCalendarCurrentWeekIndex() {
  const now = new Date();
  const today = firmCalendarUTCDate(now.getFullYear(), now.getMonth(), now.getDate());
  if (today.getUTCFullYear() !== 2026) return 0;
  for (let i = 0; i < firmCalendarWeekStarts.length; i++) {
    const start = firmCalendarWeekStarts[i];
    const end = firmCalendarWeekEnd(start);
    if (today >= start && today <= end) return i;
  }
  return 0;
}

let firmCalendarPreparers = [];
let firmCalendarPreparerView = '*';
let firmCalendarPreparerReassignItem = null;
function calendarPreparerName(username) {
  if (!username) return 'Unassigned';
  const user = firmCalendarPreparers.find(user => user.username.toLowerCase() === username.toLowerCase());
  return user ? [user.firstName,user.lastName].filter(Boolean).join(' ') || user.username : username;
}
function calendarPreparerInitials(username) {
  const name = calendarPreparerName(username);
  return name.split(/\s+/).filter(Boolean).slice(0,2).map(part => part[0]).join('').toUpperCase();
}
function calendarAssignmentsForPreparer(items, view) {
  return view === '*' ? [...items] : items.filter(item => (item.preparerUsername || '').toLowerCase() === view.toLowerCase());
}
function renderCalendarPreparerMenu() {
  const menu = $('firmCalendarPreparerMenu'); menu.replaceChildren();
  const choices = [{username:'*',label:'Entire Firm — All Preparers'},{username:'',label:'Unassigned Clients'}, ...firmCalendarPreparers.map(user => ({username:user.username,label:calendarPreparerName(user.username)}))];
  for (const choice of choices) {
    const button = document.createElement('button'); button.type='button'; button.setAttribute('role','menuitemradio');
    button.setAttribute('aria-checked',String(choice.username===firmCalendarPreparerView)); button.textContent = choice.label;
    button.addEventListener('click', () => {
      firmCalendarPreparerView=choice.username; menu.classList.add('hidden'); clearFirmCalendarClientDetail(); renderCalendarPreparerMenu(); renderFirmCalendarWeek();
    }); menu.appendChild(button);
  }
  const view = firmCalendarPreparerView;
  $('firmCalendarPreparerButton').textContent = view === '*' ? `${calendarPreparerInitials(currentUser.username)} • Entire Firm ▾` : view === '' ? 'Unassigned ▾' : `${calendarPreparerInitials(view)} • ${calendarPreparerName(view)} ▾`;
  $('firmCalendarPreparerButton').setAttribute('aria-expanded','false');
}
function hideCalendarPreparerReassign() { $('firmCalendarPreparerReassignBackdrop').classList.add('hidden'); firmCalendarPreparerReassignItem=null; }
function showCalendarPreparerReassign(item) {
  firmCalendarPreparerReassignItem={...item};
  $('firmCalendarPreparerReassignClient').textContent=`${item.clientName} — ${item.date}`;
  const select=$('firmCalendarPreparerReassignSelect'); select.replaceChildren();
  for (const user of [{username:''},...firmCalendarPreparers]) {
    const option=document.createElement('option'); option.value=user.username; option.textContent=calendarPreparerName(user.username); select.appendChild(option);
  }
  select.value=item.preparerUsername || '';
  $('firmCalendarPreparerReassignBackdrop').classList.remove('hidden'); select.focus();
}
async function saveCalendarPreparerReassign() {
  if (!firmCalendarPreparerReassignItem || $('firmCalendarPreparerReassignSave').disabled) return;
  const item={...firmCalendarPreparerReassignItem},generation=sessionGeneration;
  $('firmCalendarPreparerReassignSave').disabled=true;
  try {
    const assignments=await backend().ReassignFirmCalendarPreparer2026(item.date,item.clientId,$('firmCalendarPreparerReassignSelect').value);
    if (generation!==sessionGeneration) return;
    firmCalendarAssignments=assignments; hideCalendarPreparerReassign(); clearFirmCalendarClientDetail(); renderFirmCalendarWeek(); toast('Preparer assignment saved.');
  } catch(error) { if(generation===sessionGeneration) showError(error); }
  finally { $('firmCalendarPreparerReassignSave').disabled=false; }
}

async function showFirmCalendar() {
  if (!currentUser) return;
  await refreshTree();
  firmCalendarWeekStarts = buildFirmCalendarWeekStarts();
  firmCalendarWeekIndex = firmCalendarCurrentWeekIndex();
  firmCalendarSelectedClientID = '';
  $('firmCalendarSearch').value = '';
  firmCalendarPreparers = await backend().ListCompanyUsers();
  firmCalendarPreparerView = '*';
  renderCalendarPreparerMenu();
  firmCalendarAssignments = await backend().GetFirmCalendar2026();
  populateFirmCalendarWeekSelect();
  clearFirmCalendarClientDetail();
  renderFirmCalendarWeek();
  $('firmCalendarBackdrop').classList.remove('hidden');
  setTimeout(() => $('firmCalendarWeekSelect')?.focus(), 20);
}

function hideFirmCalendar() {
  hideCalendarScheduleClient();
  hideCalendarPreparerReassign();
  $('firmCalendarPreparerMenu').classList.add('hidden');
  $('firmCalendarBackdrop').classList.add('hidden');
}

function clearFirmCalendarClientDetail() {
  firmCalendarSelectedClientID = '';
  $('firmCalendarClientDetail').classList.add('hidden');
  $('firmCalendarClientEmpty').classList.remove('hidden');
  $('firmCalendarClientEmpty').innerHTML = '<div class="empty-icon">🗂️</div><h3>Client Documents</h3><p>Click a client name on the calendar to see that client\'s Permanent Folder and documents organized by tax year.</p>';
  $('firmCalendarClientName').textContent = '';
  $('firmCalendarClientMeta').textContent = '';
  $('firmCalendarClientDocuments').innerHTML = '';
}

function firmCalendarStatusLabel(status) {
  const labels={'drafting':'Drafting','waiting-on-reports':'Waiting on reports','completed':'Completed','8879-sent':'8879 Sent','waiting-for-signature':'Waiting for Signature','signed':'Signed','ready-to-e-file':'Ready to E-file'};
  return Object.prototype.hasOwnProperty.call(labels,status) ? labels[status] : '';
}
function firmCalendarStatusBadge(status) {
  if (status === 'completed') return '<span class="firm-calendar-completed" title="Completed" aria-label="Completed">✓</span>';
  const label=firmCalendarStatusLabel(status);
  return label ? `<span class="firm-calendar-work-status calendar-status-${status}">${label}</span>` : '';
}
function searchFirmCalendarAssignments(items, query) {
  const term = String(query || '').trim().toLocaleLowerCase();
  if (!term) return [];
  return items.filter(item => String(item.clientName || '').toLocaleLowerCase().includes(term))
    .sort((a,b) => a.date.localeCompare(b.date) || String(a.clientName).localeCompare(String(b.clientName)));
}
let calendarScheduleClient = null;
function calendarSearchAllClients(clients, query) {
  const term=String(query || '').trim().toLocaleLowerCase();
  return term ? clients.filter(client => client.type === 'client' && (String(client.name || '').toLocaleLowerCase().includes(term) || String(client.clientId || '').toLocaleLowerCase().includes(term))).sort((a,b) => a.name.localeCompare(b.name)) : [];
}
function hideCalendarScheduleClient() { $('calendarScheduleClientBackdrop').classList.add('hidden'); calendarScheduleClient=null; }
function showCalendarScheduleClient(clientID) {
  const client=treeData.find(client => client.type === 'client' && client.clientId === clientID);
  if (!client) { toast('Client no longer available. Refresh the calendar.',true); return; }
  calendarScheduleClient={...client};
  $('calendarScheduleClientName').textContent=client.name;
  const start=firmCalendarWeekStarts[firmCalendarWeekIndex];
  $('calendarScheduleClientDate').value=start && start.getUTCFullYear()===2026 ? firmCalendarISO(start) : '2026-01-01';
  const select=$('calendarScheduleClientPreparer'); select.replaceChildren();
  for (const user of [{username:''},...firmCalendarPreparers]) {
    const option=document.createElement('option'); option.value=user.username; option.textContent=calendarPreparerName(user.username); select.appendChild(option);
  }
  select.value=firmCalendarPreparerView === '*' ? currentUser.username : firmCalendarPreparerView;
  $('calendarScheduleClientBackdrop').classList.remove('hidden'); $('calendarScheduleClientDate').focus();
}
async function saveCalendarScheduleClient() {
  if (!calendarScheduleClient || $('calendarScheduleClientSave').disabled) return;
  const client={...calendarScheduleClient}, date=$('calendarScheduleClientDate').value, username=$('calendarScheduleClientPreparer').value;
  if (!/^2026-\d{2}-\d{2}$/.test(date) || !$('calendarScheduleClientDate').checkValidity()) { toast('Choose a valid date in 2026.',true); return; }
  const existing=firmCalendarAssignments.find(item => item.date===date && item.clientId===client.clientId);
  if (existing && !confirm('This client is already scheduled on that date. Update its assigned preparer?')) return;
  const generation=sessionGeneration; $('calendarScheduleClientSave').disabled=true;
  try {
    const assignments=existing ? await backend().ReassignFirmCalendarPreparer2026(date,client.clientId,username) : await backend().AssignClientToFirmCalendarForPreparer2026(date,client.path,username);
    if (generation!==sessionGeneration) return;
    firmCalendarAssignments=assignments; firmCalendarPreparerView=username;
    const selectedDate=firmCalendarDateFromISO(date);
    const index=firmCalendarWeekStarts.findIndex(start => selectedDate>=start && selectedDate<=firmCalendarWeekEnd(start));
    if(index>=0)firmCalendarWeekIndex=index;
    hideCalendarScheduleClient(); renderCalendarPreparerMenu(); renderFirmCalendarWeek(); toast('Client assigned to calendar.');
  } catch(error) {if(generation===sessionGeneration)showError(error);}
  finally { $('calendarScheduleClientSave').disabled=false; }
}

function goToClientOnCalendar(clientID, dateISO) {
  const item=firmCalendarAssignments.find(item => item.clientId===clientID && item.date===dateISO);
  if(!item){toast('This calendar assignment is no longer available.',true);return;}
  const date=firmCalendarDateFromISO(item.date);
  const index=firmCalendarWeekStarts.findIndex(start => date>=start && date<=firmCalendarWeekEnd(start));
  if(index<0)return;
  firmCalendarWeekIndex=index;
  if(firmCalendarPreparerView!=='*')firmCalendarPreparerView=item.preparerUsername || '';
  $('firmCalendarSearch').value='';
  renderCalendarPreparerMenu(); renderFirmCalendarWeek(); showFirmCalendarClientDocuments(clientID);
  const chip=[...$('firmCalendarGrid').querySelectorAll('[data-calendar-assignment-client]')].find(chip => chip.dataset.calendarAssignmentClient===clientID && chip.dataset.calendarAssignmentDate===dateISO);
  chip?.scrollIntoView({block:'nearest',inline:'nearest'});
  chip?.querySelector('button')?.focus({preventScroll:true});
}

let calendarSearchSelectedClient = '';
function calendarSearchSelectedAssignment() {
  const date=$('calendarSearchDateChoice')?.value;
  return firmCalendarAssignments.find(item => item.clientId===calendarSearchSelectedClient && item.date===date);
}
function renderFirmCalendarSearch() {
  const query=$('firmCalendarSearch').value.trim(), results=$('firmCalendarSearchResults');
  results.classList.toggle('hidden',!query);
  if(!query){results.replaceChildren();calendarSearchSelectedClient='';return;}
  const clients=calendarSearchAllClients(treeData,query);
  if(!clients.some(client=>client.clientId===calendarSearchSelectedClient))calendarSearchSelectedClient=clients[0]?.clientId || '';
  const selected=clients.find(client=>client.clientId===calendarSearchSelectedClient);
  const entries=firmCalendarAssignments.filter(item=>item.clientId===calendarSearchSelectedClient).sort((a,b)=>a.date.localeCompare(b.date));
  results.innerHTML=`<div class="calendar-search-summary">${clients.length} matching client${clients.length===1?'':'s'} · All firm clients</div><div class="calendar-search-workspace"><div class="calendar-search-list">${clients.map(client=>{
    const count=firmCalendarAssignments.filter(item=>item.clientId===client.clientId).length;
    return `<button type="button" class="calendar-search-client ${client.clientId===calendarSearchSelectedClient?'is-selected':''}" data-calendar-select-client="${escapeHTML(client.clientId)}" aria-pressed="${client.clientId===calendarSearchSelectedClient}"><strong>${escapeHTML(client.name)}</strong><small>${count ? `${count} scheduled date${count===1?'':'s'}`:'Not scheduled'}</small></button>`;
  }).join('') || '<p>No clients found. Try a different name.</p>'}</div><div class="calendar-search-detail">${selected ? `<h3>${escapeHTML(selected.name)}</h3><p>Select a scheduled entry to open it or make changes.</p>${entries.length ? `<label for="calendarSearchDateChoice">Scheduled date · preparer · status</label><select id="calendarSearchDateChoice">${entries.map(item=>`<option value="${escapeHTML(item.date)}">${escapeHTML(item.date)} · ${escapeHTML(calendarPreparerName(item.preparerUsername))} · ${escapeHTML(firmCalendarStatusLabel(item.status) || 'No status')}</option>`).join('')}</select><div class="calendar-search-actions"><button type="button" class="primary" data-calendar-search-command="go">Go to Calendar</button><select id="calendarSearchEditChoice" aria-label="Change selected assignment"><option value="">Change assignment…</option><option value="preparer">Change preparer</option><option value="date">Move to another date</option><optgroup label="Set status"><option value="status-drafting">Drafting</option><option value="status-waiting-on-reports">Waiting on reports</option><option value="status-8879-sent">8879 Sent</option><option value="status-waiting-for-signature">Waiting for Signature</option><option value="status-signed">Signed</option><option value="status-ready-to-e-file">Ready to E-file</option><option value="status-completed">Completed</option><option value="clear">Clear status</option></optgroup></select></div>` : '<p class="calendar-search-empty">This client has no scheduled dates yet.</p>'}<button type="button" class="secondary" data-calendar-schedule-client="${escapeHTML(selected.clientId)}">＋ Assign a Date &amp; Preparer</button>` : '<p>Search for a client to get started.</p>'}</div></div>`;
}
async function setFirmCalendarStatus(item, status) {
  const generation = sessionGeneration;
  try {
    const assignments = await backend().SetFirmCalendarStatus2026(item.date, item.clientId, status);
    if (generation !== sessionGeneration) return;
    firmCalendarAssignments = assignments;
    renderFirmCalendarWeek();
    toast('Calendar status saved.');
  } catch (error) { if (generation === sessionGeneration) showError(error); }
}

function firmCalendarAssignmentsForDate(dateISO) {
  return calendarAssignmentsForPreparer(firmCalendarAssignments, firmCalendarPreparerView)
    .filter(item => item.date === dateISO)
    .sort((a, b) => calendarPreparerName(a.preparerUsername).localeCompare(calendarPreparerName(b.preparerUsername)) || String(a.clientName || '').localeCompare(String(b.clientName || '')));
}

function renderFirmCalendarWeek() {
  if (!firmCalendarWeekStarts.length) return;
  firmCalendarWeekIndex = Math.max(0, Math.min(firmCalendarWeekIndex, firmCalendarWeekStarts.length - 1));
  const start = firmCalendarWeekStarts[firmCalendarWeekIndex];
  const end = firmCalendarWeekEnd(start);
  $('firmCalendarWeekSelect').value = String(firmCalendarWeekIndex);
  $('firmCalendarWeekLabel').textContent = `${firmCalendarFormatDate(start, { month: 'long', day: 'numeric' })} – ${firmCalendarFormatDate(end, { month: 'long', day: 'numeric', year: 'numeric' })}`;
  $('firmCalendarPrevWeek').disabled = firmCalendarWeekIndex === 0;
  $('firmCalendarNextWeek').disabled = firmCalendarWeekIndex === firmCalendarWeekStarts.length - 1;

  const now = new Date();
  const todayISO = firmCalendarISO(firmCalendarUTCDate(now.getFullYear(), now.getMonth(), now.getDate()));
  const dayCards = [];
  for (let offset = 0; offset < 7; offset++) {
    const date = new Date(start);
    date.setUTCDate(date.getUTCDate() + offset);
    const iso = firmCalendarISO(date);
    const inYear = date.getUTCFullYear() === 2026;
    const assignments = firmCalendarAssignmentsForDate(iso);
    const clientRows = assignments.length
      ? assignments.map(item => {
          const unavailable = !item.clientPath;
          const selected = item.clientId === firmCalendarSelectedClientID;
          const returnType = String(item.returnType || '').trim().toUpperCase();
          const returnTypeBadge = returnType ? `<span class="firm-calendar-return-type" title="Client return type">${escapeHTML(returnType)}</span>` : '<span class="firm-calendar-return-type is-missing" title="Return type not set">—</span>';
          return `<div class="firm-calendar-preparer-label">${escapeHTML(calendarPreparerName(item.preparerUsername))}</div><div class="firm-calendar-client-chip ${unavailable ? 'is-unavailable' : ''} ${selected ? 'is-selected' : ''}" data-calendar-assignment-date="${iso}" data-calendar-assignment-client="${escapeHTML(item.clientId)}">
            <button class="firm-calendar-client-link" type="button" data-calendar-client-id="${escapeHTML(item.clientId)}" data-calendar-date="${iso}" ${unavailable ? 'disabled' : ''} title="${unavailable ? 'Client is no longer active' : 'View client documents • Right-click to reschedule'}">
              <span class="firm-calendar-client-name">👶 ${escapeHTML(item.clientName || item.clientId)}</span>${firmCalendarStatusBadge(item.status)}
              ${returnTypeBadge}
            </button>
            <button class="firm-calendar-remove" type="button" data-calendar-remove-date="${iso}" data-calendar-remove-client="${escapeHTML(item.clientId)}" title="Remove from this day">×</button>
          </div>`;
        }).join('')
      : '<div class="firm-calendar-no-clients">No clients assigned</div>';

    dayCards.push(`<article class="firm-calendar-day ${inYear ? '' : 'is-outside-year'} ${iso === todayISO ? 'is-today' : ''}">
      <header class="firm-calendar-day-header">
        <div>
          <span class="firm-calendar-day-name">${firmCalendarFormatDate(date, { weekday: 'short' })}</span>
          <strong>${firmCalendarFormatDate(date, { month: 'short', day: 'numeric' })}</strong>
        </div>
        ${inYear ? `<button type="button" class="firm-calendar-add" data-calendar-add-date="${iso}" title="Assign a client to ${iso}">＋</button>` : ''}
      </header>
      <div class="firm-calendar-day-clients">${clientRows}</div>
    </article>`);
  }
  $('firmCalendarGrid').innerHTML = dayCards.join('');
  renderFirmCalendarSearch();
  $('firmCalendarStatus').textContent = `${firmCalendarAssignments.length} assignment${firmCalendarAssignments.length === 1 ? '' : 's'} in the 2026 company calendar. Multiple clients can be assigned to the same day.`;
}

async function firmCalendarAssignClient(dateISO) {
  // Temporarily hide the calendar while the existing client picker is active so
  // the picker remains the top-most Wails modal.
  $('firmCalendarBackdrop').classList.add('hidden');
  let client = null;
  try {
    client = await chooseClient('Assign Client to Firm Calendar', `Choose a client to schedule for ${firmCalendarFormatDate(firmCalendarDateFromISO(dateISO), { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}.`);
  } finally {
    $('firmCalendarBackdrop').classList.remove('hidden');
  }
  if (!client) return;
  firmCalendarAssignments = await backend().AssignClientToFirmCalendarForPreparer2026(dateISO, client.path, firmCalendarPreparerView === '*' ? currentUser.username : firmCalendarPreparerView);
  renderFirmCalendarWeek();
  toast(`${client.name} assigned to ${firmCalendarFormatDate(firmCalendarDateFromISO(dateISO), { month: 'short', day: 'numeric' })}.`);
}

async function firmCalendarRemoveClient(dateISO, clientID) {
  const item = firmCalendarAssignments.find(a => a.date === dateISO && a.clientId === clientID);
  if (!item) return;
  if (!confirm(`Remove ${item.clientName || 'this client'} from ${firmCalendarFormatDate(firmCalendarDateFromISO(dateISO), { month: 'long', day: 'numeric', year: 'numeric' })}?`)) return;
  firmCalendarAssignments = await backend().RemoveClientFromFirmCalendar2026(dateISO, clientID);
  if (firmCalendarSelectedClientID === clientID && !firmCalendarAssignments.some(a => a.clientId === clientID)) clearFirmCalendarClientDetail();
  renderFirmCalendarWeek();
}

function showFirmCalendarContextMenu(dateISO, clientID, x, y) {
  const item = firmCalendarAssignments.find(a => a.date === dateISO && a.clientId === clientID);
  if (!item) return;
  firmCalendarContextAssignment = item;
  const label = $('firmCalendarContextLabel');
  if (label) label.textContent = `${item.clientName || item.clientId}${item.returnType ? ` • ${String(item.returnType).toUpperCase()}` : ''}`;
  positionContextMenu($('firmCalendarContextMenu'), x, y);
}

function hideFirmCalendarReassign() {
  $('firmCalendarReassignBackdrop').classList.add('hidden');
}

function firmCalendarWeekIndexForDate(dateISO) {
  const target = firmCalendarDateFromISO(dateISO);
  for (let i = 0; i < firmCalendarWeekStarts.length; i++) {
    const start = firmCalendarWeekStarts[i];
    const end = firmCalendarWeekEnd(start);
    if (target >= start && target <= end) return i;
  }
  return firmCalendarWeekIndex;
}

function showFirmCalendarReassign() {
  const item = firmCalendarContextAssignment;
  if (!item) return;
  hideAllContextMenus();
  $('firmCalendarReassignClient').textContent = item.clientName || item.clientId;
  $('firmCalendarReassignCurrent').textContent = firmCalendarFormatDate(firmCalendarDateFromISO(item.date), { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' });
  $('firmCalendarReassignDate').value = item.date;
  $('firmCalendarReassignBackdrop').classList.remove('hidden');
  setTimeout(() => $('firmCalendarReassignDate')?.focus(), 20);
}

async function saveFirmCalendarReassign() {
  const item = firmCalendarContextAssignment;
  if (!item) return;
  const newDate = $('firmCalendarReassignDate').value;
  if (!newDate) { toast('Choose a new date.', true); return; }
  if (!/^2026-\d{2}-\d{2}$/.test(newDate)) { toast('Firm Calendar dates must be in 2026.', true); return; }
  try {
    firmCalendarAssignments = await backend().ReassignClientInFirmCalendar2026(item.date, newDate, item.clientId);
    firmCalendarContextAssignment = firmCalendarAssignments.find(a => a.date === newDate && a.clientId === item.clientId) || null;
    firmCalendarWeekIndex = firmCalendarWeekIndexForDate(newDate);
    hideFirmCalendarReassign();
    renderFirmCalendarWeek();
    toast(`${item.clientName || 'Client'} moved to ${firmCalendarFormatDate(firmCalendarDateFromISO(newDate), { weekday: 'short', month: 'short', day: 'numeric' })}.`);
  } catch (e) {
    showError(e);
  }
}

function findFirmCalendarClient(clientID) {
  return treeData.find(node => node.type === 'client' && node.clientId === clientID) || null;
}

function flattenCalendarDocuments(node, prefix = '') {
  const out = [];
  for (const child of node?.children || []) {
    const location = prefix ? `${prefix} / ${child.name}` : child.name;
    if (child.type === 'file') out.push({ node: child, location: prefix || node.name });
    else out.push(...flattenCalendarDocuments(child, location));
  }
  return out;
}

function renderFirmCalendarDocumentGroup(title, node, kind = 'year') {
  const docs = flattenCalendarDocuments(node);
  const rows = docs.length
    ? docs.map(({ node: doc, location }) => `<button type="button" class="firm-calendar-document-row" data-calendar-document-path="${escapeHTML(doc.path)}">
        <span class="firm-calendar-document-icon">${escapeHTML(iconFor(doc))}</span>
        <span class="firm-calendar-document-copy"><strong>${escapeHTML(doc.name)}</strong><small>${escapeHTML(location || title)}</small></span>
      </button>`).join('')
    : '<div class="firm-calendar-documents-empty">No documents in this area.</div>';
  return `<section class="firm-calendar-document-group ${kind}">
    <h4>${escapeHTML(title)}</h4>
    <div>${rows}</div>
  </section>`;
}

function showFirmCalendarClientDocuments(clientID) {
  const client = findFirmCalendarClient(clientID);
  firmCalendarSelectedClientID = clientID;
  renderFirmCalendarWeek();
  if (!client) {
    $('firmCalendarClientEmpty').classList.remove('hidden');
    $('firmCalendarClientDetail').classList.add('hidden');
    $('firmCalendarClientEmpty').innerHTML = '<div class="empty-icon">⚠️</div><h3>Client unavailable</h3><p>This scheduled client is no longer in the active client tree. Remove the assignment or unarchive the client first.</p>';
    return;
  }

  $('firmCalendarClientEmpty').classList.add('hidden');
  $('firmCalendarClientDetail').classList.remove('hidden');
  $('firmCalendarClientName').textContent = client.name;
  const assignment = firmCalendarAssignments.find(item => item.clientId === clientID);
  const returnType = assignment?.returnType ? String(assignment.returnType).toUpperCase() : '—';
  $('firmCalendarClientMeta').textContent = `Client ID ${client.clientId || '—'} • Return Type ${returnType} • Click a document below to open it in BabyFileCab.`;

  const permanent = (client.children || []).find(child => child.type === 'permanent');
  const years = (client.children || []).filter(child => child.type === 'taxyear');
  const clientSections = (client.children || []).filter(child => child.type === 'section');
  const groups = [];
  if (permanent) groups.push(renderFirmCalendarDocumentGroup('Permanent Folder', permanent, 'permanent'));
  for (const year of years) groups.push(renderFirmCalendarDocumentGroup(`Tax Year ${year.name}`, year, 'year'));
  for (const section of clientSections) groups.push(renderFirmCalendarDocumentGroup(section.name, section, 'section'));
  $('firmCalendarClientDocuments').innerHTML = groups.length ? groups.join('') : '<div class="firm-calendar-documents-empty">No client folders are available yet.</div>';
}

async function openFirmCalendarClientInTree() {
  const client = findFirmCalendarClient(firmCalendarSelectedClientID);
  if (!client) return;
  hideFirmCalendar();
  $('treeSearch').value = '';
  expanded.add(client.path);
  await refreshTree(client.path);
}

async function openFirmCalendarDocument(path) {
  if (!path) return;
  hideFirmCalendar();
  $('treeSearch').value = '';
  expandAncestors(path);
  await refreshTree(path);
}


function populateCurrencyConverterCurrencies() {
  const select = $('currencyConverterCurrency');
  if (!select) return;
  const previous = select.value || 'EUR';
  select.innerHTML = IRS_YEARLY_EXCHANGE_RATES
    .map(item => `<option value="${escapeHTML(item.code)}">${escapeHTML(item.country)} — ${escapeHTML(item.currency)} (${escapeHTML(item.code)})</option>`)
    .join('');
  if (IRS_YEARLY_EXCHANGE_RATES.some(item => item.code === previous)) select.value = previous;
  else if (IRS_YEARLY_EXCHANGE_RATES.some(item => item.code === 'EUR')) select.value = 'EUR';
}

function selectedIRSCurrency() {
  const code = $('currencyConverterCurrency')?.value || '';
  return IRS_YEARLY_EXCHANGE_RATES.find(item => item.code === code) || IRS_YEARLY_EXCHANGE_RATES[0];
}

function currencyConverterNumber() {
  const raw = String($('currencyConverterAmount')?.value || '').replace(/[$,\s]/g, '');
  if (!raw) return 0;
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0) throw new Error('Enter a valid non-negative amount to convert.');
  return value;
}

function formatIRSRate(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  if (Math.abs(n) >= 1e12) return n.toLocaleString('en-US', { maximumFractionDigits: 1 });
  return n.toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 });
}

function formatForeignAmount(value, code) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const maximumFractionDigits = Math.abs(n) >= 1000000 ? 2 : 4;
  return `${n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits })} ${code}`;
}

function updateCurrencyConverterUI() {
  const item = selectedIRSCurrency();
  const year = Number($('currencyConverterYear')?.value || 2025);
  const direction = $('currencyConverterDirection')?.value || 'foreign-to-usd';
  const rate = item?.rates?.[year];
  $('currencyConverterAmountLabel').textContent = direction === 'foreign-to-usd'
    ? `${item.code} amount to convert`
    : 'U.S. dollar amount to convert';
  $('currencyConverterRateHint').textContent = Number.isFinite(rate)
    ? `${year}: 1 USD = ${formatIRSRate(rate)} ${item.code} (${item.country} — ${item.currency})`
    : 'No IRS yearly-average rate is available for this selection.';
  calculateCurrencyConversion(false);
}

function calculateCurrencyConversion(showValidation = true) {
  try {
    const item = selectedIRSCurrency();
    const year = Number($('currencyConverterYear')?.value || 2025);
    const direction = $('currencyConverterDirection')?.value || 'foreign-to-usd';
    const rate = Number(item?.rates?.[year]);
    if (!Number.isFinite(rate) || rate <= 0) throw new Error('No IRS yearly-average rate is available for this currency and year.');
    const amount = currencyConverterNumber();
    let converted = 0;
    if (direction === 'foreign-to-usd') {
      converted = amount / rate;
      $('currencyConverterResultLabel').textContent = `${formatForeignAmount(amount, item.code)} converted to U.S. dollars`;
      $('currencyConverterResult').textContent = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(converted);
      $('currencyConverterFormula').textContent = `${amount.toLocaleString('en-US')} ${item.code} ÷ ${formatIRSRate(rate)} = ${converted.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} USD`;
    } else {
      converted = amount * rate;
      $('currencyConverterResultLabel').textContent = `${new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount)} converted to ${item.code}`;
      $('currencyConverterResult').textContent = formatForeignAmount(converted, item.code);
      $('currencyConverterFormula').textContent = `${amount.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} USD × ${formatIRSRate(rate)} = ${converted.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 4 })} ${item.code}`;
    }
    $('currencyConverterRateDisplay').textContent = `1 USD = ${formatIRSRate(rate)} ${item.code} (${year})`;
  } catch (e) {
    if (showValidation) showError(e);
  }
}

function showCurrencyConverter() {
  populateCurrencyConverterCurrencies();
  $('currencyConverterYear').value = '2025';
  $('currencyConverterDirection').value = 'foreign-to-usd';
  if (IRS_YEARLY_EXCHANGE_RATES.some(item => item.code === 'EUR')) $('currencyConverterCurrency').value = 'EUR';
  $('currencyConverterAmount').value = '0.00';
  $('currencyConverterBackdrop').classList.remove('hidden');
  updateCurrencyConverterUI();
  setTimeout(() => $('currencyConverterAmount').focus(), 30);
}

function hideCurrencyConverter() {
  $('currencyConverterBackdrop').classList.add('hidden');
}

function resetCurrencyConverter() {
  $('currencyConverterYear').value = '2025';
  $('currencyConverterDirection').value = 'foreign-to-usd';
  $('currencyConverterCurrency').value = IRS_YEARLY_EXCHANGE_RATES.some(item => item.code === 'EUR') ? 'EUR' : IRS_YEARLY_EXCHANGE_RATES[0].code;
  $('currencyConverterAmount').value = '0.00';
  updateCurrencyConverterUI();
  $('currencyConverterAmount').focus();
}

function taxNumber(id) {
  const raw = String($(id)?.value ?? '').replace(/[$,\s]/g, '');
  if (!raw) return 0;
  const value = Number(raw);
  if (!Number.isFinite(value)) throw new Error(`Enter a valid amount for ${$(id)?.closest('label')?.querySelector('span')?.textContent || id}.`);
  return value;
}

function taxCurrency(value) {
  const n = Math.abs(value) < .005 ? 0 : value;
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(n);
}

function taxPercent(value) {
  return `${Math.round(value * 100)}%`;
}

function calculateOrdinaryTax(income, filingStatus, config) {
  const taxable = Math.max(0, income);
  const brackets = config.ordinaryBrackets[filingStatus];
  let tax = 0;
  let previous = 0;
  let marginalRate = 0;
  for (const [limit, rate] of brackets) {
    if (taxable <= previous) break;
    const slice = Math.min(taxable, limit) - previous;
    if (slice > 0) {
      tax += slice * rate;
      marginalRate = rate;
    }
    if (taxable <= limit) break;
    previous = limit;
  }
  return { tax, marginalRate };
}

function classifyCapitalItems(shortTerm, longTerm, filingStatus, config) {
  let ordinaryGain = 0;
  let preferentialGain = 0;
  let lossDeduction = 0;
  let unusedLoss = 0;
  const lossLimit = config.capitalLossLimit[filingStatus];

  if (shortTerm >= 0 && longTerm >= 0) {
    ordinaryGain = shortTerm;
    preferentialGain = longTerm;
  } else {
    const net = shortTerm + longTerm;
    if (shortTerm < 0 && longTerm >= 0 && net > 0) {
      preferentialGain = net;
    } else if (shortTerm >= 0 && longTerm < 0 && net > 0) {
      ordinaryGain = net;
    } else if (net < 0) {
      const totalLoss = Math.abs(net);
      lossDeduction = Math.min(totalLoss, lossLimit);
      unusedLoss = Math.max(0, totalLoss - lossDeduction);
    }
  }

  return { ordinaryGain, preferentialGain, lossDeduction, unusedLoss };
}

function calculateLongTermCapitalTax(longTermTaxable, ordinaryTaxable, filingStatus, config) {
  let remaining = Math.max(0, longTermTaxable);
  let position = Math.max(0, ordinaryTaxable);
  const thresholds = config.capitalGains[filingStatus];

  const zeroAmount = Math.min(remaining, Math.max(0, thresholds.zeroMax - position));
  remaining -= zeroAmount;
  position += zeroAmount;

  const fifteenAmount = Math.min(remaining, Math.max(0, thresholds.fifteenMax - position));
  remaining -= fifteenAmount;
  const twentyAmount = Math.max(0, remaining);

  return {
    zeroAmount,
    fifteenAmount,
    twentyAmount,
    tax: fifteenAmount * .15 + twentyAmount * .20,
  };
}

function calculateChildTaxCredit({ children, adjustedGrossIncome, incomeTax, earnedIncome, filingStatus, config }) {
  const rules = config.childTaxCredit;
  const count = Math.max(0, Math.floor(children));
  if (!count) {
    return {
      maximumCredit: 0,
      phaseoutReduction: 0,
      creditAfterPhaseout: 0,
      nonrefundableCredit: 0,
      refundableCredit: 0,
      earnedIncomeAmount: Math.max(0, earnedIncome),
    };
  }

  const maximumCredit = count * rules.perChild;
  const phaseoutThreshold = rules.phaseoutThreshold[filingStatus];
  const excessMAGI = Math.max(0, adjustedGrossIncome - phaseoutThreshold);
  const phaseoutReduction = excessMAGI > 0 ? Math.ceil(excessMAGI / 1000) * 50 : 0;
  const creditAfterPhaseout = Math.max(0, maximumCredit - phaseoutReduction);

  // The regular CTC is nonrefundable and limited by income tax before credits.
  const nonrefundableCredit = Math.min(creditAfterPhaseout, Math.max(0, incomeTax));
  const unusedCredit = Math.max(0, creditAfterPhaseout - nonrefundableCredit);

  // Simplified ACTC estimate using the Schedule 8812 earned-income method.
  // The special alternative calculation for some taxpayers with 3+ children is not modeled.
  const earnedIncomeAmount = Math.max(0, earnedIncome);
  const earnedIncomeFormula = Math.max(0, earnedIncomeAmount - rules.earnedIncomeThreshold) * .15;
  const refundableCredit = Math.min(
    unusedCredit,
    count * rules.refundablePerChild,
    earnedIncomeFormula,
  );

  return {
    maximumCredit,
    phaseoutReduction,
    creditAfterPhaseout,
    nonrefundableCredit,
    refundableCredit,
    earnedIncomeAmount,
  };
}

function resetTaxCalculator() {
  $('taxFilingStatus').value = 'single';
  ['taxWages', 'taxSelfEmployment', 'taxInterest', 'taxShortCapital', 'taxLongCapital', 'taxFederalWithholding', 'taxQualifyingChildren'].forEach(id => { $(id).value = '0'; });
  $('taxCalculatorResults').classList.add('hidden');
  $('taxWages').focus();
}

function calculateTaxEstimate() {
  const year = selectedTaxYear();
  const config = TAX_YEARS[year];
  const filingStatus = $('taxFilingStatus').value;
  const wages = taxNumber('taxWages');
  const selfEmployment = taxNumber('taxSelfEmployment');
  const interest = taxNumber('taxInterest');
  const shortCapital = taxNumber('taxShortCapital');
  const longCapital = taxNumber('taxLongCapital');
  const federalWithholding = taxNumber('taxFederalWithholding');
  if (federalWithholding < 0) throw new Error('Federal withholding cannot be negative.');
  const qualifyingChildrenRaw = taxNumber('taxQualifyingChildren');
  if (!Number.isInteger(qualifyingChildrenRaw) || qualifyingChildrenRaw < 0) throw new Error('Enter a whole number of qualifying children (0 or more).');
  const qualifyingChildren = qualifyingChildrenRaw;

  const capital = classifyCapitalItems(shortCapital, longCapital, filingStatus, config);

  // Schedule SE generally applies the 92.35% factor to positive net SE profit.
  const seNetEarnings = selfEmployment > 0 ? selfEmployment * .9235 : 0;
  const seTaxApplies = seNetEarnings >= 400;
  const socialSecurityRoom = Math.max(0, config.socialSecurityWageBase - Math.max(0, wages));
  const seSocialSecurityBase = seTaxApplies ? Math.min(seNetEarnings, socialSecurityRoom) : 0;
  const seSocialSecurityTax = seSocialSecurityBase * .124;
  const seMedicareTax = seTaxApplies ? seNetEarnings * .029 : 0;
  const selfEmploymentTax = seSocialSecurityTax + seMedicareTax;
  const halfSETaxDeduction = selfEmploymentTax * .5;

  const additionalMedicareThreshold = config.additionalMedicareThreshold[filingStatus];
  const additionalMedicareTax = Math.max(0, Math.max(0, wages) + seNetEarnings - additionalMedicareThreshold) * .009;

  const ordinaryIncomeBeforeStandardDeduction =
    wages + selfEmployment + interest + capital.ordinaryGain - capital.lossDeduction - halfSETaxDeduction;
  const adjustedGrossIncome = ordinaryIncomeBeforeStandardDeduction + capital.preferentialGain;
  const standardDeduction = config.standardDeduction[filingStatus];
  const taxableIncome = Math.max(0, adjustedGrossIncome - standardDeduction);
  const ordinaryTaxableIncome = Math.max(0, Math.min(taxableIncome, ordinaryIncomeBeforeStandardDeduction - standardDeduction));
  const longTermTaxableIncome = Math.max(0, taxableIncome - ordinaryTaxableIncome);

  const ordinary = calculateOrdinaryTax(ordinaryTaxableIncome, filingStatus, config);
  const capitalTax = calculateLongTermCapitalTax(longTermTaxableIncome, ordinaryTaxableIncome, filingStatus, config);
  const incomeTaxBeforeCredits = ordinary.tax + capitalTax.tax;

  // Approximate Schedule 8812 earned income from the fields this planning calculator collects.
  const earnedIncomeForACTC = Math.max(0, Math.max(0, wages) + selfEmployment - halfSETaxDeduction);
  const childCredit = calculateChildTaxCredit({
    children: qualifyingChildren,
    adjustedGrossIncome,
    incomeTax: incomeTaxBeforeCredits,
    earnedIncome: earnedIncomeForACTC,
    filingStatus,
    config,
  });

  const incomeTaxAfterCTC = Math.max(0, incomeTaxBeforeCredits - childCredit.nonrefundableCredit);
  const totalFederalTax = incomeTaxAfterCTC + selfEmploymentTax + additionalMedicareTax - childCredit.refundableCredit;
  const estimatedTaxToCover = Math.max(0, totalFederalTax - federalWithholding);
  const equalQuarterlyPayment = estimatedTaxToCover / 4;

  $('taxResultTotal').textContent = taxCurrency(totalFederalTax);
  $('taxResultAGI').textContent = taxCurrency(adjustedGrossIncome);
  $('taxResultStandardDeduction').textContent = taxCurrency(standardDeduction);
  $('taxResultTaxableIncome').textContent = taxCurrency(taxableIncome);
  $('taxResultMarginalRate').textContent = taxPercent(ordinary.marginalRate);
  $('taxResultOrdinaryTax').textContent = taxCurrency(ordinary.tax);
  $('taxResultCapitalTax').textContent = taxCurrency(capitalTax.tax);
  $('taxResultIncomeTaxBeforeCredits').textContent = taxCurrency(incomeTaxBeforeCredits);
  $('taxResultCTCMaximum').textContent = taxCurrency(childCredit.maximumCredit);
  $('taxResultCTCPhaseout').textContent = childCredit.phaseoutReduction ? `(${taxCurrency(childCredit.phaseoutReduction)})` : '$0';
  $('taxResultCTC').textContent = childCredit.nonrefundableCredit ? `(${taxCurrency(childCredit.nonrefundableCredit)})` : '$0';
  $('taxResultACTC').textContent = childCredit.refundableCredit ? `(${taxCurrency(childCredit.refundableCredit)})` : '$0';
  $('taxResultIncomeTaxAfterCTC').textContent = taxCurrency(incomeTaxAfterCTC);
  $('taxResultSETax').textContent = taxCurrency(selfEmploymentTax);
  $('taxResultAdditionalMedicare').textContent = taxCurrency(additionalMedicareTax);
  $('taxResultHalfSE').textContent = `(${taxCurrency(halfSETaxDeduction)})`;
  $('taxResultCapitalLossDeduction').textContent = capital.lossDeduction ? `(${taxCurrency(capital.lossDeduction)})` : '$0';
  $('taxResultCapitalLossCarryforward').textContent = taxCurrency(capital.unusedLoss);
  $('taxResultCG0').textContent = taxCurrency(capitalTax.zeroAmount);
  $('taxResultCG15').textContent = taxCurrency(capitalTax.fifteenAmount);
  $('taxResultCG20').textContent = taxCurrency(capitalTax.twentyAmount);
  $('taxResultFederalWithholding').textContent = federalWithholding ? `(${taxCurrency(federalWithholding)})` : '$0';
  $('taxResultEstimatedTaxToCover').textContent = taxCurrency(estimatedTaxToCover);
  $('taxResultQuarterlyPayment').textContent = taxCurrency(equalQuarterlyPayment);
  config.estimatedTaxDueDates.forEach((date, index) => {
    const n = index + 1;
    $(`taxQuarter${n}Date`).textContent = date;
    $(`taxQuarter${n}Amount`).textContent = taxCurrency(equalQuarterlyPayment);
  });
  $('taxResultYearLabel').textContent = `Estimated ${year} federal tax after CTC / ACTC`;
  $('taxCalculatorResults').classList.remove('hidden');
  $('taxCalculatorResults').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

async function showManageUsers() {
  if (!currentUser) return;
  try {
    const users = await backend().ListCompanyUsers();
    const body = $('manageUsersBody');
    body.innerHTML = users.length ? users.map(user => {
      const fullName = [user.firstName, user.lastName].filter(Boolean).join(' ') || user.username || '—';
      const enrolled = user.vaultStatus === 'Enrolled';
      const vaultCell = enrolled ? 'Enrolled' : String(currentUser.role || '').toLowerCase() === 'administrator'
        ? `<button type="button" class="secondary" data-enroll-vault="${escapeHTML(user.username || '')}">Enroll user</button>` : 'Pending';
      return `<tr>
        <td>${escapeHTML(fullName)}</td>
        <td>@${escapeHTML(user.username || '')}</td>
        <td>${escapeHTML(displayRole(user.role))}</td>
        <td>${escapeHTML(user.email || '—')}</td>
        <td>${escapeHTML(user.phone || '—')}</td>
        <td>${vaultCell}</td>
      </tr>`;
    }).join('') : '<tr><td colspan="6">No users found for this company.</td></tr>';
    $('manageUsersAdd').classList.toggle('hidden', String(currentUser.role || '').toLowerCase() !== 'administrator');
    $('manageUsersBackdrop').classList.remove('hidden');
    setTimeout(() => $('manageUsersClose').focus(), 20);
  } catch (e) {
    showError(e);
  }
}

function showVaultEnrollment(username) {
  if (String(currentUser?.role || '').toLowerCase() !== 'administrator') return;
  pendingVaultEnrollmentUser = username;
  $('vaultEnrollmentPassword').value = '';
  $('vaultEnrollmentTitle').textContent = `Enroll @${username} in the company vault`;
  $('vaultEnrollmentBackdrop').classList.remove('hidden');
  setTimeout(() => $('vaultEnrollmentPassword').focus(), 20);
}

function hideVaultEnrollment() {
  $('vaultEnrollmentBackdrop').classList.add('hidden');
  $('vaultEnrollmentPassword').value = '';
  pendingVaultEnrollmentUser = '';
}

async function completeVaultEnrollment() {
  const password = $('vaultEnrollmentPassword').value;
  if (!pendingVaultEnrollmentUser || !password) { toast('Enter your own password.', true); return; }
  const button = $('vaultEnrollmentSave');
  button.disabled = true;
  try {
    await backend().EnrollExistingUser(pendingVaultEnrollmentUser, password);
    hideVaultEnrollment();
    toast('Vault access enrolled. Sign out and back in to verify.');
    await showManageUsers();
  } catch (e) { showError(e); $('vaultEnrollmentPassword').value = ''; }
  finally { button.disabled = false; }
}

function hideManageUsers() {
  $('manageUsersBackdrop').classList.add('hidden');
}

function appWorkspaceActive() {
  return Boolean(currentUser) && !$('appShell').classList.contains('hidden');
}

function activeElementIsEditable() {
  const el = document.activeElement;
  const tag = el?.tagName?.toLowerCase();
  return tag === 'input' || tag === 'textarea' || tag === 'select' || Boolean(el?.isContentEditable);
}

function modalIsOpen() {
  return Boolean(document.querySelector('.modal-backdrop:not(.hidden)'));
}

function focusPDFFind() {
  if (!currentPDF) return false;
  const input = $('pdfSearchInput');
  if (!input) return false;
  input.focus();
  input.select();
  return true;
}

function deleteHighlightedItem() {
  if (!selectedNode) return;
  if (selectedNode.type === 'file') doAction('delete-file');
  else if (selectedNode.type === 'client') doAction('delete-client');
  else if (selectedNode.type === 'taxyear') doAction('delete-year');
  else if (selectedNode.type === 'section') doAction('delete-section');
}

function openSettingsMenuFromShortcut() {
  closeAppMenus();
  const menu = $('settingsMenu');
  if (!menu) return;
  menu.classList.remove('hidden');
  document.querySelector('.menu-trigger[data-menu="settingsMenu"]')?.classList.add('active');
  const firstItem = menu.querySelector('button:not([disabled])');
  setTimeout(() => firstItem?.focus(), 20);
}

function armShortcutChord(value) {
  shortcutChord = value;
  clearTimeout(shortcutChordTimer);
  shortcutChordTimer = setTimeout(() => { shortcutChord = ''; }, 1200);
}


function passwordVisibilityIcon(visible) {
  const slash = visible ? '<path d="M4 4l16 16" />' : '';
  return `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <path d="M2.5 12s3.4-6.5 9.5-6.5 9.5 6.5 9.5 6.5-3.4 6.5-9.5 6.5S2.5 12 2.5 12Z" />
    <circle cx="12" cy="12" r="3" />
    ${slash}
  </svg>`;
}

function updatePasswordToggle(button, input) {
  const visible = input.type === 'text';
  button.innerHTML = passwordVisibilityIcon(visible);
  button.setAttribute('aria-pressed', visible ? 'true' : 'false');
  button.setAttribute('aria-label', visible ? 'Hide password' : 'Show password');
  button.title = visible ? 'Hide password' : 'Show password';
}

function installPasswordVisibilityToggles() {
  document.querySelectorAll('input[type="password"]').forEach(input => {
    if (input.closest('.password-input-wrap')) return;
    const wrap = document.createElement('div');
    wrap.className = 'password-input-wrap';
    input.parentNode.insertBefore(wrap, input);
    wrap.appendChild(input);

    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'password-visibility-toggle';
    button.dataset.passwordFor = input.id || '';
    updatePasswordToggle(button, input);
    button.addEventListener('mousedown', e => e.preventDefault());
    button.addEventListener('click', e => {
      e.preventDefault();
      e.stopPropagation();
      const start = input.selectionStart;
      const end = input.selectionEnd;
      input.type = input.type === 'password' ? 'text' : 'password';
      updatePasswordToggle(button, input);
      input.focus({ preventScroll: true });
      try { input.setSelectionRange(start, end); } catch (_) {}
    });
    wrap.appendChild(button);
  });
}

function hideAllPasswords() {
  document.querySelectorAll('.password-input-wrap input').forEach(input => {
    input.type = 'password';
    const button = input.parentElement?.querySelector('.password-visibility-toggle');
    if (button) updatePasswordToggle(button, input);
  });
}

function hideChangePassword() {
  $('changePasswordBackdrop').classList.add('hidden');
  ['changeCurrentPassword', 'changeNewPassword', 'changeConfirmPassword'].forEach(id => { $(id).value = ''; });
  hideAllPasswords();
}

async function changePassword() {
  const generation = sessionGeneration;
  const save = $('changePasswordSave');
  if (save.disabled) return;
  save.disabled = true;
  try {
    await backend().ChangePassword($('changeCurrentPassword').value, $('changeNewPassword').value, $('changeConfirmPassword').value);
    if (generation !== sessionGeneration) return;
    hideChangePassword();
    toast('Password changed. Use your new password next time you sign in.');
  } catch (e) {
    if (generation === sessionGeneration) showError(e);
  } finally { save.disabled = false; }
}

function wireEvents() {
  $('openFirmCalendarBtn').addEventListener('click', () => call(showFirmCalendar));
  $('calendarScheduleClientCancel').addEventListener('click',hideCalendarScheduleClient);
  $('calendarScheduleClientSave').addEventListener('click',saveCalendarScheduleClient);
  $('calendarScheduleClientBackdrop').addEventListener('click',e => {if(e.target===$('calendarScheduleClientBackdrop'))hideCalendarScheduleClient();});
  $('firmCalendarPreparerButton').addEventListener('click', () => {
    const menu=$('firmCalendarPreparerMenu'); menu.classList.toggle('hidden'); $('firmCalendarPreparerButton').setAttribute('aria-expanded',String(!menu.classList.contains('hidden')));
  });
  $('firmCalendarPreparerReassignCancel').addEventListener('click',hideCalendarPreparerReassign);
  $('firmCalendarPreparerReassignSave').addEventListener('click',saveCalendarPreparerReassign);
  $('firmCalendarPreparerReassignBackdrop').addEventListener('click',e => { if(e.target===$('firmCalendarPreparerReassignBackdrop')) hideCalendarPreparerReassign(); });
  $('shortcutMapperModify').addEventListener('click', () => showShortcutEditor(false));
  $('shortcutMapperClear').addEventListener('click', () => { if (shortcutSelectedRow) { shortcutDraft[shortcutSelectedRow.id].splice(shortcutSelectedRow.index,1); renderShortcutMapper(); } });
  $('shortcutEditCancel').addEventListener('click', () => { hideShortcutEditor(); $('shortcutMapperModify').focus(); });
  $('shortcutEditOK').addEventListener('click', applyShortcutEditor);
  ['Ctrl','Alt','Shift'].forEach(modifier => $(`shortcutEdit${modifier}`).addEventListener('change', updateShortcutEditorPreview));
  $('shortcutEditKey').addEventListener('change', updateShortcutEditorPreview);
  $('shortcutEditBackdrop').addEventListener('click', e => { if (e.target === $('shortcutEditBackdrop')) hideShortcutEditor(); });
  $('shortcutMapperCancel').addEventListener('click', hideShortcutMapper);
  $('shortcutMapperSave').addEventListener('click', saveShortcutMapper);
  $('shortcutMapperReset').addEventListener('click', () => { shortcutDraft = defaultShortcutBindings(); renderShortcutMapper(); });
  $('shortcutMapperBackdrop').addEventListener('click', e => { if (e.target === $('shortcutMapperBackdrop')) hideShortcutMapper(); });
  $('changePasswordCancel').addEventListener('click', hideChangePassword);
  $('changePasswordSave').addEventListener('click', changePassword);
  $('changeConfirmPassword').addEventListener('keydown', e => { if (e.key === 'Enter') changePassword(); });
  $('changePasswordBackdrop').addEventListener('click', e => { if (e.target === $('changePasswordBackdrop')) hideChangePassword(); });
  $('signInBtn').addEventListener('click', signIn);
  $('authUsername').addEventListener('keydown', e => { if (e.key === 'Enter') signIn(); });
  $('authPassword').addEventListener('keydown', e => { if (e.key === 'Enter') signIn(); });
  $('showCreateAccountBtn').addEventListener('click', showCreateAccount);
  $('showExistingCompanyUserBtn').addEventListener('click', showExistingCompanyUser);
  $('createAccountCancel').addEventListener('click', hideCreateAccount);
  $('createAccountSave').addEventListener('click', createAccount);
  $('createConfirmPassword').addEventListener('keydown', e => { if (e.key === 'Enter') createAccount(); });
  $('createAccountBackdrop').addEventListener('click', e => { if (e.target === $('createAccountBackdrop')) hideCreateAccount(); });
  $('existingCompanyUserCancel').addEventListener('click', hideExistingCompanyUser);
  $('existingCompanyUserSave').addEventListener('click', createExistingCompanyUser);
  $('existingAdminPassword').addEventListener('keydown', e => { if (e.key === 'Enter') createExistingCompanyUser(); });
  $('existingCompanyUserBackdrop').addEventListener('click', e => { if (e.target === $('existingCompanyUserBackdrop')) hideExistingCompanyUser(); });
  $('treeSearch').addEventListener('input', () => renderTree());
  $('appMenubar').addEventListener('click', e => {
    const trigger = e.target.closest('.menu-trigger[data-menu]');
    if (trigger) {
      e.stopPropagation();
      toggleAppMenu(trigger.dataset.menu);
      return;
    }
    const item = e.target.closest('[data-menu-action]');
    if (item) {
      e.stopPropagation();
      closeAppMenus();
      handleMenuAction(item.dataset.menuAction);
    }
  });
  $('toolbar').addEventListener('click', e => {
    const btn = e.target.closest('button[data-action]');
    if (btn) doAction(btn.dataset.action);
  });
  $('clientContextMenu').addEventListener('click', e => {
    const btn = e.target.closest('button[data-client-action]');
    if (!btn || !contextClient) return;
    const action = btn.dataset.clientAction;
    const client = contextClient;
    hideAllContextMenus();
    selectNode(client);
    if (action === 'edit') doAction('edit-client');
    else if (action === 'delete') doAction('delete-client');
    else if (action === 'add-year') doAction('add-year');
    else if (action === 'add-section') doAction('add-section');
    else if (action === 'properties') showClientProperties(client);
  });
  $('taxYearContextMenu').addEventListener('click', e => {
    const btn = e.target.closest('button[data-taxyear-action]');
    if (!btn || !contextTaxYear) return;
    const action = btn.dataset.taxyearAction;
    const year = contextTaxYear;
    hideAllContextMenus();
    selectNode(year);
    if (action === 'edit') doAction('edit-year');
    else if (action === 'delete') doAction('delete-year');
    else if (action === 'add-section') doAction('add-section');
  });
  $('sectionContextMenu').addEventListener('click', e => {
    const btn = e.target.closest('button[data-section-action]');
    if (!btn || !contextSection) return;
    const action = btn.dataset.sectionAction;
    const section = contextSection;
    hideAllContextMenus();
    selectNode(section);
    if (action === 'edit') doAction('edit-section');
    else if (action === 'upload') doAction('upload');
    else if (action === 'delete') doAction('delete-section');
  });
  $('documentContextMenu').addEventListener('click', e => {
    const btn = e.target.closest('button[data-document-action]');
    if (!btn || !contextDocument) return;
    const action = btn.dataset.documentAction;
    const doc = contextDocument;
    hideAllContextMenus();
    selectNode(doc);
    if (action === 'edit') doAction('rename-file');
    else if (action === 'open') call(async () => backend().OpenFile(doc.path));
    else if (action === 'delete') doAction('delete-file');
    else if (action === 'properties') showDocumentProperties(doc);
  });
  document.addEventListener('pointerdown', e => {
    if (!e.target.closest('.context-menu')) hideAllContextMenus();
    if (!e.target.closest('.app-menubar')) closeAppMenus();
  });
  window.addEventListener('blur', hideAllContextMenus);
  window.addEventListener('resize', hideAllContextMenus);
  $('propertiesClose').addEventListener('click', hideClientProperties);
  $('propertiesBackdrop').addEventListener('click', e => {
    if (e.target === $('propertiesBackdrop')) hideClientProperties();
  });
  $('documentPropertiesCancel').addEventListener('click', hideDocumentProperties);
  $('documentPropertiesSave').addEventListener('click', saveDocumentProperties);
  $('documentPropertiesBackdrop').addEventListener('click', e => {
    if (e.target === $('documentPropertiesBackdrop')) hideDocumentProperties();
  });
  $('globalFindClose').addEventListener('click', hideGlobalFind);
  $('globalFindBackdrop').addEventListener('click', e => {
    if (e.target === $('globalFindBackdrop')) hideGlobalFind();
  });
  $('globalFindSearch').addEventListener('click', runGlobalFind);
  $('globalFindInput').addEventListener('input', () => {
    clearTimeout(globalFindTimer);
    globalFindTimer = setTimeout(runGlobalFind, 220);
  });
  $('globalFindInput').addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); runGlobalFind(); }
    if (e.key === 'Escape') hideGlobalFind();
  });
  $('globalFindResults').addEventListener('click', async e => {
    const row = e.target.closest('[data-global-path]');
    if (!row) return;
    const path = row.dataset.globalPath;
    hideGlobalFind();
    expandAncestors(path);
    await refreshTree(path);
  });
  $('clientPickerCancel').addEventListener('click', () => closeClientPicker(null));
  $('clientPickerBackdrop').addEventListener('click', e => { if (e.target === $('clientPickerBackdrop')) closeClientPicker(null); });
  $('clientPickerSearch').addEventListener('input', renderClientPickerList);
  $('clientPickerList').addEventListener('click', e => {
    const row = e.target.closest('[data-client-picker-path]');
    if (!row) return;
    closeClientPicker(findNodeByPath(row.dataset.clientPickerPath, clientPickerItems));
  });
  $('clientCommunicationsClose').addEventListener('click', hideClientCommunications);
  $('clientCommunicationsBackdrop').addEventListener('click', e => { if (e.target === $('clientCommunicationsBackdrop')) hideClientCommunications(); });
  $('clientCommunicationsSearch').addEventListener('input', renderClientCommunications);
  $('clientCommunicationsSearch').addEventListener('keydown', e => { if (e.key === 'Escape') hideClientCommunications(); });
  $('clientCommunicationsBody').addEventListener('click', async e => {
    const row = e.target.closest('[data-communications-path]');
    if (!row) return;
    const path = row.dataset.communicationsPath;
    hideClientCommunications();
    $('treeSearch').value = '';
    expandAncestors(path);
    await refreshTree(path);
  });
  $('addClientCancel').addEventListener('click', () => closeClientForm(null));
  $('addClientSave').addEventListener('click', () => submitClientForm());
  $('addClientBackdrop').addEventListener('click', e => { if (e.target === $('addClientBackdrop')) closeClientForm(null); });
  $('clientNameInput').addEventListener('keydown', e => { if (e.key === 'Enter') submitClientForm(); });
  $('modalCancel').addEventListener('click', () => closeModal(null));
  $('modalOK').addEventListener('click', () => closeModal($('modalInput').value));
  $('modalInput').addEventListener('keydown', e => {
    if (e.key === 'Enter') closeModal($('modalInput').value);
    if (e.key === 'Escape') closeModal(null);
  });
  $('userManualClose').addEventListener('click', hideUserManual);
  $('userManualCloseTop').addEventListener('click', hideUserManual);
  $('userManualBackdrop').addEventListener('click', e => { if (e.target === $('userManualBackdrop')) hideUserManual(); });
  $('aboutClose').addEventListener('click', hideAbout);
  $('aboutCloseTop').addEventListener('click', hideAbout);
  $('aboutBackdrop').addEventListener('click', e => { if (e.target === $('aboutBackdrop')) hideAbout(); });
  $('engagementLetterClose').addEventListener('click', hideEngagementLetter);
  $('engagementLetterCloseTop').addEventListener('click', hideEngagementLetter);
  $('engagementLetterBackdrop').addEventListener('click', e => { if (e.target === $('engagementLetterBackdrop')) hideEngagementLetter(); });
  $('engagementClient').addEventListener('change', () => call(async () => { await engagementSelectionChanged({ client: true }); }));
  $('engagementPreparer').addEventListener('change', () => engagementSelectionChanged({ preparer: true }));
  $('engagementService').addEventListener('change', () => call(async () => { await engagementSelectionChanged(); }));
  $('engagementTaxYear').addEventListener('change', () => call(async () => engagementSelectionChanged()));
  $('engagementFeeType').addEventListener('change', () => engagementSelectionChanged());
  $('engagementFeeValue').addEventListener('input', () => renderEngagementPreview());
  $('engagementDate').addEventListener('change', () => renderEngagementPreview());
  $('engagementBrowseLogo').addEventListener('click', () => call(browseEngagementFirmLogo));
  $('engagementRemoveLogo').addEventListener('click', () => call(removeEngagementFirmLogo));
  $('engagementGeneratePDF').addEventListener('click', () => generateEngagementLetter('pdf'));
  $('engagementGenerateWord').addEventListener('click', () => generateEngagementLetter('docx'));
  $('invoiceClose').addEventListener('click', hideInvoiceGenerator);
  $('invoiceCloseTop').addEventListener('click', hideInvoiceGenerator);
  $('invoiceBackdrop').addEventListener('click', e => { if (e.target === $('invoiceBackdrop')) hideInvoiceGenerator(); });
  $('invoiceClient').addEventListener('change', () => call(async () => { await invoiceSelectionChanged({ client: true }); }));
  $('invoicePreparer').addEventListener('change', () => invoiceSelectionChanged({ preparer: true }));
  $('invoiceService').addEventListener('change', renderInvoicePreview);
  $('invoiceTaxYear').addEventListener('change', renderInvoicePreview);
  $('invoiceNumber').addEventListener('input', renderInvoicePreview);
  $('invoiceDate').addEventListener('change', renderInvoicePreview);
  $('invoiceDueDate').addEventListener('change', renderInvoicePreview);
  $('invoiceFeeType').addEventListener('change', updateInvoiceFeeUI);
  $('invoiceFeeValue').addEventListener('input', renderInvoicePreview);
  $('invoiceHours').addEventListener('input', renderInvoicePreview);
  $('invoiceDescription').addEventListener('input', renderInvoicePreview);
  $('invoiceNotes').addEventListener('input', renderInvoicePreview);
  $('invoiceBrowseLogo').addEventListener('click', () => call(browseInvoiceLogo));
  $('invoiceRemoveLogo').addEventListener('click', () => call(removeInvoiceLogo));
  $('invoiceGeneratePDF').addEventListener('click', () => generateInvoice('pdf'));
  $('invoiceGenerateWord').addEventListener('click', () => generateInvoice('docx'));
  $('appAuditClose').addEventListener('click', hideAppAuditTrail);
  $('appAuditCloseTop').addEventListener('click', hideAppAuditTrail);
  $('appAuditBackdrop').addEventListener('click', e => { if (e.target === $('appAuditBackdrop')) hideAppAuditTrail(); });
  $('appAuditSearch').addEventListener('input', renderAppAuditTrail);
  $('appAuditSearch').addEventListener('keydown', e => { if (e.key === 'Escape') hideAppAuditTrail(); });
  $('appAuditRefresh').addEventListener('click', () => call(refreshAppAuditTrail));
  $('currencyConverterClose').addEventListener('click', hideCurrencyConverter);
  $('currencyConverterCloseTop').addEventListener('click', hideCurrencyConverter);
  $('currencyConverterBackdrop').addEventListener('click', e => { if (e.target === $('currencyConverterBackdrop')) hideCurrencyConverter(); });
  $('currencyConverterYear').addEventListener('change', updateCurrencyConverterUI);
  $('currencyConverterDirection').addEventListener('change', updateCurrencyConverterUI);
  $('currencyConverterCurrency').addEventListener('change', updateCurrencyConverterUI);
  $('currencyConverterAmount').addEventListener('input', () => calculateCurrencyConversion(false));
  $('currencyConverterAmount').addEventListener('keydown', e => { if (e.key === 'Enter') calculateCurrencyConversion(true); if (e.key === 'Escape') hideCurrencyConverter(); });
  $('currencyConverterCalculate').addEventListener('click', () => calculateCurrencyConversion(true));
  $('currencyConverterReset').addEventListener('click', resetCurrencyConverter);
  $('currencyConverterOpenIRS').addEventListener('click', async () => {
    try { await backend().OpenIRSCurrencyExchangeRatesPage(); } catch (e) { showError(e); }
  });
  $('taxCalculatorClose').addEventListener('click', hideTaxCalculator);
  $('taxCalculatorCloseTop').addEventListener('click', hideTaxCalculator);
  $('taxCalculatorBackdrop').addEventListener('click', e => { if (e.target === $('taxCalculatorBackdrop')) hideTaxCalculator(); });
  $('taxCalculatorCalculate').addEventListener('click', () => { try { calculateTaxEstimate(); } catch (e) { showError(e); } });
  $('taxCalculatorReset').addEventListener('click', resetTaxCalculator);
  $('taxYear').addEventListener('change', updateTaxCalculatorYearUI);
  $('taxIRSPaymentButton').addEventListener('click', async () => {
    try { await backend().OpenIRSPaymentPage(); } catch (e) { showError(e); }
  });
  $('firmCalendarClose').addEventListener('click', hideFirmCalendar);
  $('firmCalendarCloseTop').addEventListener('click', hideFirmCalendar);
  $('firmCalendarBackdrop').addEventListener('click', e => { if (e.target === $('firmCalendarBackdrop')) hideFirmCalendar(); });
  $('firmCalendarPrevWeek').addEventListener('click', () => { if (firmCalendarWeekIndex > 0) { firmCalendarWeekIndex--; renderFirmCalendarWeek(); } });
  $('firmCalendarNextWeek').addEventListener('click', () => { if (firmCalendarWeekIndex < firmCalendarWeekStarts.length - 1) { firmCalendarWeekIndex++; renderFirmCalendarWeek(); } });
  $('firmCalendarCurrentWeek').addEventListener('click', () => { firmCalendarWeekIndex = firmCalendarCurrentWeekIndex(); renderFirmCalendarWeek(); });
  $('firmCalendarWeekSelect').addEventListener('change', () => { firmCalendarWeekIndex = Number($('firmCalendarWeekSelect').value || 0); renderFirmCalendarWeek(); });
  $('firmCalendarSearch').addEventListener('input', renderFirmCalendarSearch);
  $('firmCalendarSearchClear').addEventListener('click', () => { $('firmCalendarSearch').value = ''; renderFirmCalendarSearch(); $('firmCalendarSearch').focus(); });
  $('firmCalendarSearchResults').addEventListener('change', async e => {
    if(e.target.id!=='calendarSearchEditChoice')return;
    const action=e.target.value,item=calendarSearchSelectedAssignment();e.target.value='';
    if(!item)return;
    if(action==='preparer')showCalendarPreparerReassign(item);
    if(action==='date'){firmCalendarContextAssignment=item;showFirmCalendarReassign();}
    if(action==='clear')await setFirmCalendarStatus(item,'');
    if(action.startsWith('status-'))await setFirmCalendarStatus(item,action.slice(7));
  });
  $('firmCalendarSearchResults').addEventListener('click', async e => {
    const selected=e.target.closest('[data-calendar-select-client]');
    if(selected){calendarSearchSelectedClient=selected.dataset.calendarSelectClient;renderFirmCalendarSearch();return;}
    if(e.target.closest('[data-calendar-search-command="go"]')){const item=calendarSearchSelectedAssignment();if(item)goToClientOnCalendar(item.clientId,item.date);return;}

    const go=e.target.closest('[data-calendar-go-client]');
    if(go){goToClientOnCalendar(go.dataset.calendarGoClient,go.dataset.calendarGoDate);return;}
    const assign=e.target.closest('[data-calendar-schedule-client]');
    if(assign){showCalendarScheduleClient(assign.dataset.calendarScheduleClient);return;}
    const result = e.target.closest('[data-calendar-search-date]');
    if (!result) return;
    const date = firmCalendarDateFromISO(result.dataset.calendarSearchDate);
    const index = firmCalendarWeekStarts.findIndex(start => date >= start && date <= firmCalendarWeekEnd(start));
    if (index >= 0) { firmCalendarWeekIndex = index; renderFirmCalendarWeek(); }
    await showFirmCalendarClientDocuments(result.dataset.calendarClientId);
  });
  $('firmCalendarSearchResults').addEventListener('contextmenu', e => {
    const item = e.target.closest('[data-calendar-assignment-date][data-calendar-assignment-client]');
    if (!item) return;
    e.preventDefault(); e.stopPropagation();
    showFirmCalendarContextMenu(item.dataset.calendarAssignmentDate, item.dataset.calendarAssignmentClient, e.clientX, e.clientY);
  });
  $('firmCalendarGrid').addEventListener('click', async e => {
    const add = e.target.closest('[data-calendar-add-date]');
    if (add) { await firmCalendarAssignClient(add.dataset.calendarAddDate); return; }
    const remove = e.target.closest('[data-calendar-remove-date]');
    if (remove) { await firmCalendarRemoveClient(remove.dataset.calendarRemoveDate, remove.dataset.calendarRemoveClient); return; }
    const client = e.target.closest('[data-calendar-client-id]');
    if (client) showFirmCalendarClientDocuments(client.dataset.calendarClientId);
  });
  $('firmCalendarGrid').addEventListener('contextmenu', e => {
    const chip = e.target.closest('[data-calendar-assignment-date][data-calendar-assignment-client]');
    if (!chip) return;
    e.preventDefault();
    e.stopPropagation();
    showFirmCalendarContextMenu(chip.dataset.calendarAssignmentDate, chip.dataset.calendarAssignmentClient, e.clientX, e.clientY);
  });
  $('firmCalendarContextMenu').addEventListener('click', async e => {
    const btn = e.target.closest('button[data-calendar-context-action]');
    if (!btn || !firmCalendarContextAssignment) return;
    const item = firmCalendarContextAssignment;
    const action = btn.dataset.calendarContextAction;
    hideAllContextMenus();
    if (action.startsWith('status-')) { await setFirmCalendarStatus(item, action === 'status-clear' ? '' : action.slice(7)); return; }
    if (action === 'reassign-preparer') { showCalendarPreparerReassign(item); return; }
    if (action === 'reassign') { showFirmCalendarReassign(); return; }
    if (action === 'view') { showFirmCalendarClientDocuments(item.clientId); return; }
    if (action === 'remove') { await firmCalendarRemoveClient(item.date, item.clientId); }
  });
  $('firmCalendarReassignCancel').addEventListener('click', hideFirmCalendarReassign);
  $('firmCalendarReassignCloseTop').addEventListener('click', hideFirmCalendarReassign);
  $('firmCalendarReassignSave').addEventListener('click', saveFirmCalendarReassign);
  $('firmCalendarReassignBackdrop').addEventListener('click', e => { if (e.target === $('firmCalendarReassignBackdrop')) hideFirmCalendarReassign(); });
  $('firmCalendarReassignDate').addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); saveFirmCalendarReassign(); }
    if (e.key === 'Escape') { e.preventDefault(); hideFirmCalendarReassign(); }
  });
  $('firmCalendarClientDocuments').addEventListener('click', async e => {
    const doc = e.target.closest('[data-calendar-document-path]');
    if (doc) await openFirmCalendarDocument(doc.dataset.calendarDocumentPath);
  });
  $('firmCalendarOpenClient').addEventListener('click', openFirmCalendarClientInTree);
  ['taxWages', 'taxSelfEmployment', 'taxInterest', 'taxShortCapital', 'taxLongCapital'].forEach(id => {
    $(id).addEventListener('keydown', e => {
      if (e.key === 'Enter') {
        e.preventDefault();
        try { calculateTaxEstimate(); } catch (err) { showError(err); }
      }
    });
  });
  $('manageUsersClose').addEventListener('click', hideManageUsers);
  $('manageUsersBackdrop').addEventListener('click', e => { if (e.target === $('manageUsersBackdrop')) hideManageUsers(); });
  $('manageUsersAdd').addEventListener('click', async () => {
    hideManageUsers();
    await showExistingCompanyUser();
  });
  $('manageUsersBody').addEventListener('click', e => {
    const button = e.target.closest('[data-enroll-vault]');
    if (button) showVaultEnrollment(button.dataset.enrollVault);
  });
  $('vaultEnrollmentCancel').addEventListener('click', hideVaultEnrollment);
  $('vaultEnrollmentSave').addEventListener('click', completeVaultEnrollment);
  $('vaultEnrollmentPassword').addEventListener('keydown', e => { if (e.key === 'Enter') completeVaultEnrollment(); });
  $('vaultEnrollmentBackdrop').addEventListener('click', e => { if (e.target === $('vaultEnrollmentBackdrop')) hideVaultEnrollment(); });
  $('clientPickerSearch').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      const first = $('clientPickerList').querySelector('[data-client-picker-path]');
      if (first) {
        e.preventDefault();
        closeClientPicker(findNodeByPath(first.dataset.clientPickerPath, clientPickerItems));
      }
    } else if (e.key === 'Escape') {
      e.preventDefault();
      closeClientPicker(null);
    }
  });
  $('globalFindResults').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      const row = e.target.closest('[data-global-path]');
      if (row) { e.preventDefault(); row.click(); }
    }
  });

  document.addEventListener('keydown', async e => {
    const key = String(e.key || '');
    const lower = key.toLowerCase();
    const ctrl = e.ctrlKey || e.metaKey;
    const editingNotes = Boolean($('notesEditor')) && document.activeElement === $('notesEditor');
    const documentPropertiesOpen = !$('documentPropertiesBackdrop').classList.contains('hidden');

    // Context-sensitive Client Notes shortcuts take priority over the main
    // application Ctrl+B backup command.
    if (editingNotes && ctrl && !e.shiftKey && ['s', 'b', 'i', 'u'].includes(lower)) {
      e.preventDefault();
      if (lower === 's') $('saveNotesBtn')?.click();
      else if (lower === 'b') document.execCommand('bold', false, null);
      else if (lower === 'i') document.execCommand('italic', false, null);
      else if (lower === 'u') document.execCommand('underline', false, null);
      return;
    }

    if (!$('calendarScheduleClientBackdrop').classList.contains('hidden')) {
      if(key==='Escape'){e.preventDefault();hideCalendarScheduleClient();}
      return;
    }
    if (!$('firmCalendarPreparerReassignBackdrop').classList.contains('hidden')) {
      if (key === 'Escape') { e.preventDefault(); hideCalendarPreparerReassign(); }
      return;
    }

    // Document Properties deliberately saves on both Ctrl+S and Escape.
    if (documentPropertiesOpen) {
      if (ctrl && !e.shiftKey && lower === 's') {
        e.preventDefault();
        await saveDocumentProperties();
        return;
      }
      if (key === 'Escape') {
        e.preventDefault();
        await saveDocumentProperties();
        return;
      }
    }

    if (!$('shortcutEditBackdrop').classList.contains('hidden')) {
      if (key === 'Escape') { e.preventDefault(); hideShortcutEditor(); $('shortcutMapperModify').focus(); }
      return;
    }
    if (!$('shortcutMapperBackdrop').classList.contains('hidden')) {
      if (key === 'Escape') { e.preventDefault(); hideShortcutMapper(); }
      return;
    }
    if (!appWorkspaceActive() || e.repeat || e.isComposing) return;
    if (shortcutChord === 'g' && !ctrl && !e.altKey && !e.shiftKey && !activeElementIsEditable() && !modalIsOpen()) {
      shortcutChord = ''; clearTimeout(shortcutChordTimer);
      if (lower === 's') { e.preventDefault(); openSettingsMenuFromShortcut(); return; }
    }
    const binding = shortcutFromEvent(e);
    const action = SHORTCUT_ACTIONS.find(item => shortcutBindings[item.id].includes(binding));
    if (action && (action.id === 'help' || action.id === 'sign-out' || (!activeElementIsEditable() && !modalIsOpen()) || (action.id === 'pdf-find' && !modalIsOpen()))) {
      e.preventDefault();
      try { await runMappedShortcut(action.id); } catch (error) { showError(error); }
      return;
    }

    // Escape closes/cancels secondary windows. Search and Document Properties
    // have their own behavior above/below.
    if (key === 'Escape') {
      if (!$('vaultEnrollmentBackdrop').classList.contains('hidden')) { e.preventDefault(); hideVaultEnrollment(); return; }
      if (!$('userManualBackdrop').classList.contains('hidden')) { e.preventDefault(); hideUserManual(); return; }
      if (!$('engagementLetterBackdrop').classList.contains('hidden')) { e.preventDefault(); hideEngagementLetter(); return; }
      if (!$('taxCalculatorBackdrop').classList.contains('hidden')) { e.preventDefault(); hideTaxCalculator(); return; }
      if (!$('firmCalendarBackdrop').classList.contains('hidden')) { e.preventDefault(); hideFirmCalendar(); return; }
      if (!$('aboutBackdrop').classList.contains('hidden')) { e.preventDefault(); hideAbout(); return; }
      if (!$('manageUsersBackdrop').classList.contains('hidden')) { e.preventDefault(); hideManageUsers(); return; }
      if (!$('globalFindBackdrop').classList.contains('hidden')) { e.preventDefault(); hideGlobalFind(); return; }
      if (!$('clientCommunicationsBackdrop').classList.contains('hidden')) { e.preventDefault(); hideClientCommunications(); return; }
      if (!$('clientPickerBackdrop').classList.contains('hidden')) { e.preventDefault(); closeClientPicker(null); return; }
      if (!$('propertiesBackdrop').classList.contains('hidden')) { e.preventDefault(); hideClientProperties(); return; }
      if (!$('addClientBackdrop').classList.contains('hidden')) { e.preventDefault(); closeClientForm(null); return; }
      if (!$('modalBackdrop').classList.contains('hidden')) { e.preventDefault(); closeModal(null); return; }
      if (!$('existingCompanyUserBackdrop').classList.contains('hidden')) { e.preventDefault(); hideExistingCompanyUser(); return; }
      if (!$('createAccountBackdrop').classList.contains('hidden')) { e.preventDefault(); hideCreateAccount(); return; }
    }

    // Single-key workflow shortcuts only run in the main workspace while the
    // user is not typing and no secondary modal is open.
    if (ctrl || e.altKey || activeElementIsEditable() || modalIsOpen()) return;

    if (lower === 'g' && !e.shiftKey) {
      e.preventDefault();
      armShortcutChord('g');
      return;
    }


  });
}

async function refreshTree(selectPath = null) {
  const oldSelectedPath = selectPath || selectedNode?.path || '';
  treeData = await backend().GetTree();
  selectedFilePaths = new Set([...selectedFilePaths].filter(path => findNodeByPath(path, treeData)?.type === "file"));
  renderTree();
  if (oldSelectedPath) {
    const node = findNodeByPath(oldSelectedPath, treeData);
    if (node) {
      expandAncestors(node.path);
      renderTree();
      selectNode(node, null, true);
      scrollToPath(node.path);
    }
  }
}

function renderTree() {
  const root = $('tree');
  root.innerHTML = '';
  const q = $('treeSearch').value.trim().toLowerCase();
  const filtered = q ? filterNodes(treeData, q) : treeData;
  if (!filtered.length) {
    root.innerHTML = '<div class="tree-empty">No clients yet.<br><br>Click <b>Add Client</b> in the top-right toolbar.</div>';
    return;
  }
  for (const node of filtered) root.appendChild(renderNode(node));
}

function renderNode(node) {
  const hasChildren = node.children && node.children.length > 0;
  if (hasChildren || ['client','permanent','taxyear','section'].includes(node.type)) {
    const details = document.createElement('details');
    details.className = `node-${node.type}`;
    details.dataset.path = node.path;
    details.open = expanded.has(node.path) || node.type === 'client';
    details.addEventListener('toggle', () => {
      if (details.open) expanded.add(node.path); else expanded.delete(node.path);
    });

    const summary = document.createElement('summary');
    summary.dataset.path = node.path;
    summary.dataset.dropPath = node.path;
    summary.style.setProperty('--wails-drop-target', 'drop');
    summary.appendChild(nodeContent(node));
    if (['client','permanent','taxyear','section'].includes(node.type)) {
      summary.dataset.internalDropPath = node.path;
      summary.classList.add('internal-drop-destination');
      summary.addEventListener('dragenter', internalDragEnter);
      summary.addEventListener('dragover', internalDragOver);
      summary.addEventListener('dragleave', internalDragLeave);
      summary.addEventListener('drop', e => handleInternalDocumentDrop(e, node.path, summary));
    }
    summary.addEventListener('click', e => {
      // Let the native details marker toggle while also selecting this exact row.
      selectNode(node, summary);
    });
    if (node.type === 'client') {
      summary.addEventListener('contextmenu', e => {
        e.preventDefault();
        e.stopPropagation();
        selectNode(node, summary);
        showClientContextMenu(node, e.clientX, e.clientY);
      });
    }
    if (node.type === 'taxyear') {
      summary.addEventListener('contextmenu', e => {
        e.preventDefault();
        e.stopPropagation();
        selectNode(node, summary);
        showTaxYearContextMenu(node, e.clientX, e.clientY);
      });
    }
    if (node.type === 'section') {
      summary.addEventListener('contextmenu', e => {
        e.preventDefault();
        e.stopPropagation();
        selectNode(node, summary);
        showSectionContextMenu(node, e.clientX, e.clientY);
      });
    }
    if (selectedNode?.path === node.path) summary.classList.add('selected');
    details.appendChild(summary);

    const children = document.createElement('div');
    children.className = 'node-children';
    for (const child of (node.children || [])) children.appendChild(renderNode(child));
    details.appendChild(children);
    return details;
  }

  const row = document.createElement('div');
  row.className = 'tree-leaf';
  row.dataset.path = node.path;
  row.dataset.dropPath = node.path;
  row.style.setProperty('--wails-drop-target', 'drop');
  row.appendChild(nodeContent(node));
  row.addEventListener('click', event => handleFileSelection(event, node, row));
  if (node.type === 'file') {
    row.addEventListener('contextmenu', e => {
      e.preventDefault();
      e.stopPropagation();
      selectNode(node, row, selectedFilePaths.has(node.path));
      showDocumentContextMenu(node, e.clientX, e.clientY);
    });
  }

  // Internal document drag-and-drop. Documents can be moved between clients,
  // Permanent Folder, tax years, and sections. notes.txt remains protected.
  if (node.type === 'file' && node.name !== 'notes.txt') {
    row.draggable = true;
    row.dataset.dragFilePath = node.path;
    row.title = 'Drag this document to another client, tax year, or section';
    row.addEventListener('dragstart', e => startInternalDocumentDrag(e, node, row));
    row.addEventListener('dragend', finishInternalDocumentDrag);
  }

  if (selectedFilePaths.has(node.path) || selectedNode?.path === node.path) row.classList.add('selected');
  return row;
}

function nodeContent(node) {
  const wrap = document.createElement('span');
  wrap.className = 'node-content';
  const icon = document.createElement('span');
  icon.className = 'node-icon';
  if (node.type === 'client') {
    icon.classList.add('client-baby-icon');
    icon.innerHTML = `<svg viewBox="0 0 32 32" aria-label="baby face" role="img"><circle cx="16" cy="17" r="11" fill="#ffd25a" stroke="#d8a62b" stroke-width="1.2"/><path d="M13 7c0-3.2 5.6-3.1 5.6.2 0 1.7-1.6 2.5-3 2.1-1.3-.3-1.7-1.5-1.1-2.4" fill="none" stroke="#9b6427" stroke-width="1.8" stroke-linecap="round"/><circle cx="12" cy="16" r="1.3" fill="#4e3927"/><circle cx="20" cy="16" r="1.3" fill="#4e3927"/><path d="M12.5 21c1.8 2 5.2 2 7 0" fill="none" stroke="#a14f46" stroke-width="1.6" stroke-linecap="round"/><circle cx="8" cy="18.5" r="1.4" fill="#f4a7a1" opacity=".65"/><circle cx="24" cy="18.5" r="1.4" fill="#f4a7a1" opacity=".65"/></svg>`;
  } else {
    icon.textContent = iconFor(node);
  }
  const name = document.createElement('span');
  name.className = 'node-name';
  name.textContent = node.name;
  wrap.append(icon, name);
  return wrap;
}

function iconFor(node) {
  if (node.type === 'client') return '👶';
  if (node.type === 'permanent') return '📁';
  if (node.type === 'taxyear') return '🗓️';
  if (node.type === 'section') return '📁';
  const ext = (node.name.split('.').pop() || '').toLowerCase();
  if (ext === 'pdf') return '📕';
  if (['png','jpg','jpeg','gif','webp'].includes(ext)) return '🖼️';
  if (ext === 'csv' || ext === 'xls' || ext === 'xlsx') return '📊';
  if (ext === 'doc' || ext === 'docx' || ext === 'rtf') return '📘';
  if (['eml','msg','mbox','oft'].includes(ext)) return '✉️';
  return '📄';
}

function fileSelectionRange(paths, anchor, target) {
  const first = paths.indexOf(anchor), last = paths.indexOf(target);
  if (first < 0 || last < 0) return [target];
  return paths.slice(Math.min(first, last), Math.max(first, last) + 1);
}

function visibleClientFilePaths(node) {
  const client = treeData.find(c => node.path.startsWith(c.path + '/') || node.path.startsWith(c.path + '\\'));
  if (!client) return [node.path];
  return [...$('tree').querySelectorAll('.tree-leaf[data-path]')].filter(row => {
    const path = row.dataset.path;
    if (!(path.startsWith(client.path + '/') || path.startsWith(client.path + '\\'))) return false;
    for (let parent = row.parentElement; parent; parent = parent.parentElement) {
      if (parent.tagName === 'DETAILS' && !parent.open) return false;
    }
    return true;
  }).map(row => row.dataset.path);
}

function paintFileSelection() {
  $('tree').querySelectorAll('.tree-leaf[data-path]').forEach(row => {
    row.classList.toggle('selected', selectedFilePaths.has(row.dataset.path));
    row.setAttribute('aria-selected', String(selectedFilePaths.has(row.dataset.path)));
  });
}

function handleFileSelection(event, node, row) {
  if (event.shiftKey && fileSelectionAnchor) {
    event.preventDefault();
    selectedFilePaths = new Set(fileSelectionRange(visibleClientFilePaths(node), fileSelectionAnchor, node.path));
  } else {
    selectedFilePaths = new Set([node.path]);
    fileSelectionAnchor = node.path;
  }
  selectNode(node, row, true);
}

async function selectNode(node, element = null, preserveFileSelection = false) {
  if (!preserveFileSelection || node.type !== 'file') {
    selectedFilePaths = new Set(node.type === 'file' ? [node.path] : []);
    fileSelectionAnchor = node.type === 'file' ? node.path : null;
  }
  selectedNode = node;
  if (selectedElement) selectedElement.classList.remove('selected');
  selectedElement = element || document.querySelector(`[data-path="${cssEscape(node.path)}"] > summary, [data-path="${cssEscape(node.path)}"].tree-leaf`);
  if (selectedElement) selectedElement.classList.add('selected');
  // Keep the 5-digit client ID out of the normal tree/selection display.
  // It is available from right-click > Client Properties.
  paintFileSelection();
  $('selectionTitle').textContent = selectedFilePaths.size > 1 ? `${selectedFilePaths.size} files selected — ${node.name}` : node.name;
  await updateSelectionClientSummary(node);
  const shell = document.querySelector('.preview-shell');
  if (shell) { shell.dataset.dropPath = node.path; shell.style.setProperty('--wails-drop-target', 'drop'); }
  await showSelection(node);
}

async function showSelection(node) {
  const generation = sessionGeneration;
  currentPDF = null;
  pdfSearchResults = [];
  pdfSearchIndex = -1;
  if (node.type === 'client' || node.type === 'permanent' || (node.type === 'file' && node.name === 'notes.txt')) {
    const [notes, history] = await Promise.all([
      backend().GetNotesDocument(node.path),
      backend().GetAuditTrail(node.path),
    ]);
    if (generation !== sessionGeneration || !currentUser) return;
    renderNotes(node, notes, history || []);
    return;
  }
  if (node.type !== 'file') {
    renderFolderInfo(node);
    return;
  }
  const preview = await backend().GetPreview(node.path);
  if (generation !== sessionGeneration || !currentUser) return;
  await renderPreview(preview);
}

function renderNotes(node, notes, history) {
  $('preview').className = '';
  $('preview').innerHTML = `
    <div class="preview-header">
      <div class="preview-title">Client Notes — ${escapeHTML(clientNameFor(node.path))}</div>
      <button id="saveNotesBtn" class="primary">💾 Save Notes</button>
    </div>
    <div class="preview-body client-notes-view">
      <div class="notes-toolbar" aria-label="Notes formatting toolbar">
        <label>Font
          <select id="notesFont">
            <option value="Arial">Arial</option>
            <option value="Georgia">Georgia</option>
            <option value="Times New Roman">Times New Roman</option>
            <option value="Verdana">Verdana</option>
            <option value="Trebuchet MS">Trebuchet MS</option>
            <option value="Courier New">Courier New</option>
          </select>
        </label>
        <label>Color <input id="notesColor" type="color" value="#172a47" /></label>
        <button type="button" id="notesBold" title="Bold"><b>B</b></button>
        <button type="button" id="notesItalic" title="Italic"><i>I</i></button>
        <button type="button" id="notesBullets" title="Bulleted list">• Bullets</button>
        <button type="button" id="notesClearFormat" title="Clear formatting">Clear format</button>
      </div>
      <div id="notesEditor" class="notes-editor rich-notes-editor" contenteditable="true" spellcheck="true" aria-label="Client notes"></div>

      <section class="audit-section">
        <div class="audit-heading-row">
          <div>
            <h3>Audit Trail / Client History</h3>
            <p>Recent activity for this client is recorded locally.</p>
          </div>
          <button id="refreshAuditBtn" class="secondary">↻ Refresh</button>
        </div>
        <div id="auditTrail" class="audit-trail"></div>
      </section>
    </div>`;

  const editor = $('notesEditor');
  editor.addEventListener('paste', event => {
    event.preventDefault();
    const html = event.clipboardData?.getData('text/html');
    const text = event.clipboardData?.getData('text/plain') || '';
    document.execCommand('insertHTML', false, html ? sanitizeNotesHTML(html) : plainTextToNotesHTML(text));
  });
  // Native drops can insert active markup directly into a contenteditable element.
  editor.addEventListener('drop', event => event.preventDefault());
  notesSelectionRange = null;
  if (notes?.html) {
    editor.innerHTML = sanitizeNotesHTML(notes.html);
  } else {
    editor.innerHTML = plainTextToNotesHTML(notes?.text || '');
  }
  renderAuditTrail(history || []);

  const rememberNotesSelection = () => {
    const selection = window.getSelection();
    if (selection && selection.rangeCount && editor.contains(selection.anchorNode)) {
      notesSelectionRange = selection.getRangeAt(0).cloneRange();
    }
  };
  const focusEditor = () => {
    editor.focus();
    if (notesSelectionRange) {
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(notesSelectionRange);
    }
  };
  editor.addEventListener('mouseup', rememberNotesSelection);
  editor.addEventListener('keyup', rememberNotesSelection);
  editor.addEventListener('input', rememberNotesSelection);

  $('notesFont').addEventListener('change', e => {
    focusEditor();
    document.execCommand('fontName', false, e.target.value);
    rememberNotesSelection();
  });
  $('notesColor').addEventListener('input', e => {
    focusEditor();
    document.execCommand('foreColor', false, e.target.value);
    rememberNotesSelection();
  });
  $('notesBold').addEventListener('mousedown', e => e.preventDefault());
  $('notesBold').addEventListener('click', () => { focusEditor(); document.execCommand('bold', false, null); rememberNotesSelection(); });
  $('notesItalic').addEventListener('mousedown', e => e.preventDefault());
  $('notesItalic').addEventListener('click', () => { focusEditor(); document.execCommand('italic', false, null); rememberNotesSelection(); });
  $('notesBullets').addEventListener('mousedown', e => e.preventDefault());
  $('notesBullets').addEventListener('click', () => { focusEditor(); document.execCommand('insertUnorderedList', false, null); rememberNotesSelection(); });
  $('notesClearFormat').addEventListener('mousedown', e => e.preventDefault());
  $('notesClearFormat').addEventListener('click', () => { focusEditor(); document.execCommand('removeFormat', false, null); rememberNotesSelection(); });

  $('saveNotesBtn').addEventListener('click', () => call(async () => {
    const cleanHTML = sanitizeNotesHTML(editor.innerHTML);
    const plainText = notesEditorPlainText(editor);
    await backend().SaveNotesDocument(node.path, cleanHTML, plainText);
    editor.innerHTML = cleanHTML;
    const updatedHistory = await backend().GetAuditTrail(node.path);
    renderAuditTrail(updatedHistory || []);
    toast('Notes saved.');
  }));

  $('refreshAuditBtn').addEventListener('click', () => call(async () => {
    const updatedHistory = await backend().GetAuditTrail(node.path);
    renderAuditTrail(updatedHistory || []);
  }));
}

function renderAuditTrail(entries) {
  const root = $('auditTrail');
  if (!root) return;
  if (!entries?.length) {
    root.innerHTML = '<div class="audit-empty">No client history has been recorded yet.</div>';
    return;
  }
  root.innerHTML = entries.map(entry => `
    <div class="audit-entry">
      <div class="audit-time">${escapeHTML(formatAuditTime(entry.timestamp))}</div>
      <div class="audit-content">
        <div class="audit-action">${escapeHTML(entry.action || 'Activity')}</div>
        ${entry.details ? `<div class="audit-details">${escapeHTML(entry.details)}</div>` : ''}
      </div>
    </div>`).join('');
}

function formatAuditTime(value) {
  if (!value) return '';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString();
}

function plainTextToNotesHTML(text) {
  return escapeHTML(text || '').replace(/\r?\n/g, '<br>');
}

function sanitizeNotesHTML(input) {
  if (typeof input !== 'string' || input.length > 2 * 1024 * 1024) return '';
  const template = document.createElement('template');
  template.innerHTML = input;
  const output = document.createElement('div');
  const allowed = new Set(['DIV','P','BR','UL','OL','LI','SPAN','B','STRONG','I','EM','U','FONT']);
  const drop = new Set(['SCRIPT','STYLE','IFRAME','OBJECT','EMBED','SVG','MATH','TEMPLATE','NOSCRIPT','HEAD','TITLE','LINK','META','BASE']);
  const fonts = new Set(['arial','georgia','times new roman','verdana','trebuchet ms','courier new','sans-serif','serif','monospace']);
  const safeStyle = (name, value) => {
    switch (name) {
      case 'color': return /^(#[0-9a-f]{3}|#[0-9a-f]{6}|black|white|red|green|blue|gray|grey|yellow|orange|purple|transparent|rgb\(\s*[0-9]{1,3}\s*,\s*[0-9]{1,3}\s*,\s*[0-9]{1,3}\s*\))$/i.test(value);
      case 'font-family': return fonts.has(value.replace(/^['"]|['"]$/g, '').toLowerCase());
      case 'font-weight': return ['normal','bold','400','700'].includes(value);
      case 'font-style': return ['normal','italic'].includes(value);
      case 'text-decoration': return ['none','underline','line-through'].includes(value);
      default: return false;
    }
  };
  const copy = (source, destination, depth) => {
    if (depth > 128) throw new Error('Formatted notes nesting is too deep.');
    for (const child of source.childNodes) {
      if (child.nodeType === Node.TEXT_NODE) {
        destination.appendChild(document.createTextNode(child.textContent));
      } else if (child.nodeType === Node.ELEMENT_NODE) {
        if (child.namespaceURI !== 'http://www.w3.org/1999/xhtml' || drop.has(child.tagName)) continue;
        let target = destination;
        if (allowed.has(child.tagName)) {
          target = document.createElement(child.tagName.toLowerCase());
          const style = [];
          for (const declaration of (child.getAttribute('style') || '').split(';')) {
            const colon = declaration.indexOf(':');
            if (colon < 0) continue;
            const name = declaration.slice(0, colon).trim().toLowerCase();
            const value = declaration.slice(colon + 1).trim();
            if (safeStyle(name, value)) style.push(`${name}: ${value}`);
          }
          if (style.length) target.setAttribute('style', style.join('; '));
          if (child.tagName === 'FONT') {
            for (const [attr, prop] of [['color','color'],['face','font-family']]) {
              const value = child.getAttribute(attr) || '';
              if (safeStyle(prop, value)) target.setAttribute(attr, value);
            }
          }
          destination.appendChild(target);
        }
        copy(child, target, depth + 1);
      }
    }
  };
  copy(template.content, output, 0);
  return output.innerHTML;
}

function notesEditorPlainText(editor) {
  const clone = editor.cloneNode(true);
  clone.querySelectorAll('br').forEach(br => br.replaceWith(document.createTextNode('\n')));
  clone.querySelectorAll('li').forEach(li => {
    li.insertBefore(document.createTextNode('• '), li.firstChild);
    li.appendChild(document.createTextNode('\n'));
  });
  clone.querySelectorAll('div, p').forEach(el => el.appendChild(document.createTextNode('\n')));
  return (clone.textContent || '').replace(/\n{3,}/g, '\n\n').trimEnd();
}

function renderFolderInfo(node) {
  $('preview').className = '';
  $('preview').innerHTML = `
    <div class="preview-header"><div class="preview-title">${escapeHTML(node.name)}</div></div>
    <div class="preview-body">
      <h2>${escapeHTML(node.name)}</h2>
      <p>Select a document beneath this folder to preview it, or use <b>Upload Files</b> above.</p>
    </div>`;
}

async function renderPreview(p) {
  $('preview').className = '';
  const openBtn = `<button id="openFileBtn">↗ Open File</button>`;
  if (p.kind === 'pdf') {
    currentPDF = p;
    pdfPage = 1;
    pdfZoom = 1.0;
    pdfRotation = 0;
    const pdfHeaderActions = `
      <div class="preview-header-actions">
        <button id="rotatePdfBtn" title="Rotate PDF preview 90° clockwise">↻ Rotate PDF</button>
        ${openBtn}
      </div>`;
    $('preview').innerHTML = `
      <div class="preview-header"><div class="preview-title">${escapeHTML(p.name)}</div>${pdfHeaderActions}</div>
      <div class="pdf-searchbar">
        <input id="pdfSearchInput" type="search" placeholder="Search this PDF… (Ctrl+F)" aria-label="Search this PDF" />
        <button id="pdfSearchBtn" class="primary">🔍 Search</button>
        <button id="pdfMatchPrev" title="Previous match" disabled>▲</button>
        <button id="pdfMatchNext" title="Next match" disabled>▼</button>
        <span id="pdfSearchStatus">Ctrl+F to search</span>
      </div>
      <div id="pdfSearchSnippet" class="pdf-search-snippet hidden"></div>
      <div class="pdf-controls">
        <div class="pdf-page-controls">
          <button id="pdfPrev">◀ Previous</button>
          <strong id="pdfLabel">Page 1 of ${p.pageCount}</strong>
          <button id="pdfNext">Next ▶</button>
        </div>
        <div class="pdf-zoom-controls">
          <button id="pdfZoomOut" title="Zoom out">−</button>
          <strong id="pdfZoomLabel">100%</strong>
          <button id="pdfZoomIn" title="Zoom in">+</button>
          <button id="pdfZoomReset" title="Reset zoom">Reset</button>
        </div>
      </div>
      <div class="preview-body pdf-preview-body">
        <div id="pdfLoading">Rendering PDF page…</div>
        <div id="pdfRotateStage" class="pdf-rotate-stage">
          <div id="pdfPageWrap" class="pdf-page-wrap">
            <img id="pdfImage" class="pdf-page" alt="PDF page preview" />
            <div id="pdfSearchHighlights" class="pdf-search-highlights" aria-hidden="true"></div>
          </div>
        </div>
      </div>`;
    wireOpen(p.path);
    $('rotatePdfBtn').addEventListener('click', rotateCurrentPDF);
    $('pdfPrev').addEventListener('click', () => changePDFPage(-1));
    $('pdfNext').addEventListener('click', () => changePDFPage(1));
    $('pdfZoomOut').addEventListener('click', () => changePDFZoom(-0.1));
    $('pdfZoomIn').addEventListener('click', () => changePDFZoom(0.1));
    $('pdfZoomReset').addEventListener('click', () => setPDFZoom(1.0));
    $('pdfSearchBtn').addEventListener('click', searchCurrentPDF);
    $('pdfSearchInput').addEventListener('keydown', e => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const typed = e.currentTarget.value.trim();
        if (typed && typed === pdfSearchQuery && pdfSearchResults.length) {
          movePDFSearchMatch(e.shiftKey ? -1 : 1);
        } else {
          searchCurrentPDF();
        }
      } else if (e.key === 'Escape') {
        e.preventDefault();
        e.currentTarget.value = '';
        clearPDFSearch();
        e.currentTarget.blur();
      }
    });
    $('pdfSearchInput').addEventListener('input', e => {
      clearTimeout(pdfSearchTimer);
      const value = e.target.value.trim();
      if (!value) {
        clearPDFSearch();
        return;
      }
      // Windows-style find-as-you-type. A short debounce prevents launching
      // multiple Poppler searches while the user is still typing.
      pdfSearchTimer = setTimeout(() => searchCurrentPDF(), 160);
    });
    $('pdfMatchPrev').addEventListener('click', () => movePDFSearchMatch(-1));
    $('pdfMatchNext').addEventListener('click', () => movePDFSearchMatch(1));
    await loadPDFPage();
    return;
  }
  if (p.kind === 'image') {
    $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)}</div>${openBtn}</div><div class="preview-body"><img id="documentImagePreview" class="image-preview" alt="Document preview" /></div>`;
    const image = $('documentImagePreview');
    if (/^data:image\/(png|jpeg|gif|webp|bmp);base64,[A-Za-z0-9+/]*={0,2}$/.test(p.imageData || '')) image.src = p.imageData;
    else image.alt = 'Unsupported image preview';
    wireOpen(p.path); return;
  }
  if (p.kind === 'csv') {
    const table = p.rows.map(r => `<tr>${r.map(c => `<td>${escapeHTML(c)}</td>`).join('')}</tr>`).join('');
    $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)}</div>${openBtn}</div><div class="preview-body csv-wrap"><table>${table}</table></div>`;
    wireOpen(p.path); return;
  }
  if (p.kind === 'spreadsheet') {
    const table = (p.rows || []).map((r, i) => `<tr>${r.map(c => `<${i === 0 ? 'th' : 'td'}>${escapeHTML(c)}</${i === 0 ? 'th' : 'td'}>`).join('')}</tr>`).join('');
    $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)} — Excel Preview</div>${openBtn}</div><div class="preview-body csv-wrap"><table>${table}</table><p class="preview-note">Previewing the first worksheet (up to 200 rows). Open the file for full Excel features.</p></div>`;
    wireOpen(p.path); return;
  }
  if (p.kind === 'email') {
    $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)} — Email Preview</div>${openBtn}</div><div class="preview-body"><div class="email-preview">${escapeHTML(p.text || '')}</div></div>`;
    wireOpen(p.path); return;
  }
  if (p.kind === 'text') {
    $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)}</div>${openBtn}</div><div class="preview-body"><div class="text-preview">${escapeHTML(p.text || '')}</div></div>`;
    wireOpen(p.path); return;
  }
  $('preview').innerHTML = `<div class="preview-header"><div class="preview-title">${escapeHTML(p.name)}</div>${openBtn}</div><div class="preview-body"><h3>Preview unavailable</h3><p>${escapeHTML(p.message || 'Open this file in its desktop application.')}</p></div>`;
  wireOpen(p.path);
}

async function loadPDFPage() {
  if (!currentPDF) return;
  const generation = sessionGeneration;
  const pdf = currentPDF;
  const requestedPage = pdfPage;
  $('pdfLoading').style.display = 'block';
  $('pdfPageWrap').style.display = 'none';
  try {
    const data = await backend().RenderPDFPage(pdf.path, requestedPage);
    if (generation !== sessionGeneration || currentPDF !== pdf || !currentUser) return;
    const img = $('pdfImage');
    await new Promise((resolve, reject) => {
      img.onload = () => resolve();
      img.onerror = () => reject(new Error('Could not display the rendered PDF page.'));
      img.src = data;
    });
    if (generation !== sessionGeneration || currentPDF !== pdf || !currentUser) return;
    $('pdfPageWrap').style.display = 'inline-block';
    applyPDFZoom();
    $('pdfLoading').style.display = 'none';
    $('pdfLabel').textContent = `Page ${pdfPage} of ${currentPDF.pageCount}`;
    $('pdfPrev').disabled = pdfPage <= 1;
    $('pdfNext').disabled = pdfPage >= currentPDF.pageCount;
    updatePDFHighlights();
  } catch (e) { showError(e); }
}

function applyPDFZoom() {
  const img = $('pdfImage');
  const wrap = $('pdfPageWrap');
  const stage = $('pdfRotateStage');
  const body = document.querySelector('.pdf-preview-body');
  if (!img || !wrap || !stage || !body || !img.naturalWidth || !img.naturalHeight) return;

  const fitWidth = Math.max(320, Math.min(img.naturalWidth, body.clientWidth - 42));
  const width = Math.round(fitWidth * pdfZoom);
  const height = Math.max(1, Math.round(width * (img.naturalHeight / img.naturalWidth)));

  img.style.width = `${width}px`;
  img.style.height = `${height}px`;
  img.style.maxWidth = 'none';
  wrap.style.width = `${width}px`;
  wrap.style.height = `${height}px`;
  wrap.style.transformOrigin = 'top left';

  const rotation = ((pdfRotation % 360) + 360) % 360;
  if (rotation === 90) {
    wrap.style.transform = `translateX(${height}px) rotate(90deg)`;
    stage.style.width = `${height}px`;
    stage.style.height = `${width}px`;
  } else if (rotation === 180) {
    wrap.style.transform = `translate(${width}px, ${height}px) rotate(180deg)`;
    stage.style.width = `${width}px`;
    stage.style.height = `${height}px`;
  } else if (rotation === 270) {
    wrap.style.transform = `translateY(${width}px) rotate(270deg)`;
    stage.style.width = `${height}px`;
    stage.style.height = `${width}px`;
  } else {
    wrap.style.transform = 'none';
    stage.style.width = `${width}px`;
    stage.style.height = `${height}px`;
  }

  if ($('pdfZoomLabel')) $('pdfZoomLabel').textContent = `${Math.round(pdfZoom * 100)}%`;
  if ($('rotatePdfBtn')) {
    $('rotatePdfBtn').title = `Rotate PDF preview 90° clockwise (currently ${rotation}°)`;
  }
  updatePDFHighlights();
}

function rotateCurrentPDF() {
  if (!currentPDF) return;
  pdfRotation = (pdfRotation + 90) % 360;
  applyPDFZoom();
}

function setPDFZoom(value) {
  pdfZoom = Math.max(0.5, Math.min(3.0, Math.round(value * 10) / 10));
  applyPDFZoom();
}

function changePDFZoom(delta) {
  setPDFZoom(pdfZoom + delta);
}

async function changePDFPage(delta) {
  if (!currentPDF) return;
  const next = Math.max(1, Math.min(currentPDF.pageCount, pdfPage + delta));
  if (next === pdfPage) return;
  pdfPage = next;
  await loadPDFPage();
}

async function searchCurrentPDF() {
  if (!currentPDF) return;
  const generation = sessionGeneration;
  const pdf = currentPDF;
  const input = $('pdfSearchInput');
  const query = input?.value.trim() || '';
  if (!query) { clearPDFSearch(); input?.focus(); return; }

  clearTimeout(pdfSearchTimer);
  pdfSearchQuery = query;
  $('pdfSearchStatus').textContent = 'Searching…';
  $('pdfSearchBtn').disabled = true;
  try {
    const response = await backend().SearchPDF(pdf.path, query);
    if (generation !== sessionGeneration || currentPDF !== pdf || !currentUser) return;
    // Ignore an old result if the user typed something new while the backend was searching.
    if (($('pdfSearchInput')?.value.trim() || '') !== query) return;

    pdfSearchResults = response.results || [];
    pdfSearchIndex = pdfSearchResults.length ? 0 : -1;

    if (!response.searchable) {
      $('pdfSearchStatus').textContent = 'Not searchable';
      showPDFSearchMessage(response.message || 'This PDF has no searchable text.');
      updatePDFHighlights();
      return;
    }

    if (!response.total) {
      $('pdfSearchStatus').textContent = 'No matches';
      showPDFSearchMessage(response.message || `No match for “${query}”.`);
      $('pdfMatchPrev').disabled = true;
      $('pdfMatchNext').disabled = true;
      updatePDFHighlights();
      return;
    }

    updatePDFSearchMatch(response.total);
    await goToPDFSearchMatch();
  } catch (e) {
    showError(e);
    $('pdfSearchStatus').textContent = 'Search failed';
  } finally {
    if ($('pdfSearchBtn')) $('pdfSearchBtn').disabled = false;
  }
}

function clearPDFSearch() {
  clearTimeout(pdfSearchTimer);
  pdfSearchResults = [];
  pdfSearchIndex = -1;
  pdfSearchQuery = '';
  if ($('pdfSearchStatus')) $('pdfSearchStatus').textContent = 'Ctrl+F to search';
  if ($('pdfMatchPrev')) $('pdfMatchPrev').disabled = true;
  if ($('pdfMatchNext')) $('pdfMatchNext').disabled = true;
  if ($('pdfSearchSnippet')) $('pdfSearchSnippet').classList.add('hidden');
  updatePDFHighlights();
}

function showPDFSearchMessage(message) {
  const snippet = $('pdfSearchSnippet');
  if (!snippet) return;
  snippet.textContent = message;
  snippet.classList.remove('hidden');
}

function updatePDFSearchMatch(totalOverride = null) {
  const total = totalOverride ?? pdfSearchResults.length;
  const has = pdfSearchIndex >= 0 && pdfSearchResults.length > 0;
  $('pdfSearchStatus').textContent = has ? `Match ${pdfSearchIndex + 1} of ${total}` : 'No matches';
  $('pdfMatchPrev').disabled = !has || pdfSearchResults.length < 2;
  $('pdfMatchNext').disabled = !has || pdfSearchResults.length < 2;
  const snippet = $('pdfSearchSnippet');
  if (has) {
    const match = pdfSearchResults[pdfSearchIndex];
    snippet.innerHTML = `<strong>Page ${match.page}</strong> — ${highlightSearchText(match.snippet || '', pdfSearchQuery)}`;
    snippet.classList.remove('hidden');
  } else {
    snippet.classList.add('hidden');
  }
}

function highlightSearchText(text, query) {
  let html = escapeHTML(text);
  const terms = query.trim().split(/\s+/).filter(Boolean).sort((a,b) => b.length - a.length);
  for (const term of terms) {
    const safe = escapeHTML(term).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    if (!safe) continue;
    html = html.replace(new RegExp(`(${safe})`, 'gi'), '<mark>$1</mark>');
  }
  return html;
}

function updatePDFHighlights() {
  const layer = $('pdfSearchHighlights');
  if (!layer) return;
  layer.innerHTML = '';

  if (!pdfSearchResults.length || !currentPDF) return;

  pdfSearchResults.forEach((match, index) => {
    if (match.page !== pdfPage || !match.pageWidth || !match.pageHeight) return;

    const padX = 1.8;
    const padY = 1.2;
    const left = Math.max(0, ((match.x - padX) / match.pageWidth) * 100);
    const top = Math.max(0, ((match.y - padY) / match.pageHeight) * 100);
    const width = Math.min(100 - left, ((match.width + padX * 2) / match.pageWidth) * 100);
    const height = Math.min(100 - top, ((match.height + padY * 2) / match.pageHeight) * 100);

    const mark = document.createElement('div');
    mark.className = `pdf-search-highlight${index === pdfSearchIndex ? ' active' : ''}`;
    mark.style.left = `${left}%`;
    mark.style.top = `${top}%`;
    mark.style.width = `${width}%`;
    mark.style.height = `${height}%`;
    layer.appendChild(mark);
  });
}

async function goToPDFSearchMatch() {
  if (pdfSearchIndex < 0 || !pdfSearchResults.length) return;
  const match = pdfSearchResults[pdfSearchIndex];
  pdfPage = Math.max(1, Math.min(currentPDF.pageCount, match.page));
  updatePDFSearchMatch();
  await loadPDFPage();
}

async function movePDFSearchMatch(delta) {
  if (!pdfSearchResults.length) return;
  pdfSearchIndex = (pdfSearchIndex + delta + pdfSearchResults.length) % pdfSearchResults.length;
  await goToPDFSearchMatch();
}

function wireOpen(path) {
  $('openFileBtn')?.addEventListener('click', () => call(async () => backend().OpenFile(path)));
}

function positionContextMenu(menu, x, y) {
  hideAllContextMenus();
  menu.classList.remove('hidden');
  const width = menu.offsetWidth || 230;
  const height = menu.offsetHeight || 170;
  const left = Math.min(x, window.innerWidth - width - 8);
  const top = Math.min(y, window.innerHeight - height - 8);
  menu.style.left = `${Math.max(8, left)}px`;
  menu.style.top = `${Math.max(8, top)}px`;
}

function showClientContextMenu(client, x, y) {
  contextClient = client;
  positionContextMenu($('clientContextMenu'), x, y);
}

function showTaxYearContextMenu(year, x, y) {
  contextTaxYear = year;
  positionContextMenu($('taxYearContextMenu'), x, y);
}

function showSectionContextMenu(section, x, y) {
  contextSection = section;
  positionContextMenu($('sectionContextMenu'), x, y);
}

function showDocumentContextMenu(doc, x, y) {
  contextDocument = doc;
  const editBtn = $('documentContextMenu').querySelector('[data-document-action="edit"]');
  if (editBtn) editBtn.disabled = doc.name === 'notes.txt';
  const deleteBtn = $('documentContextMenu').querySelector('[data-document-action="delete"]');
  if (deleteBtn) deleteBtn.disabled = doc.name === 'notes.txt';
  positionContextMenu($('documentContextMenu'), x, y);
}

function hideAllContextMenus() {
  document.querySelectorAll('.context-menu').forEach(menu => menu.classList.add('hidden'));
}

function hideClientContextMenu() { hideAllContextMenus(); }

async function showClientProperties(client) {
  try {
    const props = await backend().GetClientProperties(client.path);
    $('propertyClientName').textContent = props.name || client.name;
    $('propertyClientID').textContent = props.clientId || client.clientId || '—';
    $('propertyReturnType').textContent = props.returnType || '—';
    $('propertyAddress').textContent = props.address || '—';
    $('propertyPhone').textContent = props.phone || '—';
    $('propertyEmail').textContent = props.email || '—';
    $('propertyTaxID').textContent = props.taxId || '—';
    $('propertyClientPath').textContent = props.path || client.path;
    $('propertiesBackdrop').classList.remove('hidden');
  } catch (e) {
    showError(e);
  }
}

function hideClientProperties() {
  $('propertiesBackdrop').classList.add('hidden');
}

async function showDocumentProperties(doc) {
  try {
    const props = await backend().GetDocumentProperties(doc.path);
    documentPropertiesPath = props.path || doc.path;
    $('docPropertyName').textContent = props.name || doc.name;
    $('docPropertyClient').textContent = props.clientName ? `${props.clientName} (${props.clientId || ''})` : '—';
    $('docPropertyLocation').textContent = props.location || '—';
    $('docPropertySize').textContent = formatFileSize(props.size || 0);
    $('docPropertyModified').textContent = props.modified ? new Date(props.modified).toLocaleString() : '—';
    $('documentDescriptionInput').value = props.description || '';
    $('documentPropertiesBackdrop').classList.remove('hidden');
    setTimeout(() => $('documentDescriptionInput').focus(), 20);
  } catch (e) { showError(e); }
}

function hideDocumentProperties() {
  documentPropertiesPath = null;
  $('documentPropertiesBackdrop').classList.add('hidden');
}

async function saveDocumentProperties() {
  if (!documentPropertiesPath) return;
  try {
    const description = $('documentDescriptionInput').value.trim();
    await backend().SaveDocumentDescription(documentPropertiesPath, description);
    hideDocumentProperties();
    toast('Document description saved.');
  } catch (e) { showError(e); }
}

function formatFileSize(bytes) {
  const n = Number(bytes) || 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function showGlobalFind() {
  $('globalFindBackdrop').classList.remove('hidden');
  $('globalFindStatus').textContent = 'Type a word or phrase to search.';
  $('globalFindResults').innerHTML = '';
  setTimeout(() => { $('globalFindInput').focus(); $('globalFindInput').select(); }, 20);
}

function hideGlobalFind() {
  $('globalFindBackdrop').classList.add('hidden');
}

async function runGlobalFind() {
  const query = $('globalFindInput').value.trim();
  const resultsEl = $('globalFindResults');
  if (!query) {
    $('globalFindStatus').textContent = 'Type a word or phrase to search.';
    resultsEl.innerHTML = '';
    return;
  }
  $('globalFindStatus').textContent = 'Searching…';
  try {
    const results = await backend().GlobalFind(query);
    $('globalFindStatus').textContent = `${results.length} result${results.length === 1 ? '' : 's'} found`;
    if (!results.length) {
      resultsEl.innerHTML = '<div class="global-find-empty">No matching documents.</div>';
      return;
    }
    resultsEl.innerHTML = results.map(r => `
      <button type="button" class="global-find-result" data-global-path="${escapeHTML(r.path)}">
        <span class="global-find-result-title">${escapeHTML(r.name)}</span>
        <span class="global-find-result-client">👶 ${escapeHTML(r.clientName)} • ${escapeHTML(r.location || '')}</span>
        ${r.description ? `<span class="global-find-result-description">${escapeHTML(r.description)}</span>` : ''}
        <span class="global-find-result-match">Matched: ${escapeHTML(r.match || 'Document')}</span>
      </button>`).join('');
  } catch (e) {
    $('globalFindStatus').textContent = 'Search failed';
    showError(e);
  }
}

async function doAction(action) {
  await call(async () => {
    switch (action) {
      case 'global-find': {
        showGlobalFind();
        break;
      }
      case 'add-client': {
        const info = await askClientDetails();
        if (!info) return;
        const path = await backend().AddClient(info.name, info.address, info.phone, info.email, info.taxId, info.returnType);
        expanded.add(path);
        expanded.add(`${path}/Permanent Folder`);
        await refreshTree(path);
        break;
      }
      case 'edit-client': {
        const client = selectedClient(); if (!client) throw new Error('Select a client first.');
        const props = await backend().GetClientProperties(client.path);
        const info = await askClientDetails(props, 'edit');
        if (!info) return;
        let clientPath = client.path;
        if (info.name && info.name.trim() !== client.name.trim()) {
          clientPath = await backend().RenameClient(client.path, info.name);
        }
        await backend().UpdateClientProfile(clientPath, info.address, info.phone, info.email, info.taxId, info.returnType);
        await refreshTree(clientPath);
        toast('Client profile updated.');
        break;
      }
      case 'rename-client': {
        const client = selectedClient(); if (!client) throw new Error('Select a client first.');
        const name = await ask('Rename Client', 'Enter the new client or business name:', client.name);
        if (!name) return;
        const path = await backend().RenameClient(client.path, name);
        await refreshTree(path);
        break;
      }
      case 'delete-client': {
        const client = selectedClient(); if (!client) throw new Error('Select a client first.');
        if (!confirm(`Delete ${client.name} and every document inside it?`)) return;
        await backend().DeleteClient(client.path); selectedNode = null; await refreshTree(); clearPreview(); break;
      }
      case 'add-year': {
        const client = selectedClient(); if (!client) throw new Error('Select a client first.');
        const year = await ask('Add Tax Year', 'Enter a 4-digit tax year:', new Date().getFullYear().toString());
        if (!year) return;
        const path = await backend().AddTaxYear(client.path, year);
        expanded.add(client.path); expanded.add(path); await refreshTree(path); break;
      }
      case 'edit-year': {
        if (selectedNode?.type !== 'taxyear') throw new Error('Select a tax year first.');
        const year = await ask('Edit Tax Year', 'Enter the new 4-digit tax year:', selectedNode.name);
        if (!year) return;
        const path = await backend().RenameTaxYear(selectedNode.path, year); await refreshTree(path); break;
      }
      case 'delete-year': {
        if (selectedNode?.type !== 'taxyear') throw new Error('Select a tax year first.');
        if (!confirm(`Delete tax year ${selectedNode.name} and everything inside it?`)) return;
        await backend().DeleteTaxYear(selectedNode.path); selectedNode = null; await refreshTree(); clearPreview(); break;
      }
      case 'add-section': {
        if (!selectedNode || !['client','taxyear','section'].includes(selectedNode.type)) throw new Error('Select a client, tax year, or section first.');
        const selectedParent = selectedNode.path;
        const name = await ask('Add Section', 'Enter the section name:', 'Source Documents');
        if (!name) return;
        const path = await backend().AddSection(selectedParent, name);
        expandAncestors(path); expanded.add(path); clearPreview();
        await refreshTree(path); break;
      }
      case 'edit-section': {
        if (selectedNode?.type !== 'section') throw new Error('Select a section first.');
        const name = await ask('Edit Section', 'Enter the new section name:', selectedNode.name);
        if (!name) return;
        const path = await backend().RenameSection(selectedNode.path, name); await refreshTree(path); break;
      }
      case 'delete-section': {
        if (selectedNode?.type !== 'section') throw new Error('Select a section first.');
        if (!confirm(`Delete empty section ${selectedNode.name}? Files and subsections must be removed first.`)) return;
        await backend().DeleteSection(selectedNode.path); selectedNode = null; await refreshTree(); clearPreview(); break;
      }
      case 'upload': {
        if (!selectedNode) throw new Error('Select a client, Permanent Folder, tax year, or section first.');
        const files = await backend().UploadFiles(selectedNode.path);
        if (files && files.length) {
          const last = files[files.length - 1];
          expandAncestors(last); await refreshTree(last); toast(`${files.length} file(s) uploaded.`);
        }
        break;
      }
      case 'rename-file': {
        if (selectedNode?.type !== 'file' || selectedNode.name === 'notes.txt') throw new Error('Select a file other than notes.txt first.');
        const document = selectedNode;
        const name = await ask('Edit Document Name', 'Enter the new file name, including its extension:', document.name);
        if (!name) return;
        const path = await backend().RenameFile(document.path, name);
        selectedFilePaths.clear(); fileSelectionAnchor = path;
        selectedFilePaths.add(path); await refreshTree(path); break;
      }
      case 'delete-file': {
        if (selectedNode?.type !== 'file' || selectedNode.name === 'notes.txt') throw new Error('Select a file other than notes.txt first.');
        const files = [...selectedFilePaths].map(path => findNodeByPath(path, treeData)).filter(node => node?.type === 'file');
        if (!files.length) files.push(selectedNode);
        if (files.some(node => node.name === 'notes.txt')) throw new Error('Client notes.txt cannot be deleted. Deselect it first.');
        if (!confirm(files.length > 1 ? `Delete all ${files.length} selected files?` : `Delete ${files[0].name}?`)) return;
        try {
          for (const file of files) await backend().DeleteFile(file.path);
        } finally {
          selectedFilePaths.clear(); fileSelectionAnchor = null;
          selectedNode = null; await refreshTree(); clearPreview();
        }
        break;
      }
    }
  });
}

function selectedClient() {
  if (!selectedNode) return null;
  for (const c of treeData) {
    if (selectedNode.path === c.path || selectedNode.path.startsWith(c.path + '/') || selectedNode.path.startsWith(c.path + '\\')) return c;
  }
  return null;
}

function clientNameFor(path) {
  for (const c of treeData) if (path === c.path || path.startsWith(c.path + '/') || path.startsWith(c.path + '\\')) return c.name;
  return 'Client';
}

function findNodeByPath(path, nodes) {
  for (const n of nodes) {
    if (n.path === path) return n;
    const found = findNodeByPath(path, n.children || []);
    if (found) return found;
  }
  return null;
}

function filterNodes(nodes, q) {
  const out = [];
  for (const n of nodes) {
    const kids = filterNodes(n.children || [], q);
    if (n.name.toLowerCase().includes(q) || (n.clientId || '').includes(q) || kids.length) out.push({...n, children: kids});
  }
  return out;
}

function expandAncestors(path) {
  for (const c of treeData) {
    if (path === c.path || path.startsWith(c.path + '/') || path.startsWith(c.path + '\\')) {
      expanded.add(c.path);
      markExpanded(c, path);
    }
  }
}

function markExpanded(node, path) {
  for (const child of node.children || []) {
    if (path === child.path || path.startsWith(child.path + '/') || path.startsWith(child.path + '\\')) {
      if (child.type !== 'file') expanded.add(child.path);
      markExpanded(child, path);
    }
  }
}

function scrollToPath(path) {
  setTimeout(() => {
    const els = [...document.querySelectorAll('[data-path]')];
    const el = els.find(x => x.dataset.path === path);
    el?.scrollIntoView({block:'nearest'});
  }, 30);
}

function startInternalDocumentDrag(e, node, element) {
  internalDraggedPath = node.path;
  internalDraggedElement = element;
  element.classList.add('internal-dragging');
  document.body.classList.add('internal-document-drag-active');

  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('application/x-babyfilecab-document', node.path);
    e.dataTransfer.setData('text/plain', node.path);
    try {
      const ghost = element.cloneNode(true);
      ghost.classList.add('internal-drag-ghost');
      ghost.style.width = `${Math.max(240, element.getBoundingClientRect().width)}px`;
      document.body.appendChild(ghost);
      e.dataTransfer.setDragImage(ghost, 18, 18);
      setTimeout(() => ghost.remove(), 0);
    } catch (_) {}
  }
}

function finishInternalDocumentDrag() {
  internalDraggedElement?.classList.remove('internal-dragging');
  internalDraggedElement = null;
  internalDraggedPath = '';
  document.body.classList.remove('internal-document-drag-active');
  document.querySelectorAll('.internal-drop-active').forEach(el => el.classList.remove('internal-drop-active'));
}

function internalDragEnter(e) {
  if (!internalDraggedPath) return;
  e.preventDefault();
  e.currentTarget.classList.add('internal-drop-active');
}

function internalDragOver(e) {
  if (!internalDraggedPath) return;
  e.preventDefault();
  e.stopPropagation();
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  e.currentTarget.classList.add('internal-drop-active');
}

function internalDragLeave(e) {
  if (!internalDraggedPath) return;
  const current = e.currentTarget;
  if (e.relatedTarget && current.contains(e.relatedTarget)) return;
  current.classList.remove('internal-drop-active');
}

async function handleInternalDocumentDrop(e, targetPath, targetElement) {
  if (!internalDraggedPath) return;
  e.preventDefault();
  e.stopPropagation();
  targetElement?.classList.remove('internal-drop-active');

  const sourcePath = internalDraggedPath;
  try {
    const movedPath = await backend().MoveDocument(sourcePath, targetPath);
    finishInternalDocumentDrag();
    expandAncestors(movedPath);
    await refreshTree(movedPath);
    toast('Document moved.');
  } catch (err) {
    finishInternalDocumentDrag();
    showError(err);
  }
}

function wireNativeFileDrop() {
  if (!window.runtime || typeof window.runtime.OnFileDrop !== 'function') {
    console.warn('Wails file-drop runtime is not available.');
    return;
  }
  window.runtime.OnFileDrop(async (x, y, paths) => {
    try {
      if (!paths || !paths.length) return;
      let el = document.elementFromPoint(x, y);
      let target = el?.closest?.('[data-drop-path]');
      let targetPath = target?.dataset?.dropPath || selectedNode?.path || '';
      if (!targetPath) throw new Error('Select a client or folder before dropping files.');
      const copied = await backend().ImportDroppedFiles(targetPath, paths);
      if (copied && copied.length) {
        const last = copied[copied.length - 1];
        expandAncestors(last);
        await refreshTree(last);
        toast(`${copied.length} file(s) added by drag and drop.`);
      }
    } catch (e) {
      showError(e);
    }
  }, true);
}

function closeAppMenus() {
  document.querySelectorAll('.menu-dropdown').forEach(menu => menu.classList.add('hidden'));
  document.querySelectorAll('.menu-trigger').forEach(btn => btn.classList.remove('active'));
}

function toggleAppMenu(id) {
  const target = $(id);
  const wasOpen = target && !target.classList.contains('hidden');
  closeAppMenus();
  if (!target || wasOpen) return;
  target.classList.remove('hidden');
  document.querySelector(`.menu-trigger[data-menu="${id}"]`)?.classList.add('active');
}

async function handleMenuAction(action) {
  try {
    switch (action) {
      case 'new-client':
        await doAction('add-client');
        break;
      case 'edit-client-picker': {
        const client = await chooseClient('Edit Client', 'Choose the client whose contact and tax profile you want to edit.');
        if (!client) return;
        selectNode(client);
        await doAction('edit-client');
        break;
      }
      case 'rename-client-picker': {
        const client = await chooseClient('Rename Client', 'Choose the client you want to rename.');
        if (!client) return;
        selectNode(client);
        await doAction('rename-client');
        break;
      }
      case 'delete-client-picker': {
        const client = await chooseClient('Delete Client', 'Choose the client you want to permanently delete.');
        if (!client) return;
        selectNode(client);
        await doAction('delete-client');
        break;
      }
      case 'backup-client': {
        const client = await chooseClient('Back Up Client Files', 'Choose one client from the full active client list. BabyFileCab will create an encrypted .bfcbackup archive of that client.');
        if (!client) return;
        const dest = await backend().BackupClient(client.path);
        if (dest) toast(`Backup created: ${dest}`);
        break;
      }
      case 'restore-client': {
        const client = await chooseClient('Restore Client', 'Choose the client that should receive the selected encrypted .bfcbackup or legacy ZIP backup.');
        if (!client) return;
        if (!confirm(`Restore a backup into ${client.name}? Files from the backup may replace files with the same name.`)) return;
        const source = await backend().RestoreClient(client.path);
        if (source) {
          await refreshTree(client.path);
          toast(`Client restored from ${source}`);
        }
        break;
      }
      case 'archive-client': {
        const client = await chooseClient('Archive Client', 'Choose the client you want to move out of the active Work Space.');
        if (!client) return;
        if (!confirm(`Archive ${client.name}? The client will be moved to BabyFileCab's local Archive folder.`)) return;
        await backend().ArchiveClient(client.path);
        if (selectedNode && (selectedNode.path === client.path || selectedNode.path.startsWith(client.path + '/'))) {
          selectedNode = null;
          clearPreview();
        }
        await refreshTree();
        toast(`${client.name} archived.`);
        break;
      }
      case 'client-communications':
        await showClientCommunications();
        break;
      case 'sign-out':
        await signOut();
        break;
      case 'exit':
        if (confirm('Exit BabyFileCab?')) backend().ExitBabyFileCab();
        break;
      case 'search-clients': {
        const client = await chooseClient('Search Clients', 'Search by client name or client ID, then select a client to open its documents.');
        if (client) { expandAncestors(client.path); await refreshTree(client.path); }
        break;
      }
      case 'search-documents':
      case 'global-find':
        showGlobalFind();
        break;
      case 'unarchive-clients': {
        const archivedClients = await backend().ListArchivedClients();
        if (!archivedClients || !archivedClients.length) {
          toast('There are no archived clients.', true);
          return;
        }
        const client = await chooseClientFromItems('Unarchive Clients', 'Choose an archived client to return to the active Work Space.', archivedClients);
        if (!client) return;
        if (!confirm(`Unarchive ${client.name}? The client will return to the active Work Space.`)) return;
        const restoredPath = await backend().UnarchiveClient(client.path);
        expanded.add(restoredPath);
        await refreshTree(restoredPath);
        toast(`${client.name} unarchived.`);
        break;
      }
      case 'rename-selected-document':
        await doAction('rename-file');
        break;
      case 'delete-selected':
        if (selectedNode?.type === 'file') await doAction('delete-file');
        else if (selectedNode?.type === 'client') await doAction('delete-client');
        else if (selectedNode?.type === 'taxyear') await doAction('delete-year');
        else if (selectedNode?.type === 'section') await doAction('delete-section');
        else throw new Error('Select a client, tax year, section, or document first.');
        break;
      case 'open-data-folder':
        await backend().OpenDataFolder();
        break;
      case 'manage-users':
        await showManageUsers();
        break;
      case 'create-company-user':
        await showExistingCompanyUser();
        break;
      case 'shortcut-mapper':
        showShortcutMapper();
        break;
      case 'change-password':
        hideChangePassword();
        $('changePasswordBackdrop').classList.remove('hidden');
        $('changeCurrentPassword').focus();
        break;
      case 'auto-lock-15':
      case 'auto-lock-30':
      case 'auto-lock-45': {
        const minutes = Number(action.split('-').pop());
        await backend().SetAutoLockMinutes(minutes);
        AUTO_LOCK_MS = minutes * 60 * 1000;
        lastDesktopInput = Date.now();
        updateAutoLockMenu();
        toast(`Auto-lock set to ${minutes} minutes of inactivity.`);
        break;
      }
      case 'theme-light':
        applyTheme('light');
        break;
      case 'theme-dark':
        applyTheme('dark');
        break;
      case 'tax-calculator':
        await backend().LogAppEvent('Federal Tax Calculator opened', 'Tools menu');
        showTaxCalculator();
        break;
      case 'firm-calendar':
        await backend().LogAppEvent('Firm Calendar opened', '2026 weekly client schedule');
        await showFirmCalendar();
        break;
      case 'engagement-letter':
        await backend().LogAppEvent('Engagement Letter opened', 'Tools menu');
        await showEngagementLetter();
        break;
      case 'create-invoice':
        await backend().LogAppEvent('Create Invoice opened', 'Tools menu');
        await showInvoiceGenerator();
        break;
      case 'currency-converter':
        await backend().LogAppEvent('Currency Converter opened', 'IRS yearly-average currency exchange rates');
        showCurrencyConverter();
        break;
      case 'app-audit-trail':
        await backend().LogAppEvent('Audit Trail opened', 'Company-wide history viewed');
        await showAppAuditTrail();
        break;
      case 'how-to-use':
        showUserManual();
        $('gettingStartedGuide').querySelector('details').open = true;
        break;
      case 'user-manual':
        showUserManual();
        break;
      case 'about':
        showAbout();
        break;
    }
  } catch (e) {
    showError(e);
  }
}

function chooseClient(title, help) {
  return chooseClientFromItems(title, help, treeData);
}

function chooseClientFromItems(title, help, items) {
  clientPickerItems = Array.isArray(items) ? items : [];
  if (!clientPickerItems.length) {
    toast('There are no clients to choose from.', true);
    return Promise.resolve(null);
  }
  $('clientPickerTitle').textContent = title;
  $('clientPickerHelp').textContent = help;
  $('clientPickerSearch').value = '';
  $('clientPickerBackdrop').classList.remove('hidden');
  renderClientPickerList();
  setTimeout(() => $('clientPickerSearch').focus(), 20);
  return new Promise(resolve => clientPickerResolver = resolve);
}

function renderClientPickerList() {
  const q = $('clientPickerSearch').value.trim().toLowerCase();
  const clients = clientPickerItems.filter(c => !q || c.name.toLowerCase().includes(q) || (c.clientId || '').includes(q));
  $('clientPickerList').innerHTML = clients.length ? clients.map(c => `
    <button type="button" class="client-picker-row" data-client-picker-path="${escapeHTML(c.path)}">
      <span class="client-picker-baby">👶</span>
      <span class="client-picker-name">${escapeHTML(c.name)}</span>
    </button>`).join('') : '<div class="client-picker-empty">No matching clients.</div>';
}

function closeClientPicker(value) {
  $('clientPickerBackdrop').classList.add('hidden');
  if (!clientPickerResolver) return;
  const resolve = clientPickerResolver;
  clientPickerResolver = null;
  clientPickerItems = [];
  resolve(value);
}

async function showClientCommunications() {
  clientCommunicationsRows = await backend().GetClientCommunications();
  $('clientCommunicationsSearch').value = '';
  renderClientCommunications();
  $('clientCommunicationsBackdrop').classList.remove('hidden');
  setTimeout(() => $('clientCommunicationsSearch').focus(), 20);
}

function renderClientCommunications() {
  const query = $('clientCommunicationsSearch').value.trim().toLowerCase();
  // Mirror this search into the main Work Space so Client Communications
  // also serves as a quick client-name navigator.
  $('treeSearch').value = query;
  renderTree();

  const rows = clientCommunicationsRows.filter(c => !query || String(c.name || '').toLowerCase().includes(query));
  const body = $('clientCommunicationsBody');
  body.innerHTML = rows.length ? rows.map(c => `
    <tr class="communications-row" data-communications-path="${escapeHTML(c.path)}" title="Select ${escapeHTML(c.name)} in the Work Space">
      <td><span class="communications-client">👶 ${escapeHTML(c.name)}</span></td>
      <td><span class="communications-return-type">${escapeHTML(c.returnType || '—')}</span></td>
      <td>${c.email ? `<a href="mailto:${escapeHTML(c.email)}" onclick="event.stopPropagation()">${escapeHTML(c.email)}</a>` : '<span class="communications-missing">—</span>'}</td>
      <td>${c.phone ? escapeHTML(c.phone) : '<span class="communications-missing">—</span>'}</td>
    </tr>`).join('') : `<tr><td colspan="4" class="communications-empty">${query ? 'No clients match this name.' : 'No clients have been created yet.'}</td></tr>`;
  $('clientCommunicationsStatus').textContent = `${rows.length} client${rows.length === 1 ? '' : 's'}`;
}

function hideClientCommunications() {
  $('clientCommunicationsBackdrop').classList.add('hidden');
}

function askClientDetails(initial = null, mode = 'new') {
  clientFormMode = mode;
  clientFormClient = initial || null;
  const props = initial || {};
  $('clientNameInput').value = props.name || '';
  $('clientAddressInput').value = props.address || '';
  $('clientPhoneInput').value = props.phone || '';
  $('clientEmailInput').value = props.email || '';
  $('clientTaxIDInput').value = props.taxId || '';
  $('clientReturnTypeInput').value = props.returnType || '1040';
  $('clientNameInput').disabled = false;
  $('addClientTitle').textContent = mode === 'edit' ? 'Edit Client' : 'Add Client';
  $('addClientSave').textContent = mode === 'edit' ? 'Save Changes' : 'Create Client';
  $('addClientBackdrop').querySelector('.client-form-modal > p').textContent = mode === 'edit'
    ? 'Update the client name, contact information, tax ID, or return type.'
    : 'Create a client profile. BabyFileCab will automatically assign the next 5-digit Client ID.';
  $('addClientBackdrop').classList.remove('hidden');
  setTimeout(() => $('clientNameInput').focus(), 20);
  return new Promise(resolve => clientFormResolver = resolve);
}

function submitClientForm() {
  if (!clientFormResolver) return;
  const name = $('clientNameInput').value.trim();
  if (!name) {
    $('clientNameInput').focus();
    toast('Client / Business Name is required.', true);
    return;
  }
  const value = {
    name,
    address: $('clientAddressInput').value.trim(),
    phone: $('clientPhoneInput').value.trim(),
    email: $('clientEmailInput').value.trim(),
    taxId: $('clientTaxIDInput').value.trim(),
    returnType: $('clientReturnTypeInput').value,
  };
  closeClientForm(value);
}

function closeClientForm(value) {
  if (!clientFormResolver) return;
  $('addClientBackdrop').classList.add('hidden');
  $('clientNameInput').disabled = false;
  const resolve = clientFormResolver;
  clientFormResolver = null;
  resolve(value);
}

function ask(title, help, value='') {
  $('modalTitle').textContent = title;
  $('modalHelp').textContent = help;
  $('modalInput').value = value;
  $('modalBackdrop').classList.remove('hidden');
  setTimeout(() => { $('modalInput').focus(); $('modalInput').select(); }, 20);
  return new Promise(resolve => modalResolver = resolve);
}

function closeModal(value) {
  if (!modalResolver) return;
  $('modalBackdrop').classList.add('hidden');
  const r = modalResolver; modalResolver = null; r(value);
}

async function updateSelectionClientSummary(node) {
  const client = clientForPath(node?.path || '');
  if (!client) {
    $('selectionClientName').textContent = '—';
    $('selectionReturnType').textContent = '—';
    return;
  }
  $('selectionClientName').textContent = client.name || '—';
  try {
    const props = await backend().GetClientProperties(client.path);
    $('selectionReturnType').textContent = props?.returnType || '—';
  } catch (_) {
    $('selectionReturnType').textContent = '—';
  }
}

function clientForPath(path) {
  for (const c of treeData) {
    if (path === c.path || path.startsWith(c.path + '/') || path.startsWith(c.path + '\\')) return c;
  }
  return null;
}

function updateAutoLockMenu() {
  for (const minutes of [15, 30, 45]) {
    const item = $(`autoLock${minutes}MenuItem`);
    if (item) {
      const chosen = AUTO_LOCK_MS === minutes * 60 * 1000;
      item.textContent = `${chosen ? '✓ ' : ''}${minutes} minutes`;
      item.setAttribute('aria-checked', String(chosen));
    }
  }
}

function applyTheme(theme, persist = true) {
  currentTheme = theme === 'dark' ? 'dark' : 'light';
  document.documentElement.dataset.theme = currentTheme;
  if (persist) localStorage.setItem('babyfilecab-theme', currentTheme);
  const logoSource = currentTheme === 'dark' ? 'logo-dark.png' : 'logo.png';
  document.querySelectorAll('img.logo, img.auth-logo').forEach(img => { img.src = logoSource; });
  if ($('lightThemeMenuItem')) $('lightThemeMenuItem').textContent = `${currentTheme === 'light' ? '✓ ' : ''}Light Theme`;
  if ($('darkThemeMenuItem')) $('darkThemeMenuItem').textContent = `${currentTheme === 'dark' ? '✓ ' : ''}Dark Theme`;
}

function clearPreview() {
  $('selectionTitle').textContent = 'Select a client or document';
  $('selectionClientName').textContent = '—';
  $('selectionReturnType').textContent = '—';
  $('preview').className = 'preview-empty';
  $('preview').innerHTML = '<div class="empty-icon">📄</div><h2>Document Preview</h2><p>Select a PDF, Word, Excel, email, image, CSV, text file, or notes.txt from the Work Space.</p>';
}

async function call(fn) {
  try { return await fn(); }
  catch (e) { showError(e); }
}

function showError(e) {
  const msg = typeof e === 'string' ? e : (e?.message || String(e));
  toast(msg, true);
}

function toast(message, error=false) {
  const t = $('toast'); t.textContent = message; t.className = `toast${error ? ' error' : ''}`;
  clearTimeout(t._timer); t._timer = setTimeout(() => t.className = 'toast hidden', 3500);
}

function escapeHTML(v) {
  return String(v ?? '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));
}
function cssEscape(v) { return String(v).replace(/["\\]/g, '\\$&'); }
