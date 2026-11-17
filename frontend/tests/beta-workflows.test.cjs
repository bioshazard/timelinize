const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const source = name => fs.readFileSync(path.join(__dirname, '../resources/js', name + '.js'), 'utf8');
function section(context, name, start, end) {
  const text = source(name);
  assert.ok(text.indexOf(start) >= 0 && text.indexOf(end) > text.indexOf(start));
  vm.runInContext(text.slice(text.indexOf(start), text.indexOf(end)), context);
}
function filterContext(search) {
  const selected = [];
  const control = {addOption() {}, addItem(id) {selected.push(id);}};
  const c = vm.createContext({
    URLSearchParams, Number, console,
    window: {location: {search}}, currentPageNum: () => 1,
    tlz: {openRepos: [{instance_id: 'demo'}]},
    $: selector => selector === '.entity-input' ? {tomselect: control} : null,
    $$: () => [],
    app: {SearchEntities: async params => params.row_id.map(id => ({id}))},
  });
  section(c, 'common-lib', 'function filterToQueryString()', '// commonFilterSearchParams');
  return {c, selected};
}

test('multi-person URLs restore every selected entity, ignoring malformed IDs', async () => {
  const {c, selected} = filterContext('?entity=2,3,NaN,-1');
  await c.queryStringToFilter();
  assert.deepEqual(selected, [2, 3]);
});

test('changed filters reset pagination while retaining unrelated filter context', () => {
  const {c} = filterContext('?page=9&entity=2,3&data_source=facebook');
  const qs = c.filterToQueryString();
  assert.equal(qs.has('page'), false);
  assert.equal(qs.get('entity'), '2,3');
  assert.equal(qs.get('data_source'), 'facebook');
});

test('old classification names do not crash saved URLs', async () => {
  const {c} = filterContext('?class=obsolete');
  await c.queryStringToFilter();
});

test('page parsing rejects invalid, negative, fractional and infinite values', () => {
  const c = vm.createContext({URLSearchParams, window: {location: {search: ''}}});
  section(c, 'common', 'function currentPageNum()', 'function activateTooltips()');
  for (const value of ['-1', '0', 'NaN', 'Infinity', '2.5']) {
    c.window.location.search = '?page=' + value;
    assert.equal(c.currentPageNum(), 1);
  }
  c.window.location.search = '?page=3';
  assert.equal(c.currentPageNum(), 3);
});

test('render guards reject stale responses and detached pages', () => {
  const c = vm.createContext({});
  section(c, 'common', 'function beginViewRender(', 'function emptyState(');
  const node = {isConnected: true};
  const first = c.beginViewRender(node);
  const second = c.beginViewRender(node);
  assert.equal(first(), false);
  assert.equal(second(), true);
  node.isConnected = false;
  assert.equal(second(), false);
});

test('JSON responses with a charset decode and never change disabled controls', async () => {
  const c = vm.createContext({
    fetch: async () => ({ok: true, headers: {get: () => 'application/json; charset=utf-8'}, json: async () => ({ok: true})}),
    $$: () => {throw new Error('Network helper touched form state');},
  });
  section(c, 'app', 'async function myFetch(', 'const app =');
  assert.equal((await c.myFetch(null, '/api/example')).ok, true);
});

test('failed native requests reject rather than pretending success', async () => {
  const c = vm.createContext({fetch: async () => ({ok: false, status: 503, headers: {get: () => 'application/json'}, json: async () => ({message: 'unavailable'})})});
  section(c, 'app', 'async function myFetch(', 'const app =');
  await assert.rejects(c.myFetch(null, '/api/example'), err => err.response.status === 503);
});

test('archive-provided names are escaped in HTML and quoted titles', () => {
  const c = vm.createContext({});
  const text = source('common-lib');
  vm.runInContext(text.slice(text.indexOf('function escapeHTML(')), c);
  assert.equal(c.escapeHTML('<img src=x onerror="oops"> & \'quoted\''), '&lt;img src=x onerror=&quot;oops&quot;&gt; &amp; &#39;quoted&#39;');
});

function navigationContext() {
  const classes = new Set();
  const page = {classList: {add: x => classes.add(x), remove: x => classes.delete(x)}, before() {}, innerHTML: ''};
  const errors = [];
  const c = vm.createContext({
    console: {error() {}}, setTimeout, clearTimeout,
    window: {location: {pathname: '/items', search: ''}, scrollTo() {}},
    tlz: {openRepos: [{}], pageControllers: {'/pages/items.html': {}}},
    $: selector => selector === '#page-content' ? page : null,
    $$: () => [], updateRepoOwners: async () => true,
    showPageError: () => errors.push('visible error'),
  });
  section(c, 'spa', 'function addrBarPathToPagePath(', '// when links to pages within the app');
  return {c, page, classes, errors};
}

test('failed navigation reveals an error instead of leaving an invisible page', async () => {
  const {c, classes, errors} = navigationContext();
  c.fetch = async () => ({ok: false});
  await c.navigateSPA('/items');
  assert.equal(classes.has('opacity0'), false);
  assert.equal(errors.length, 1);
});

test('an old navigation response cannot overwrite a newer destination', async () => {
  const {c, page, errors} = navigationContext();
  let resolve, started;
  const ready = new Promise(r => {started = r;});
  c.fetch = () => {started(); return new Promise(r => {resolve = r;});};
  const first = c.navigateSPA('/items');
  await ready;
  c.fetch = async () => ({ok: false});
  await c.navigateSPA('/unavailable');
  resolve({ok: true, text: async () => '<old page>'});
  await first;
  assert.equal(page.innerHTML, '');
  assert.equal(errors.length, 1);
});

test('job requests block duplicate taps and recover controls after an uncertain failure', async () => {
  const classes = new Set();
  const attributes = new Set();
  const control = {dataset: {jobId: '4'}, classList: {contains: x => classes.has(x), add: x => classes.add(x), remove: x => classes.delete(x)}, setAttribute: x => attributes.add(x), removeAttribute: x => attributes.delete(x)};
  let reject, calls = 0;
  const notices = [];
  const c = vm.createContext({
    on() {}, tlz: {openRepos: [{instance_id: 'demo'}]},
    $$: () => [control], notify: n => notices.push(n),
    app: {PauseJob: () => {calls++; return new Promise((_, r) => {reject = r;});}},
  });
  const text = source('jobs');
  vm.runInContext(text.slice(text.indexOf('async function runJobAction(')), c);
  const event = {preventDefault() {}, target: {closest: () => control}};
  const first = c.runJobAction(event, 'pause');
  await c.runJobAction(event, 'pause');
  assert.equal(calls, 1);
  reject(new Error('network lost'));
  await first;
  assert.equal(classes.has('disabled'), false);
  assert.equal(attributes.has('aria-disabled'), false);
  assert.match(notices[0].message, /Refresh job status/);
});


test('an explicit empty type selection does not silently search all native classes', async () => {
  let called = false;
  const c = vm.createContext({post: () => {called = true;}, Promise});
  const text = source('app');
  vm.runInContext('var api = ({' + text.slice(text.indexOf('SearchItems(params)'), text.indexOf('SearchEntities(params)')) + '});', c);
  const result = await c.api.SearchItems({classification: []});
  assert.equal(result.total, 0);
  assert.equal(called, false);
  c.api.SearchItems({});
  assert.equal(called, true);
});
