////////////////////////
// MERGE ENTITY
////////////////////////

// when "merge entity" dialog is shown, set up the form
on('show.bs.modal', '#modal-merge-entity', async e => {
	const entitySelectMerge = newEntitySelect($('#modal-merge-entity .entity-merge'), 1, true);
	const entitySelectKeep = newEntitySelect($('#modal-merge-entity .entity-keep'), 1, true);
	entitySelectMerge.clear(true);
	entitySelectKeep.clear(true);
	$('#do-entity-merge').classList.add('disabled');
	const entityIDMerge =  e?.relatedTarget?.dataset?.entityIDMerge;
	const entityIDKeep = e?.relatedTarget?.dataset?.entityIDKeep;
	if (entityIDMerge) {
		const entities = await app.SearchEntities({
			repo: tlz.openRepos[0].instance_id,
			row_id: [Number(entityIDMerge)]
		});
		entitySelectMerge.addOption(entities[0]);
		entitySelectMerge.addItem(entities[0].id);
	}
	if (entityIDKeep) {
		const entities = await app.SearchEntities({
			repo: tlz.openRepos[0].instance_id,
			row_id: [Number(entityIDKeep)]
		});
		entitySelectKeep.addOption(entities[0]);
		entitySelectKeep.addItem(entities[0].id);
	}
});

on('change', '#modal-merge-entity select', e => {
	const keepID = $('#modal-merge-entity .entity-keep').tomselect.getValue();
	const mergeID = $('#modal-merge-entity .entity-merge').tomselect.getValue();
	$('#do-entity-merge').title = Number(mergeID) === 1 ? 'Keep the timeline owner as the remaining entity.' : '';
	if (keepID && mergeID && String(keepID) !== String(mergeID) && Number(mergeID) !== 1) {
		$('#do-entity-merge').classList.remove('disabled');
	} else {
		$('#do-entity-merge').classList.add('disabled');
	}
});

on('click', '#do-entity-merge', async e => {
	e.preventDefault();
	const button = $('#do-entity-merge');
	if (button.classList.contains('disabled')) return;
	const keepID = Number($('#modal-merge-entity .entity-keep').tomselect.getValue());
	const mergeID = Number($('#modal-merge-entity .entity-merge').tomselect.getValue());
	if (!keepID || !mergeID || keepID === mergeID || mergeID === 1) return;
	button.classList.add('disabled');
	try {
		await app.MergeEntities(tlz.openRepos[0].instance_id, keepID, [mergeID]);
		bootstrap.Modal.getInstance($('#modal-merge-entity'))?.hide();
		if (keepID === 1 || mergeID === 1) await updateRepoOwners(true);
		notify({type: 'success', title: 'Entities merged', duration: 2000});
		await navigateSPA(`/entities/${tlz.openRepos[0].instance_id}/${keepID}`, true);
	} catch (error) {
		notify({type: 'warning', title: 'Merge request unconfirmed', message: 'Refresh the entity list before trying again.'});
	} finally { button.classList.remove('disabled'); }
});
