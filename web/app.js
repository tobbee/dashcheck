'use strict';

const $ = (id) => document.getElementById(id);
const worker = new Worker('worker.js');
let nextID = 1;
let current = 0; // id of the running check
let rules = null; // rule ID -> rule
const pending = new Map(); // id -> {resolve, reject}

worker.onmessage = (ev) => {
  const msg = ev.data;
  if (msg.type === 'progress') {
    if (msg.id === current) showProgress(msg.progress);
    return;
  }
  const p = pending.get(msg.id);
  if (!p) return;
  pending.delete(msg.id);
  if (msg.type === 'error') p.reject(new Error(msg.message));
  else p.resolve(msg);
};

function call(msg) {
  const id = nextID++;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    worker.postMessage({ ...msg, id });
  });
}

function options() {
  return {
    maxSegments: Number($('maxSegments').value) || 4,
    reference: $('reference').value.trim() || 'none',
  };
}

async function loadRules() {
  if (!rules) {
    const msg = await call({ type: 'rules' });
    rules = Object.fromEntries(msg.rules.map((r) => [r.id, r]));
  }
  return rules;
}

async function run(msg, label) {
  setBusy(true);
  $('results').hidden = true;
  $('status').hidden = false;
  $('bar').style.width = '2%';
  $('statusText').textContent = `Checking ${label}…`;
  try {
    await loadRules();
    current = nextID; // the id call() assigns next
    const res = await call(msg);
    if (current === res.id) render(res.report);
  } catch (e) {
    renderError(e.message);
  } finally {
    setBusy(false);
    $('status').hidden = true;
  }
}

let busy = false;

function setBusy(b) {
  busy = b;
  updateButtons();
}

// The Check button stays disabled until a URL is entered, so an empty
// field with its example placeholder does not look ready to run.
function updateButtons() {
  $('run').disabled = busy || $('url').value.trim() === '';
  $('runLocal').disabled = busy;
}

$('url').addEventListener('input', updateButtons);

function showProgress(p) {
  const phases = { mpd: 5, clock: 15, segments: 20 };
  let pct = phases[p.phase] || 5;
  if (p.phase === 'segments' && p.total > 0) pct = 20 + (80 * p.done) / p.total;
  $('bar').style.width = `${pct}%`;
  if (p.message) $('statusText').textContent = p.message;
}

function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'class') e.className = v;
    else e.setAttribute(k, v);
  }
  for (const c of children) {
    if (c != null) e.append(c);
  }
  return e;
}

function render(rpt) {
  const summary = $('summary');
  summary.replaceChildren();
  const add = (k, v) => summary.append(el('dt', {}, k), el('dd', {}, v));
  add('MPD', rpt.mpdURL);
  if (rpt.type) add('Type', rpt.type);
  if (rpt.clockSource === 'local clock') {
    add('Clock', 'local clock (no time source could be reached)');
  } else if (rpt.clockSource) {
    add('Clock', `synced to ${rpt.clockSource}; this computer's clock is ${describeLocal(rpt.clockOffsetS || 0)} it`);
  }
  add('Segments', `${rpt.segmentsChecked} checked in ${rpt.durationS.toFixed(1)} s`);

  const findings = rpt.findings || [];
  const counts = { error: 0, warning: 0, info: 0 };
  for (const f of findings) counts[f.severity]++;
  $('nError').textContent = counts.error;
  $('nWarning').textContent = counts.warning;
  $('nInfo').textContent = counts.info;

  // Group by Representation; stream-level findings first.
  const groups = new Map([['', []]]);
  for (const f of findings) {
    const key = f.location.repID || '';
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(f);
  }
  const container = $('findings');
  container.replaceChildren();
  for (const [rep, list] of groups) {
    if (list.length === 0 && rep !== '') continue;
    const title = rep === '' ? 'MPD and stream' : 'Representation ';
    const h = el('h2', {}, title, rep ? el('span', { class: 'muted' }, rep) : null);
    const group = el('div', { class: 'group' }, h);
    if (list.length === 0) group.append(el('p', { class: 'ok' }, 'No findings.'));
    for (const f of list) group.append(renderFinding(f));
    container.append(group);
  }
  applyFilter();
  $('results').hidden = false;
}

// describeLocal describes this computer's clock relative to the clock source,
// given offset = source minus local clock, in seconds.
function describeLocal(offsetS) {
  const ms = Math.round(offsetS * 1000);
  if (ms > 0) return `${ms}ms behind`;
  if (ms < 0) return `${-ms}ms ahead of`;
  return 'in sync with';
}

