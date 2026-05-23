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
      $('#app-modal-input').on('keydown.appmodal', function (e) {
        if (e.key === 'Enter') {
          e.preventDefault();
          const val = $('#app-modal-input').val().trim();
          cleanup(); resolve(val || null);
        }
      });
      $(document).on('keydown.appmodal', function (e) {
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
      $(document).on('keydown.appmodal', function (e) {
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
      $(document).on('keydown.appmodal', function (e) {
        if (e.key === 'Enter' || e.key === 'Escape') { cleanup(); resolve(); }
      });
    });
  }

  // ─── Folder Tree (jqTree) ──────────────────────────────────────────────────

  const $folderList = $('#folder-list');
  const currentMailbox = String($folderList.data('current-mailbox') || '');
  const foldersUrl = String($folderList.data('folders-url') || '/api/v1/folders');
  let folderTreeReady = false;

  function iconNameForFolder(iconType) {
    switch (iconType) {
      case 'inbox': return 'inbox';
      case 'sent': return 'send';
      case 'drafts': return 'pencil';
      case 'trash': return 'trash-2';
      case 'junk': return 'alert-triangle';
      default: return 'folder';
    }
  }

  function createChevronElement(opened) {
    const span = document.createElement('span');
    span.className = 'folder-toggle-chevron' + (opened ? ' is-open' : '');
    span.setAttribute('aria-hidden', 'true');
    span.innerHTML = '<svg viewBox="0 0 6 10" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M1 1l4 4-4 4"></path></svg>';
    return span;
  }

  function renderFolderIcons() {
    if (window.lucide && typeof window.lucide.createIcons === 'function') {
      window.lucide.createIcons();
    }
  }

  function buildFolderTreeData(folders) {
    const nodesById = new Map();
    const roots = [];

    folders.forEach(function (folder) {
      nodesById.set(folder.Name, {
        id: folder.Name,
        name: folder.DisplayName || folder.Name,
        fullName: folder.Name,
        delim: folder.Delim || '/',
        unseen: Number(folder.Unseen || 0),
        iconType: folder.IconType || 'folder',
        isTrash: folder.IsTrash === true,
        isSystem: folder.IsSystem === true,
        noSelect: folder.NoSelect === true,
        parentName: folder.ParentName || '',
        children: []
      });
    });

    folders.forEach(function (folder) {
      const node = nodesById.get(folder.Name);
      if (node.parentName && nodesById.has(node.parentName)) {
        nodesById.get(node.parentName).children.push(node);
      } else {
        roots.push(node);
      }
    });

    return roots;
  }

  function decorateFolderNode(node, $li) {
    const $element = $li.children('.jqtree-element');
    const $title = $element.children('.jqtree-title');
    const depth = Math.max(0, (typeof node.getLevel === 'function' ? node.getLevel() : 1) - 1);

    $element.addClass('folder-tree-row');
    $element.css('--folder-depth', String(depth));
    if (node.id === currentMailbox) {
      $element.addClass('is-active');
    }

    $element.children('.folder-tree-spacer, .folder-tree-unseen, .folder-menu-btn').remove();
    if (!$element.children('.jqtree-toggler').length) {
      $element.prepend('<span class="folder-tree-spacer" aria-hidden="true"></span>');
    }

    $title.addClass('folder-tree-title').empty();

    const $node = $('<span class="folder-tree-node"></span>');
    const $icon = $('<span class="folder-tree-icon"></span>');
    const $label = $('<span class="folder-tree-label"></span>').text(node.name);

    if (node.noSelect) {
      $label.addClass('is-placeholder');
    }

    $icon.append($('<i>').attr('data-lucide', iconNameForFolder(node.iconType)));
    $node.append($icon, $label);
    $title.append($node);

    if (node.unseen > 0) {
      $element.append(
        $('<span class="folder-tree-unseen"></span>').text(String(node.unseen))
      );
    }

    $element.append(
      $('<button type="button" class="folder-menu-btn" title="Folder actions">&#8942;</button>')
    );
  }

  function showFolderStatus(message) {
    $folderList.html(
      $('<div class="folder-tree-status"></div>').text(message)
    );
  }

  function revealFolderNode(nodeId) {
    if (!folderTreeReady || !nodeId) return;
    const node = $folderList.tree('getNodeById', nodeId);
    if (!node) return;
    if (node.parent) {
      $folderList.tree('openNode', node.parent, false);
    }
    $folderList.tree('scrollToNode', node);
  }

  function applyFolderTreeData(folders, options) {
    const treeData = buildFolderTreeData(folders);
    const state = folderTreeReady ? $folderList.tree('getState') : null;

    if (!folderTreeReady) {
      $folderList.empty().tree({
        data: treeData,
        autoOpen: 0,
        animationSpeed: 110,
        selectable: false,
        saveState: 'wm_folder_tree',
        useContextMenu: true,
        closedIcon: createChevronElement(false),
        openedIcon: createChevronElement(true),
        onCreateLi: decorateFolderNode
      });
      bindFolderTreeEvents();
      folderTreeReady = true;
    } else {
      $folderList.tree('loadData', treeData);
      if (state) {
        $folderList.tree('setState', state);
      }
    }

    renderFolderIcons();

    const targetNodeId = options && options.revealNodeId
      ? options.revealNodeId
      : currentMailbox;

    if (targetNodeId) {
      revealFolderNode(targetNodeId);
    }
  }

  function refreshFolderTree(options) {
    if (!folderTreeReady) {
      showFolderStatus('Loading folders...');
    }
    return $.getJSON(foldersUrl)
      .done(function (folders) {
        applyFolderTreeData(folders, options || {});
      })
      .fail(function () {
        if (!folderTreeReady) {
          showFolderStatus('Could not load folders.');
        }
      });
  }

  function bindFolderTreeEvents() {
    $folderList.on('tree.click', function (event) {
      const target = event.click_event && event.click_event.target;
      if ($(target).closest('.folder-menu-btn, .jqtree-toggler').length) {
        event.preventDefault();
        return;
      }
      if (event.node.noSelect) {
        event.preventDefault();
        return;
      }
      window.location.href = '/mail/' + encodeURIComponent(event.node.id);
      event.preventDefault();
    });

    $folderList.on('tree.contextmenu', function (event) {
      event.preventDefault();
      showContextMenu(event.click_event.clientX, event.click_event.clientY, event.node);
    });
  }

  refreshFolderTree();

  // ─── Context Menu ──────────────────────────────────────────────────────────

  let ctxFolder   = null;
  let ctxDelim    = '/';
  let ctxIsTrash  = false;
  let ctxIsSystem = false;

  function showContextMenu(x, y, node) {
    if (!node) return;

    ctxFolder   = String(node.fullName || node.id || '');
    ctxDelim    = String(node.delim || '/');
    ctxIsTrash  = node.isTrash === true;
    ctxIsSystem = node.isSystem === true;

    $('#ctx-non-system').toggle(!ctxIsSystem);
    $('#ctx-trash-only').toggle(ctxIsTrash);

    const $menu = $('#folder-context-menu').removeClass('hidden');
    const menuWidth = $menu.outerWidth();
    const menuHeight = $menu.outerHeight();
    const maxLeft = Math.max(8, window.innerWidth - menuWidth - 8);
    const maxTop = Math.max(8, window.innerHeight - menuHeight - 8);
    const left = Math.max(8, Math.min(x, maxLeft));
    const top = Math.max(8, Math.min(y, maxTop));

    $menu.css({ top: top, left: left });
  }

  $('#folder-list').on('click', '.folder-menu-btn', function (e) {
    e.preventDefault();
    e.stopPropagation();
    if (!folderTreeReady) return;
    const node = $folderList.tree('getNodeByHtmlElement', this);
    showContextMenu(e.clientX, e.clientY, node);
  });

  $(document).on('click.ctxmenu', function () {
    $('#folder-context-menu').addClass('hidden');
  });

  // ─── Context Menu Actions ──────────────────────────────────────────────────

  function isCurrentMailboxAffected(targetFolder, delim) {
    if (!targetFolder || !currentMailbox) return false;
    return currentMailbox === targetFolder || currentMailbox.startsWith(targetFolder + delim);
  }

  $('#ctx-subfolder').on('click', async function () {
    $('#folder-context-menu').addClass('hidden');
    if (!ctxFolder) return;
    const name = await appPrompt('New Subfolder', 'Enter the name for the new subfolder:', '');
    if (!name) return;
    $.post('/api/v1/folders', { parent: ctxFolder, delim: ctxDelim, name: name })
      .done(function (res) {
        refreshFolderTree({ revealNodeId: res && res.name ? res.name : ctxFolder });
      })
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
    $.post('/api/v1/folders/rename', { name: ctxFolder, newname: newname })
      .done(function () {
        if (isCurrentMailboxAffected(ctxFolder, ctxDelim)) {
          const nextMailbox = currentMailbox === ctxFolder
            ? newname
            : newname + currentMailbox.slice(ctxFolder.length);
          window.location.href = '/mail/' + encodeURIComponent(nextMailbox);
          return;
        }
        refreshFolderTree({ revealNodeId: newname });
      })
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
    $.post('/api/v1/folders/delete', { name: ctxFolder })
      .done(function () {
        if (isCurrentMailboxAffected(ctxFolder, ctxDelim)) {
          window.location.href = '/mail/INBOX';
          return;
        }
        refreshFolderTree();
      })
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
      success: function () {
        if (currentMailbox === ctxFolder) {
          location.reload();
          return;
        }
        refreshFolderTree({ revealNodeId: ctxFolder });
      },
      error: async function () { await appAlert('Error', 'Error emptying folder.'); }
    });
  });

});
