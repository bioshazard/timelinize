// render functions should be synchronous (async should be awaited) so that the page load isn't janky.
tlz.pageControllers = {
	"/pages/conversations.html": {
		async load() {
			await conversationsPageMain();
		},
		async render() {
			await renderConversationsPage();
		},
		unload() {
			conversationVersion++;
			messages = [];
			_rendering = false;
		}
	},

	"/pages/dashboard.html": {
		load() {
			$('.navbar').classList.add('navbar-overlap');
			$('.navbar').dataset.bsTheme = "dark";

			const ctaButton = $('.nav-item .btn-outline-primary');
			ctaButton.classList.remove('btn-outline-primary');
			ctaButton.classList.add('btn-primary');
		},
		async render() {
			await renderDashboard();
		},
		unload() {
			$('.navbar').classList.remove('navbar-overlap');
			delete($('.navbar').dataset.bsTheme);

			const ctaButton = $('.nav-item .btn-primary');
			ctaButton.classList.remove('btn-primary');
			ctaButton.classList.add('btn-outline-primary');
		}

	},

	"/pages/setup.html": {
		async load() {
			// fill out year picker
			const startYear = new Date().getFullYear() - 10;
			for (let year = startYear; year > startYear - 110; year--) {
				$('select[name=dob-year]').innerHTML += `<option value="${year}">${year}</option>`;
			}

			for (let i = 0; i < tlz.openRepos.length; i++) {
				if (await app.RepositoryIsEmpty(tlz.openRepos[i].instance_id)) {
					notify({
						type: "info",
						title: "Timeline is empty",
						message: "It needs your profile"
					});
					emptyRepo = tlz.openRepos[i];
					advanceToPersonForm();
					return;
				}
			}
		}
	},

	"/pages/settings.html": {
		async load() {
			changeSettingsTab(window.location.hash || "#general");

			await loadSettingsForm();
		}
	},

	"/pages/entities.html": {
		async render() {
			await entitiesPageMain();
		}
	},

	"/pages/entity.html": {
		async render() {
			await entityPageMain();
		}
	},
	
	"/pages/gallery.html": {
		load() {
			$('body').classList.add('layout-fluid');
			$('header').classList.add('sticky-top', 'translucent');
			
			$('#content-column').prepend(cloneTemplate('#tpl-pagination'));
			$('#content-column').append(cloneTemplate('#tpl-pagination'));
			
			newEntitySelect($('.entity-input'), 1);
			
			$('.filter').prepend(newDatePicker({
				passthru: {
					range: true,
				},
				time: false,
				timeToggle: true,
				proximity: true,
				vertical: true
			}));
			$('.filter .date-sort').value = "DESC"; // makes sense in a gallery view, to start with most recent
		},
		async render() {
			await galleryPageMain();
		},
		unload() {
			$('body').classList.remove('layout-fluid');
			$('header').classList.remove('sticky-top', 'translucent');
		}
	},

	"/pages/import.html": {
		load() {
			// TODO: maybe this could be a more standard pattern for "global" vars on a specific page load only (see also the delete in the unload below)
			window.page = {};

			// populate default settings so even if the settings modal isn't opened, essential defaults will be set
			saveAdvancedSettings();

			$('#timeframe').append(newDatePicker({
				passthru: {
					range: true
				},
				sort: false,
				timeToggle: true,
				noApply: true
			}));
		},
		unload() {
			delete window.page;
		}
	},

	"/pages/input.html": {
		async render() {
			inputPageMain();
		},
	},

	"/pages/item.html": {
		async render() {
			await itemPageMain();
		}
	},

	"/pages/items.html": {
		load() {
			$('.pagination-container').append(cloneTemplate('#tpl-pagination'));

			$('.tl-date-picker').append(newDatePicker({
				passthru: {
					range: true
				},
				rangeToggle: true,
				time: false,
				timeToggle: true,
				proximity: true,
				vertical: true
			}));

			newEntitySelect($('.entity-input'), 5, true);
		},
		async render() {
			await itemsMain();
		}
	},

	"/pages/map.html": {
		load() {
			$('body').classList.add('layout-fluid');

			$('.tl-date-picker').prepend(newDatePicker({
				passthru: {
					range: false
				},
				rangeToggle: true,
				time: false,
				timeToggle: true,
				proximity: false,
				vertical: false
			}));
			newEntitySelect($('#select-person'), 1, true);
		},
		async render() {
			await loadAndRenderMapData();
		},
		unload() {
			$('body').classList.remove('layout-fluid');
			mapData = {};
		}
	},

	"/pages/timeline.html": {
		load() {
			newEntitySelect($('.entity-input'), 1);

			$('.filter').prepend(newDatePicker({
				passthru: {
					range: false
				},
				rangeToggle: true,
				time: false,
				timeToggle: true,
				proximity: false,
				vertical: true
			}));
		},
		async render() {
			const stillCurrent = beginViewRender($('.filter-results'));
			// figure out the most recent day with content, and use that as the default/initial timeline
			const presearchParams = {sort: "DESC"};
			commonFilterSearchParams(presearchParams);
			if (!presearchParams.start_timestamp && !presearchParams.end_timestamp && !presearchParams.timestamp) {
				const presearch = await app.SearchItems({
					...presearchParams,
					limit: 1,
					flat: true
				});
				if (!stillCurrent()) return;
				if (presearch?.items?.length) {
					$('.filter .date-input').datepicker.selectDate(presearch.items[0].timestamp);
				}
			}

			const groups = await loadAllGroups();

			console.log("ALL GROUPS:", groups)

			if (!stillCurrent()) return;
			$('.filter-results').replaceChildren(groups.length ? renderTimelineGroups(groups) : emptyState('No items match this date and these filters.'));
		}
	},

	"/pages/job.html": {
		load() {
			$('.navbar').classList.add('navbar-overlap');
		},
		async render() {
			jobPageMain();
		},
		unload() {
			$('.navbar').classList.remove('navbar-overlap');
			clearInterval(jobThroughputInterval); // stop trying to update the chart
		}
	},
}