// When .map-container or .loader-container elems are added to the DOM, observe them for scrolling into view.
const mutationObs = new MutationObserver(function(mutations) {
	for (const mut of mutations) {
		for (const node of mut.addedNodes) {
			if (!(node instanceof HTMLElement)) continue; // skip text/whitespace nodes
			
			if (node.matches('.map-container')) {
				tlz.mapIntersectionObs.observe(node);
			}
			$$('.map-container', node).forEach(el => {
				tlz.mapIntersectionObs.observe(el);
			});
		}
		// don't unobserve removedNodes, because the intersection observer
		// removes map containers from a set when they go out of view,
		// and if we unobserve it can't do that, causing a memory leak;
		// but we can fire mapMoved events for special-use maps that may
		// need to reset some state or initialization, such as the map on
		// the Demo Mode Settings page, which adds draw controls (it needs
		// to remove them when navigating away).
		for (const node of mut.removedNodes) {
			if (!(node instanceof HTMLElement)) continue; // skip text/whitespace nodes
			
			if (node.matches('.map-container')) {
				document.dispatchEvent(new CustomEvent("mapMoved", { detail: { previousElement: node } }));
			}
			$$('.map-container', node).forEach(el => {
				document.dispatchEvent(new CustomEvent("mapMoved", { detail: { previousElement: el } }));
			});
		}
	}
 });
 mutationObs.observe(document, {childList: true, subtree:true});


function addrBarPathToPagePath(userFriendlyPath) {
	if (userFriendlyPath == "/") {
		userFriendlyPath = "/dashboard"
	}
	return `/pages${userFriendlyPath}.html`;
}

function splitPathAndQueryString(uri) {
	const qsStart = uri.indexOf('?');
	if (qsStart > -1) {
		uri.substring(0, qsStart);
		return {
			path: uri.substring(0, qsStart),
			query: uri.substring(qsStart)
		};
	}
	return {path: uri};
}

function currentURI() {
	return window.location.pathname + window.location.search;
}

// navigateSPA changes the page. The argument should be the URI (not including scheme/host; i.e. just
// path and optional query string) to show in the address bar (the user-friendly URI). If omitted,
// or not a string (sometimes it is an event handler) it defaults to the path and query currently
// in the address bar.
let navigationVersion = 0;

async function navigateSPA(addrBarDestination, scrollToTop) {
	const version = ++navigationVersion;
	const current = () => version === navigationVersion;
	const page = $('#page-content');
	page.classList.add('opacity0');
	let loader;
	const slowLoadingHandle = setTimeout(() => {
		if (!current()) return;
		loader = document.createElement('span');
		loader.classList.add('slow-loader', 'page-loader');
		page.before(loader);
	}, 1000);

	try {
		if (typeof addrBarDestination !== 'string') addrBarDestination = currentURI();
		if (!tlz.openRepos.length || !await updateRepoOwners()) {
			addrBarDestination = '/setup';
		} else if (addrBarDestination.startsWith('/setup')) {
			addrBarDestination = '/';
		}
		if (!current()) return;
		const skipPushState = addrBarDestination === currentURI();
		const parts = splitPathAndQueryString(addrBarDestination);
		const matches = parts.path.match(/\/(items|entities|jobs)\/[\w-]+\/\d+$/);
		if (matches) parts.path = {items: '/item', entities: '/entity', jobs: '/job'}[matches[1]];
		const destPath = addrBarPathToPagePath(parts.path);
		const response = await fetch(destPath + (parts.query || ''));
		if (!response.ok) throw new Error('Page unavailable');
		const data = await response.text();
		// An unknown route may return the SPA shell with HTTP 200.
		if (!tlz.pageControllers[destPath]) throw new Error('Page unavailable');
		if (!current()) return;

		if (tlz.map) {
			tlz.map.tl_containers = new Map();
			tlz.map.tl_clear();
		}
		for (const input of $$('.date-input')) input.datepicker?.destroy();
		for (const element of $$('[data-bs-toggle="tooltip"]')) bootstrap.Tooltip.getInstance(element)?.dispose();
		for (const element of $$('#page-content *')) element.chart?.dispose();
		await tlz.currentPageController?.unload?.();
		if (!current()) return;
		if (!skipPushState) history.pushState(null, null, addrBarDestination);
		page.innerHTML = data;
		initResponsiveFilters();
		if (matchMedia('(max-width: 767.98px)').matches) {
			bootstrap.Collapse.getInstance($('#navbar-menu'))?.hide();
		}
		const title = $('body title');
		if (title) { document.title = title.textContent; title.remove(); }
		const controller = tlz.pageControllers[destPath];
		tlz.currentPageController = controller;
		await controller.load?.();
		if (!current()) return;
		for (const element of $$('.filter select.tl-data-source')) {
			await newDataSourceSelect(element);
			if (!current()) return;
		}
		for (const element of $$('.filter .tl-item-class-dropdown')) {
			renderFilterDropdown(element, 'Types', 'item_classes');
		}
		await queryStringToFilter();
		if (!current()) return;
		await controller.render?.();
		if (!current()) return;
		activateTooltips();
		if (scrollToTop) window.scrollTo({top: 0, left: 0, behavior: 'instant'});
	} catch (error) {
		if (current()) showPageError(page, addrBarDestination);
		console.error('Page could not be loaded', error);
	} finally {
		clearTimeout(slowLoadingHandle);
		loader?.remove();
		if (current()) {
			page.classList.remove('opacity0');
			$('#app-loader')?.remove();
		}
	}
}

