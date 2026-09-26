const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const script = fs.readFileSync(path.join(__dirname, 'invitation.js'), 'utf8');

function page(hash, language = 'ru') {
  const elements = Object.fromEntries(['open', 'title', 'description', 'install', 'hint', 'privacy']
    .map(id => [id, { hidden: true, textContent: '', removeAttribute(name) { delete this[name]; } }]));
  const document = { documentElement: {}, getElementById: id => elements[id] };
  const listeners = {};
  const window = { location: { hash }, addEventListener: (name, handler) => { listeners[name] = handler; } };
  vm.runInNewContext(script, { document, window, navigator: { language } });
  return { elements, document, window, listeners };
}

test('opens the exact public contact without navigating automatically', () => {
  const key = Buffer.alloc(32, 1).toString('base64url');
  const { elements, window } = page('#c2/' + key);
  assert.equal(elements.open.href, 'shum://c2/' + key);
  assert.equal(elements.open.hidden, false);
  assert.deepEqual(window.location, { hash: '#c2/' + key });
});

test('invalid and injected fragments cannot become app links', () => {
  for (const hash of ['', '#', '#c2/short', '#javascript:alert(1)', '#c2/' + 'A'.repeat(44), '#c2/' + 'A'.repeat(42) + 'B', '#c2/<script>alert(1)</script>', '#c2/' + 'A'.repeat(43) + '?next=https://evil.test']) {
    const { elements } = page(hash);
    assert.equal(elements.open.hidden, true, hash);
    assert.equal(elements.open.href, undefined, hash);
  }
});

test('changing to an invalid invitation clears the previous contact', () => {
  const result = page('#c2/' + 'A'.repeat(43));
  result.window.location.hash = '#invalid';
  result.listeners.hashchange();
  assert.equal(result.elements.open.hidden, true);
  assert.equal(result.elements.open.href, undefined);
});

test('English instructions are available for other device languages', () => {
  const result = page('#c2/' + 'A'.repeat(43), 'en-US');
  assert.equal(result.document.documentElement.lang, 'en');
  assert.equal(result.elements.open.textContent, 'Open in Shum');
});
