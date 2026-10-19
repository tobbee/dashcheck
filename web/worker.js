// Web Worker that runs dashcheck.wasm, keeping the page responsive.
// It adds a "local:" URL scheme to fetch, served from files the user picked,
// so local packager output can be checked without uploading anything.
importScripts('wasm_exec.js');

const localFiles = new Map(); // path -> File
const networkFetch = self.fetch.bind(self);

self.fetch = async (input, init = {}) => {
  const url = typeof input === 'string' ? input : input.url;
  if (!url.startsWith('local:')) {
    return networkFetch(input, init);
  }
  const path = decodeURIComponent(new URL(url).pathname);
  const file = localFiles.get(path);
  if (!file) {
    return new Response('not found', { status: 404 });
  }
  let body = file;
  let status = 200;
  const range = init.headers && init.headers.Range;
  if (range) {
    const m = /bytes=(\d+)-(\d*)/.exec(range);
    if (m) {
      const start = Number(m[1]);
      const end = m[2] === '' ? file.size : Number(m[2]) + 1;
      body = file.slice(start, end);
      status = 206;
    }
  }
  return new Response(init.method === 'HEAD' ? null : body, { status });
};

const go = new Go();
const ready = (async () => {
  let result;
  try {
    result = await WebAssembly.instantiateStreaming(networkFetch('dashcheck.wasm'), go.importObject);
  } catch (e) {
    // Servers that do not send application/wasm.
    const buf = await (await networkFetch('dashcheck.wasm')).arrayBuffer();
    result = await WebAssembly.instantiate(buf, go.importObject);
  }
  go.run(result.instance);
  if (!self.dashcheck) {
    throw new Error('dashcheck did not initialize');
  }
})();

self.onmessage = async (ev) => {
  const msg = ev.data;
  const reply = (type, data) => self.postMessage({ type, id: msg.id, ...data });
  const onProgress = (p) => reply('progress', { progress: JSON.parse(p) });
  try {
    await ready;
    switch (msg.type) {
      case 'check': {
        const rpt = await self.dashcheck.check(msg.url, JSON.stringify(msg.options), onProgress);
        reply('result', { report: JSON.parse(rpt) });
        break;
      }
      case 'checkLocal': {
        localFiles.clear();
        for (const f of msg.files) {
          localFiles.set('/' + (f.webkitRelativePath || f.name), f);
        }
        const mpdURL = 'local:/' + encodeURI(msg.mpdPath);
        const mpd = localFiles.get('/' + msg.mpdPath);
        const data = new Uint8Array(await mpd.arrayBuffer());
        const rpt = await self.dashcheck.checkData(data, mpdURL, JSON.stringify(msg.options), onProgress);
        reply('result', { report: JSON.parse(rpt) });
        break;
      }
      case 'rules':
        reply('rules', { rules: JSON.parse(self.dashcheck.rules()) });
        break;
      case 'version':
        reply('version', { version: self.dashcheck.version() });
        break;
    }
  } catch (e) {
    reply('error', { message: String(e && e.message ? e.message : e) });
  }
};