// when links to pages within the app are clicked, fake-navigate
on('click', '[href^="/"]:not([download])', async e => {
	const link = e.target.closest('a');
	if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey || link?.target === '_blank') return;
	e.preventDefault();
	if (link?.matches('.disabled, [aria-disabled="true"]')) return;
	const destination = e.target.closest(':not(use)[href]').getAttribute('href'); // can't use .href because that returns a fully-qualified URL, which actually breaks in Wails dev; and we only accept path+query
	await navigateSPA(destination, true);
	return false;
});

// this is for filter changes
on('click', `.filter [href^="?"], .pagination [href^="?"]`, async event => {
	event.preventDefault();
	if (event.target.closest('a')?.matches('.disabled, [aria-disabled="true"]')) return;
	
	// update URL bar so the filter will read the updated page number
	const destination = event.target.closest(':not(use)[href]').getAttribute('href'); // can't use .href because that returns a fully-qualified URL, which actually breaks in Wails dev; and we only accept path+query
	history.pushState(null, null, destination);

	updateFilterResults();

	return false;
});


// updateRepoOwners updates the repo owners as stored locally, only if
// the information is missing (or if forced === true).
async function updateRepoOwners(forced) {
	let anyUpdated = false;
	for (const repo of tlz.openRepos) {
		if (!forced && repo.owner) {
			continue; // avoid re-checking on _every single page load_
		}
		if (await app.RepositoryIsEmpty(repo.instance_id)) {
			return false;
		} else {
			repo.owner = await app.GetEntity(repo.instance_id, 1);
			anyUpdated = true;
		}
	}
	if (anyUpdated) {
		store('open_repos', tlz.openRepos);

		const repoOwner = tlz.openRepos[0].owner;
		store('owner', repoOwner);
		
		let birthPlace = "";
		if (repoOwner.attributes) {
			for (const attr of repoOwner.attributes) {
				if (attr.name == "birth_place") {
					birthPlace = attr.value;
					break;
				}
			}
		}

		repoOwner.forceUpdate = forced; // can notify UI to load new profile picture or other details, if applicable
		$('#repo-owner-name').innerText = repoOwner.name;
		$('#repo-owner-title').innerText = birthPlace;
		$('#repo-owner-avatar').innerHTML = avatar(false, repoOwner, "avatar-sm");
		$('header .profile-link').href = `/entities/${tlz.openRepos[0].instance_id}/${repoOwner.id}`;
	}
	return true;
}

async function updateItemClasses() {
	const clArray = await app.ItemClassifications(tlz.openRepos[0].instance_id);
	const classes = {};
	for (const cl of clArray) {
		classes[cl.name] = cl;
	}
	store('item_classes', classes);
}


async function initialize() {
	// classifications are stored in the database, so get open repos first and attach owner info
	tlz.openRepos = await app.OpenRepositories();

	await updateRepoOwners();

	// if there's an open repo, load classifications
	if (tlz.openRepos.length && !load('item_classes')) {
		await updateItemClasses();
	}

	// load app settings
	tlz.settings = await app.GetSettings();

	// now that settings are loaded, initialize the map
	initMapSingleton();

	// perform initial page load
	if (document.readyState === 'loading') {
		window.addEventListener('DOMContentLoaded', navigateSPA);
	} else {
		navigateSPA();
	}

	// when back button is pressed, also do SPA nav (TODO: test this works)
	window.addEventListener('popstate', navigateSPA);
}

initialize();

// Collapsed by default on phones; preserve the user's choice while filtering.
// Reinitialize only for a new page, and reopen when crossing to desktop.
function initResponsiveFilters() {
	const mobile = window.matchMedia('(max-width: 767.98px)').matches;
	for (const panel of document.querySelectorAll('.responsive-filters')) {
		panel.open = !mobile;
	}
}
window.matchMedia('(max-width: 767.98px)').addEventListener('change', initResponsiveFilters);
