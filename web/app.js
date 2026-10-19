'use strict';

const $ = (id) => document.getElementById(id);
const worker = new Worker('worker.js');
let nextID = 1;
let current = 0; // id of the running check
let rules = null; // rule ID -> rule
let ruleList = []; // rules in display order, grouped
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
    // 0 in the form means no update checks, which the Go side spells -1.
    mpdUpdates: Number($('mpdUpdates').value) || -1,
    profiles: profileChoice(),
    hlsInterop: $('hlsInterop').checked,
    reference: $('reference').value.trim() || 'none',
  };
}

// profileChoice returns the checked profiles as -profiles spells them,
// such as "auto" or "dvb,hbbtv".
function profileChoice() {
  const names = [...document.querySelectorAll('#profiles input[data-profile]:checked')].map((i) => i.dataset.profile);
  if ($('profileAuto').checked) names.unshift('auto');
  return names.join(',') || 'none';
}

// setProfileChoice checks the profiles of a choice such as "auto,dvb".
function setProfileChoice(choice) {
  const names = choice.split(',').map((s) => s.trim());
  $('profileAuto').checked = names.includes('auto');
  for (const i of document.querySelectorAll('#profiles input[data-profile]')) {
    i.checked = names.includes(i.dataset.profile);
  }
}

async function loadProfiles() {
  const msg = await call({ type: 'profiles' });
  $('profileChoices').replaceChildren(...msg.profiles.map((p) =>
    el('label', { class: 'toggle', title: p.prefix ? `${p.prefix}…` : p.urn },
      el('input', { type: 'checkbox', 'data-profile': p.name }), ` ${p.title}`)));
}

async function loadRules() {
  if (!rules) {
    const msg = await call({ type: 'rules' });
    ruleList = msg.rules;
    rules = Object.fromEntries(msg.rules.map((r) => [r.id, r]));
  }
  return rules;
}

// Visits and checks are counted with GoatCounter, without cookies, on the
// published page only, so not on localhost or on a fork's copy. The counts are
// sent to its /count endpoint directly rather than with its count.js, which
// would also send the query string, and that can hold the MPD URL of a check.
// They carry the path, title, screen width and the referrer without its query.
const goatCounter = 'https://dashcheck.goatcounter.com/count';
const countedHost = 'tobbee.github.io';

