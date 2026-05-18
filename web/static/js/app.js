$(function () {

  // ─── Custom Modal Helpers ─────────────────────────────────────────────────

  function appPrompt(title, message, value) {
    return new Promise(function (resolve) {
      $('#app-modal-title').text(title);
      $('#app-modal-message').text(message);
      $('#app-modal-input-wrap').removeClass('hidden');
      $('#app-modal-input').val(value || '');
      $('#app-modal-cancel').show();
      $('#app-modal').removeClass('hidden');
      $('#app-modal-input').trigger('focus');

      function cleanup() {
        $('#app-modal').addClass('hidden');
        $('#app-modal-input-wrap').addClass('hidden');
        $('#app-modal-ok, #app-modal-cancel, #app-modal-backdrop').off('.appmodal');
        $(document).off('keydown.appmodal');
      }

      $('#app-modal-ok').one('click.appmodal', function () {
        const val = $('#app-modal-input').val().trim();
        cleanup(); resolve(val || null);
      });
      $('#app-modal-cancel, #app-modal-backdrop').one('click.appmodal', function () {
        cleanup(); resolve(null);
      });
      $(document).one('keydown.appmodal', function (e) {
        if (e.key === 'Enter') { const val = $('#app-modal-input').val().trim(); cleanup(); resolve(val || null); }
        else if (e.key === 'Escape') { cleanup(); resolve(null); }
      });
    });
  }

  function appConfirm(title, message) {
    return new Promise(function (resolve) {
      $('#app-modal-title').text(title);
      $('#app-modal-message').text(message);
      $('#app-modal-input-wrap').addClass('hidden');
      $('#app-modal-cancel').show();
      $('#app-modal').removeClass('hidden');

      function cleanup() {
        $('#app-modal').addClass('hidden');
        $('#app-modal-ok, #app-modal-cancel, #app-modal-backdrop').off('.appmodal');
        $(document).off('keydown.appmodal');
      }

      $('#app-modal-ok').one('click.appmodal', function () { cleanup(); resolve(true); });
      $('#app-modal-cancel, #app-modal-backdrop').one('click.appmodal', function () { cleanup(); resolve(false); });
      $(document).one('keydown.appmodal', function (e) {
        if (e.key === 'Enter') { cleanup(); resolve(true); }
        else if (e.key === 'Escape') { cleanup(); resolve(false); }
      });
    });
  }

  function appAlert(title, message) {
    return new Promise(function (resolve) {
      $('#app-modal-title').text(title);
      $('#app-modal-message').text(message);
      $('#app-modal-input-wrap').addClass('hidden');
      $('#app-modal-cancel').hide();
      $('#app-modal').removeClass('hidden');

      function cleanup() {
        $('#app-modal').addClass('hidden');
        $('#app-modal-cancel').show();
        $('#app-modal-ok, #app-modal-backdrop').off('.appmodal');
        $(document).off('keydown.appmodal');
      }

      $('#app-modal-ok').one('click.appmodal', function () { cleanup(); resolve(); });
      $('#app-modal-backdrop').one('click.appmodal', function () { cleanup(); resolve(); });
      $(document).one('keydown.appmodal', function (e) {
        if (e.key === 'Enter' || e.key === 'Escape') { cleanup(); resolve(); }
      });
    });
  }

  // ─── Folder Tree (expand/collapse) ────────────────────────────────────────

  const collapsed = new Set(
    JSON.parse(localStorage.getItem('wm_collapsed') || '[]')
  );

  function saveCollapsed() {
    localStorage.setItem('wm_collapsed', JSON.stringify([...collapsed]));
  }

  function rowByFolder(name) {
    return $('#folder-list .folder-row').filter(function () {
      return $(this).data('folder') === name;
    });
  }

  function isDescendantHidden(parent) {
    let p = parent;
    while (p) {
      if (collapsed.has(p)) return true;
      const parentRow = rowByFolder(p);
      p = parentRow.length ? String(parentRow.data('parent') || '') : '';
    }
    return false;
  }

  function updateTree() {
    $('#folder-list .folder-row').each(function () {
      const row    = $(this);
      const parent = String(row.data('parent') || '');
      row.toggleClass('hidden', !!parent && isDescendantHidden(parent));
    });

    $('#folder-list .folder-row').each(function () {
      if (!$(this).data('has-children')) return;
      const folder = String($(this).data('folder') || '');
      const deg = collapsed.has(folder) ? '0deg' : '90deg';
      $(this).find('.toggle-chevron').css('transform', 'rotate(' + deg + ')');
    });
  }

  updateTree();

  $('#folder-list').on('click', '.folder-toggle', function (e) {
    e.preventDefault();
    e.stopPropagation();
    const folder = String($(this).closest('.folder-row').data('folder') || '');
    if (collapsed.has(folder)) {
      collapsed.delete(folder);
    } else {
      collapsed.add(folder);
    }
    saveCollapsed();
    updateTree();
  });

  // ─── Context Menu ──────────────────────────────────────────────────────────

  let ctxFolder   = null;
  let ctxDelim    = '/';
  let ctxIsTrash  = false;
  let ctxIsSystem = false;

  function showContextMenu(x, y, nodeEl) {
    const row     = nodeEl.closest('.folder-row');
    ctxFolder     = String(row.data('folder') || '');
    ctxDelim      = String(row.data('delim') || '/');
    ctxIsTrash    = row.data('is-trash')  === true;
    ctxIsSystem   = row.data('is-system') === true;

    $('#ctx-non-system').toggle(!ctxIsSystem);
    $('#ctx-trash-only').toggle(ctxIsTrash);

    $('#folder-context-menu').css({ top: y, left: x }).removeClass('hidden');
  }

  $('#folder-list').on('click', '.folder-menu-btn', function (e) {
    e.preventDefault();
    e.stopPropagation();
    showContextMenu(e.pageX, e.pageY, $(this));
  });

  $(document).on('click.ctxmenu', function () {
    $('#folder-context-menu').addClass('hidden');
  });

  // ─── Context Menu Actions ──────────────────────────────────────────────────

  $('#ctx-subfolder').on('click', async function () {
    $('#folder-context-menu').addClass('hidden');
    if (!ctxFolder) return;
    const name = await appPrompt('New Subfolder', 'Enter the name for the new subfolder:', '');
    if (!name) return;
    $.post('/api/folders', { parent: ctxFolder, delim: ctxDelim, name: name })
      .done(function () { location.reload(); })
      .fail(async function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : 'unknown error';
        await appAlert('Error', 'Error creating subfolder: ' + msg);
      });
  });

  $('#ctx-rename').on('click', async function () {
    $('#folder-context-menu').addClass('hidden');
    if (!ctxFolder) return;
    const parts   = ctxFolder.split(ctxDelim);
    const current = parts[parts.length - 1];
    const newDisplay = await appPrompt('Rename Folder', 'Rename "' + current + '" to:', current);
    if (!newDisplay || newDisplay === current) return;
    parts[parts.length - 1] = newDisplay;
    const newname = parts.join(ctxDelim);
    $.post('/api/folders/rename', { name: ctxFolder, newname: newname })
      .done(function () { window.location.href = '/mail/' + encodeURIComponent(newname); })
      .fail(async function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : 'unknown error';
        await appAlert('Error', 'Error renaming folder: ' + msg);
      });
  });

  $('#ctx-delete').on('click', async function () {
    $('#folder-context-menu').addClass('hidden');
    if (!ctxFolder) return;
    const ok = await appConfirm('Delete Folder', 'Delete "' + ctxFolder + '" and all its messages? This cannot be undone.');
    if (!ok) return;
    $.post('/api/folders/delete', { name: ctxFolder })
      .done(function () { window.location.href = '/mail/INBOX'; })
      .fail(async function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : 'unknown error';
        await appAlert('Error', 'Error deleting folder: ' + msg);
      });
  });

  $('#ctx-empty').on('click', async function () {
    $('#folder-context-menu').addClass('hidden');
    if (!ctxFolder) return;
    const ok = await appConfirm('Empty Trash', 'Empty "' + ctxFolder + '"? This cannot be undone.');
    if (!ok) return;
    $.ajax({
      url: '/mail/' + encodeURIComponent(ctxFolder),
      method: 'DELETE',
      success: function () { location.reload(); },
      error: async function () { await appAlert('Error', 'Error emptying folder.'); }
    });
  });

});
