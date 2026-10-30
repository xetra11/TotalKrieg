// Exports concept/project/Main.dc.html as static HTML and a PNG (default state, popup open).
// Run from anywhere: node tools/export-concept.js
// Set CHROME if Chrome is not at the default Windows path.
const fs = require('fs');
const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const concept = path.join(__dirname, '..', 'concept');
const src = path.join(concept, 'project', 'Main.dc.html');
const outHtml = path.join(concept, 'operational-map-battle-popup.html');
const outPng = path.join(concept, 'operational-map-battle-popup.png');
const chrome = process.env.CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';

const html = fs.readFileSync(src, 'utf8');

const code = html.match(/<script type="text\/x-dc"[^>]*>([\s\S]*?)<\/script>/)[1];
const Component = new Function('DCLogic', code + '\nreturn Component;')(
  class { constructor() { this.props = {}; } setState() {} }
);
const vals = new Component().renderVals();

const lookup = (p, scope) => {
  if (p === 'true') return true;
  if (p === 'false') return false;
  return p.split('.').reduce((v, k) => (v == null ? v : v[k]), scope);
};

// Finds the close tag matching the open tag at `start`, counting nested ones.
function closeOf(s, tag, start) {
  const re = new RegExp(`<${tag}\\b|</${tag}>`, 'g');
  re.lastIndex = start;
  let depth = 0, m;
  while ((m = re.exec(s))) {
    depth += m[0].startsWith('</') ? -1 : 1;
    if (depth === 0) return m.index;
  }
  throw new Error('unclosed ' + tag);
}

function expand(s, scope) {
  let result = '';
  let pos = 0;
  const re = /<(sc-for|sc-if)\b([^>]*)>/g;
  let m;
  while ((m = re.exec(s))) {
    if (m.index < pos) continue;
    result += holes(s.slice(pos, m.index), scope);
    const end = closeOf(s, m[1], m.index);
    const body = s.slice(m.index + m[0].length, end);
    if (m[1] === 'sc-for') {
      const list = lookup(m[2].match(/list="\{\{\s*([\w.]+)\s*\}\}"/)[1], scope);
      const as = m[2].match(/as="(\w+)"/)[1];
      for (const item of list) result += expand(body, { ...scope, [as]: item });
    } else if (lookup(m[2].match(/value="\{\{\s*([\w.]+)\s*\}\}"/)[1], scope)) {
      result += expand(body, scope);
    }
    pos = end + m[1].length + 3;
    re.lastIndex = pos;
  }
  return result + holes(s.slice(pos), scope);
}

function holes(s, scope) {
  return s
    .replace(/\s+on[A-Z]\w*="\{\{[^}]*\}\}"/g, '')
    .replace(/\s+disabled="\{\{\s*([\w.]+)\s*\}\}"/g, (_, p) => (lookup(p, scope) ? ' disabled' : ''))
    .replace(/\{\{\s*([\w.]+)\s*\}\}/g, (_, p) => {
      const v = lookup(p, scope);
      return v == null ? '' : String(v);
    });
}

const helmet = html.match(/<helmet>([\s\S]*?)<\/helmet>/)[1];
const tpl = html.match(/<\/helmet>([\s\S]*?)<\/x-dc>/)[1];
const title = html.match(/<title>(.*?)<\/title>/)[1];

fs.writeFileSync(outHtml, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>${title}</title>
${helmet.trim()}
</head>
<body>
${expand(tpl, vals).trim()}
</body>
</html>
`);

const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'totalkrieg-export-'));
execFileSync(chrome, [
  '--headless=new',
  '--disable-gpu',
  '--hide-scrollbars',
  '--force-device-scale-factor=2',
  '--window-size=1600,900',
  '--virtual-time-budget=6000',
  `--user-data-dir=${profile}`,
  `--screenshot=${outPng}`,
  'file:///' + outHtml.replace(/\\/g, '/'),
], { stdio: 'ignore' });
fs.rmSync(profile, { recursive: true, force: true });