function countURL(path, title, event) {
  const q = new URLSearchParams({ p: path, t: title, e: String(event), s: String(screen.width) });
  if (!event && document.referrer) q.set('r', document.referrer.split(/[?#]/)[0]);
  q.set('rnd', Math.random().toString(36).slice(2, 7));
  return `${goatCounter}?${q}`;
}

function count(path, title, event) {
  if (location.hostname !== countedHost || navigator.webdriver) return;
  const url = countURL(path, title, event);
  if (!navigator.sendBeacon || !navigator.sendBeacon(url)) new Image().src = url;
}

// countCheck counts a check as an event, by kind only, never with the MPD URL.
function countCheck(kind) {
  count(`check-${kind}`, `Check ${kind}`, true);
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
  if (rpt.type) {
    add('Type', rpt.type);
    const names = (rpt.profiles || []).join(', ') || 'no profile-specific rules';
    const choice = rpt.profileChoice || 'auto';
    add('Profiles', choice === 'auto' ? `${names} (from @profiles)` : `${names} (chosen: ${choice})`);
  }
  if (rpt.clockSource === 'local clock') {
    add('Clock', 'local clock (no time source could be reached)');
  } else if (rpt.clockSource) {
    add('Clock', `synced to ${rpt.clockSource}; this computer's clock is ${describeLocal(rpt.clockOffsetS || 0)} it`);
  }
  add('Segments', `${rpt.segmentsChecked} checked in ${rpt.durationS.toFixed(1)} s`);
  if (rpt.mpdUpdatesChecked) {
    const n = rpt.mpdUpdatesChecked;
    add('Updates', `MPD fetched ${n} more time${n > 1 ? 's' : ''} and compared with the version before`);
  }

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
  const profiles = profileChoice();
  if (profiles === 'auto') u.searchParams.delete('profiles');
  else u.searchParams.set('profiles', profiles);
  history.replaceState(null, '', u);
  countCheck('url');
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
  countCheck('folder');
  run({ type: 'checkLocal', files: folderFiles, mpdPath, options: options() }, mpdPath);
});

for (const a of document.querySelectorAll('[data-url]')) {
  a.addEventListener('click', (ev) => {
    ev.preventDefault();
    $('url').value = a.dataset.url;
    $('form').requestSubmit();
  });
}

// renderRules lists the rules under a heading per group, with links to the
// groups at the top.
function renderRules() {
  const groups = [];
  for (const r of ruleList) {
    if (!groups.length || groups[groups.length - 1].name !== r.group) groups.push({ name: r.group, rules: [] });
    groups[groups.length - 1].rules.push(r);
  }
  const bodies = groups.map((g, i) => el('tbody', { id: `rules-group-${i}` },
    el('tr', { class: 'group' }, el('th', { colspan: '3' }, g.name, el('span', { class: 'count' }, String(g.rules.length)))),
    ...g.rules.map((x) => el('tr', { class: 'rule-row', 'data-text': `${x.id} ${x.desc} ${x.specRef || ''}`.toLowerCase() },
      el('td', {}, x.id),
      el('td', {}, el('span', { class: `sev ${x.severity}` }, x.severity)),
      el('td', {}, x.desc, x.specRef ? el('div', { class: 'ref' }, x.specRef) : null)))));
  $('rulesList').replaceChildren(el('table', {}, ...bodies));
  $('rulesGroups').replaceChildren(...groups.map((g, i) => {
    const a = el('a', { href: `#rules-group-${i}` }, g.name);
    a.addEventListener('click', (ev) => {
      ev.preventDefault();
      $(`rules-group-${i}`).scrollIntoView({ block: 'start' });
    });
    return a;
  }));
}

// filterRules hides the rules that do not contain the filter text, and the
// groups and group links that have none left.
function filterRules() {
  const q = $('rulesFilter').value.trim().toLowerCase();
  document.querySelectorAll('#rulesList tbody').forEach((body, i) => {
    let n = 0;
    for (const row of body.querySelectorAll('.rule-row')) {
      const hit = !q || row.dataset.text.includes(q);
      row.hidden = !hit;
      if (hit) n++;
    }
    body.hidden = n === 0;
    body.querySelector('.count').textContent = String(n);
    $('rulesGroups').children[i].hidden = n === 0;
  });
}

$('showRules').addEventListener('click', async () => {
  await loadRules();
  if (!$('rulesList').firstChild) renderRules();
  $('rulesFilter').value = '';
  filterRules();
  $('rulesDialog').showModal();
  $('rulesList').scrollTop = 0;
});
$('rulesFilter').addEventListener('input', filterRules);
$('closeRules').addEventListener('click', () => $('rulesDialog').close());
// A click on the backdrop closes the dialog. Clicks inside it, also on its
// padding, and keyboard activation of its buttons do not.
$('rulesDialog').addEventListener('click', (ev) => {
  const d = ev.currentTarget;
  if (ev.target !== d) return;
  const r = d.getBoundingClientRect();
  if (ev.clientX < r.left || ev.clientX > r.right || ev.clientY < r.top || ev.clientY > r.bottom) d.close();
});

updateButtons();
count(location.pathname, document.title, false);
call({ type: 'version' }).then((msg) => { $('version').textContent = msg.version; }, () => {});
loadRules().then(() => { $('ruleCount').textContent = ` (${ruleList.length})`; }, () => {});
// A shared link has the MPD URL and, unless auto, the profiles.
const params = new URL(location.href).searchParams;
loadProfiles().then(() => {
  if (params.has('profiles')) setProfileChoice(params.get('profiles'));
}, () => {}).then(() => {
  const initial = params.get('url');
  if (initial) {
    $('url').value = initial;
    $('form').requestSubmit();
  }
});