function renderFinding(f) {
  const rule = rules && rules[f.rule];
  const loc = f.location || {};
  const parts = [];
  if (loc.segment) parts.push(`segment ${loc.segment}`);
  if (loc.path) parts.push(loc.path);
  if (loc.line) parts.push(`line ${loc.line}:${loc.col}`);
  if (loc.url && !loc.repID) parts.push(loc.url);
  let msg = f.message;
  if (f.count > 1) msg += ` (and ${f.count - 1} more segments)`;
  const head = el('div', {},
    el('span', { class: 'rule', title: rule ? rule.desc : '' }, f.rule), ' ',
    el('span', { class: 'msg' }, msg));
  const row = el('div', { class: 'finding', 'data-sev': f.severity },
    el('span', { class: `sev ${f.severity}` }, f.severity), head);
  const ref = f.specRef ? el('span', { class: 'ref' }, ` · ${f.specRef}`) : null;
  if (parts.length || ref) row.append(el('div', { class: 'loc' }, parts.join(' · '), ref));
  return row;
}

function renderError(message) {
  const container = $('findings');
  let hint = '';
  if (/Failed to fetch|NetworkError|Load failed/i.test(message)) {
    hint = ' The server may not allow cross-origin requests (CORS), or an https page tried to fetch an http:// URL. ' +
      'The command-line tool has neither limitation.';
  }
  $('summary').replaceChildren();
  for (const id of ['nError', 'nWarning', 'nInfo']) $(id).textContent = '0';
  container.replaceChildren(el('div', { class: 'error-box' }, message + hint));
  $('results').hidden = false;
}

function applyFilter() {
  const shown = {};
  for (const chip of document.querySelectorAll('.chip')) {
    shown[chip.dataset.sev] = chip.getAttribute('aria-pressed') === 'true';
  }
  for (const row of document.querySelectorAll('.finding')) {
    row.classList.toggle('hidden', !shown[row.dataset.sev]);
  }
}

for (const chip of document.querySelectorAll('.chip')) {
  chip.addEventListener('click', () => {
    const on = chip.getAttribute('aria-pressed') === 'true';
    chip.setAttribute('aria-pressed', String(!on));
    applyFilter();
  });
}

$('form').addEventListener('submit', (ev) => {
  ev.preventDefault();
  const url = $('url').value.trim();
  if (!url) return;
  const u = new URL(location.href);
  u.searchParams.set('url', url);
  history.replaceState(null, '', u);
  run({ type: 'check', url, options: options() }, url);
});

let folderFiles = [];
$('folder').addEventListener('change', () => {
  folderFiles = Array.from($('folder').files);
  const mpds = folderFiles.filter((f) => f.name.toLowerCase().endsWith('.mpd'));
  const select = $('mpdSelect');
  select.replaceChildren(...mpds.map((f) => el('option', { value: f.webkitRelativePath || f.name }, f.webkitRelativePath || f.name)));
  select.hidden = mpds.length === 0;
  $('runLocal').hidden = mpds.length === 0;
  if (mpds.length === 0) renderError('No .mpd file in the selected folder.');
});

$('runLocal').addEventListener('click', () => {
  const mpdPath = $('mpdSelect').value;
  run({ type: 'checkLocal', files: folderFiles, mpdPath, options: options() }, mpdPath);
});

for (const a of document.querySelectorAll('[data-url]')) {
  a.addEventListener('click', (ev) => {
    ev.preventDefault();
    $('url').value = a.dataset.url;
    $('form').requestSubmit();
  });
}

$('showRules').addEventListener('click', async () => {
  const r = await loadRules();
  const rows = Object.values(r).map((x) => el('tr', {},
    el('td', {}, x.id), el('td', {}, x.severity), el('td', {}, x.desc, x.specRef ? el('div', { class: 'ref' }, x.specRef) : null)));
  $('rulesList').replaceChildren(el('table', {},
    el('thead', {}, el('tr', {}, el('th', {}, 'Rule'), el('th', {}, 'Severity'), el('th', {}, 'Description'))),
    el('tbody', {}, ...rows)));
  $('rulesDialog').showModal();
});
$('closeRules').addEventListener('click', () => $('rulesDialog').close());

updateButtons();
const initial = new URL(location.href).searchParams.get('url');
if (initial) {
  $('url').value = initial;
  $('form').requestSubmit();
}
