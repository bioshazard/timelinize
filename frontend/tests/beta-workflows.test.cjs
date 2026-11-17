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
    $: selector => selector === '.entity-input' ? {tomselect: control, isConnected: true} : null,
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

test('page-size selection survives a filter URL round trip', async () => {
  const {c} = filterContext('?limit=10&page=2');
  const input = {value: '10'};
  c.$ = selector => selector === '#filter-limit' ? input : null;
  const helpers = source('common-lib');
  vm.runInContext(helpers.slice(helpers.indexOf('function boundedPageSize('), helpers.indexOf('function escapeHTML(')), c);
  const qs = c.filterToQueryString();
  assert.equal(qs.get('limit'), '10');
  input.value = '50';
  await c.queryStringToFilter();
  assert.equal(input.value, 10);
});

test('none and Unknown type selections have distinct reload semantics', async () => {
  const {c} = filterContext('');
  const unknown = {value: '', checked: false};
  const message = {value: 'message', checked: false};
  c.$$ = selector => selector.includes('input:checked') ? [unknown, message].filter(x => x.checked) : selector.includes('tl-item-class-dropdown input') ? [unknown, message] : [];
  const none = c.filterToQueryString();
  assert.equal(none.get('types'), 'none');
  assert.equal(none.has('class'), false);
  unknown.checked = message.checked = true;
  c.window.location.search = '?' + none;
  await c.queryStringToFilter();
  assert.equal(unknown.checked, false);
  assert.equal(message.checked, false);
  unknown.checked = true;
  const onlyUnknown = c.filterToQueryString();
  assert.equal(onlyUnknown.get('class'), '');
  assert.equal(onlyUnknown.has('types'), false);
});

test('filter restoration ignores entity lookup after its original input is detached', async () => {
  const {c, selected} = filterContext('?entity=2');
  let resolve;
  const input = {tomselect: {addOption() {throw new Error('stale option');}, addItem() {throw new Error('stale item');}}, isConnected: true};
  c.$ = selector => selector === '.entity-input' ? input : null;
  c.app.SearchEntities = () => new Promise(r => {resolve = r;});
  const pending = c.queryStringToFilter();
  input.isConnected = false;
  resolve([{id: 2}]);
  await pending;
  assert.deepEqual(selected, []);
});

test('Enter on filter forms prevents native submission and applies current filters', () => {
  let handler, applied = 0, prevented = 0;
  const c = vm.createContext({on: (_event, _selector, fn) => {handler = fn;}, applyFilters: () => {applied++;}});
  section(c, 'common', "on('submit', 'form'", 'function classInfo(');
  handler({submitter: null, target: {matches: () => true}, preventDefault() {prevented++;}});
  assert.equal(applied, 1);
  assert.equal(prevented, 1);
});

test('conversation type-none queries never send native all-class searches', async () => {
  const c = vm.createContext({post: () => {throw new Error('unexpected request');}, Promise});
  const text = source('app');
  const conversation = text.slice(text.indexOf('LoadConversation(params)'), text.indexOf('ChartStats('));
  const recent = text.slice(text.indexOf('LoadRecentConversations(params)'), text.indexOf('MergeEntities('));
  vm.runInContext('var api = ({' + conversation + recent + '});', c);
  assert.equal((await c.api.LoadConversation({classification: []})).items.length, 0);
  assert.equal((await c.api.LoadRecentConversations({classification: []})).length, 0);
});

test('a failed page transition retains the intended Retry destination', async () => {
  const {c} = navigationContext();
  let destination;
  c.showPageError = (_page, target) => {destination = target;};
  c.fetch = async () => ({ok: false});
  await c.navigateSPA('/map?entity=2');
  assert.equal(destination, '/map?entity=2');
});

test('a stale map presearch cannot write a date into the next page', async () => {
  let resolve, current = true;
  const c = vm.createContext({
    beginViewRender: () => () => current,
    tlz: {openRepos: [{instance_id: 'demo'}]}, mapData: {},
    loadAndRenderHeatmap() {}, mapPageFilterParams: () => ({}),
    $: selector => {
      if (selector === '#map-page') return {};
      if (!current) throw new Error('stale map accessed next-page controls');
      return {datepicker: {selectedDates: []}};
    },
    app: {SearchItems: () => new Promise(r => {resolve = r;})},
  });
  section(c, 'map', 'async function loadAndRenderMapData()', 'function makeMarker(');
  const pending = c.loadAndRenderMapData();
  current = false;
  resolve({items: [{timestamp: '2020-01-01T12:00:00Z'}]});
  await pending;
});

test('delegated asynchronous failures produce feedback rather than unhandled rejection', async () => {
  const notices = [];
  const c = vm.createContext({console: {error() {}}, notify: n => notices.push(n), Promise});
  section(c, 'app', 'function invokeEventHandler(', 'function on(eventName,');
  c.invokeEventHandler(async () => {throw new Error('offline');}, {}, {});
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(notices.length, 1);
  assert.match(notices[0].message, /Check the current state/);
});

test('single-entity merge refuses self-merge and removal of the timeline owner', async () => {
  const handlers = new Map();
  let kept = '2', merged = '2', calls = 0;
  const button = {classList: {contains: () => false}};
  const c = vm.createContext({
    on: (_event, selector, handler) => handlers.set(selector, handler),
    $: selector => selector === '#do-entity-merge' ? button : {tomselect: {getValue: () => selector.includes('entity-keep') ? kept : merged}},
    app: {MergeEntities: async () => {calls++;}},
  });
  vm.runInContext(source('modals'), c);
  const submit = handlers.get('#do-entity-merge');
  await submit({preventDefault() {}});
  merged = '1';
  await submit({preventDefault() {}});
  assert.equal(calls, 0);
});
