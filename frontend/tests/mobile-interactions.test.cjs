const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');

function conversationContext(media = []) {
  const context = vm.createContext({
    on() {}, setTimeout, clearTimeout,
    $$: () => media,
    requestAnimationFrame() { throw new Error('Unbounded media polling'); },
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../resources/js/conversations.js'), 'utf8'), context);
  return context;
}

test('broken images do not block opening a conversation', async () => {
  const c = conversationContext([{tagName: 'IMG', complete: true, naturalHeight: 0, offsetHeight: 0}]);
  await c.forMediaToRender();
});

test('a failed pending attachment settles and removes its listeners', async () => {
  const listeners = new Map();
  const media = {tagName: 'IMG', complete: false, addEventListener: (event, fn) => listeners.set(event, fn), removeEventListener: event => listeners.delete(event)};
  const c = conversationContext([media]);
  const ready = c.forMediaToRender();
  assert.ok(listeners.has('error'));
  listeners.get('error')();
  await ready;
  assert.equal(listeners.size, 0);
});

test('opening another conversation clears prior paging state before loading', async () => {
  const c = conversationContext();
  const node = {classList: {add() {}, remove() {}}, replaceChildren() {}, scrollHeight: 100};
  c.$ = selector => selector === '.entity-input' ? {tomselect: {items: ['2'], options: {'2': {name: 'Example'}}}} : node;
  c.messages = [{timestamp: 'old-thread'}];
  let previousCount;
  c.renderConversationChunk = async () => {previousCount = c.messages.length;};
  await c.renderSingleConversation();
  assert.equal(previousCount, 0);
  assert.equal(c.messages.length, 0);
  assert.equal(node.textContent, 'Example');
});

test('item pagination requests one sentinel beyond the selected page size', () => {
  const c = vm.createContext({
    on() {}, console: {log() {}},
    $: selector => selector === '#filter-limit' ? {value: '50'} : selector === '.date-sort' ? {value: 'DESC'} : {datepicker: {}},
    currentPageNum: () => 2,
    commonFilterSearchParams() {}, queryParam: () => null,
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../resources/js/items.js'), 'utf8'), c);
  const params = c.itemsPageFilterParams();
  assert.equal(params.offset, 50);
  assert.equal(params.limit, 51);
});
