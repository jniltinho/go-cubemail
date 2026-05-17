$(function () {
  // Atualiza contagem de não-lidos na sidebar a cada 2 minutos
  function refreshUnread() {
    $('.sidebar-folder').each(function () {
      const name = $(this).data('name');
      $.getJSON('/api/folders/' + encodeURIComponent(name) + '/count', function (res) {
        const badge = $('[data-folder-badge="' + name + '"]');
        if (res.unseen > 0) {
          badge.text(res.unseen).show();
        } else {
          badge.hide();
        }
      });
    });
  }

  setInterval(refreshUnread, 120000);

  // Expand / collapse subfolders
  function setDescendantsVisible(folderName, visible) {
    $('[data-parent="' + folderName + '"]').each(function () {
      if (visible) {
        $(this).show();
        if (!$(this).hasClass('folder-collapsed')) {
          setDescendantsVisible($(this).data('folder'), true);
        }
      } else {
        $(this).hide();
        setDescendantsVisible($(this).data('folder'), false);
      }
    });
  }

  $(document).on('click', '.folder-toggle', function (e) {
    e.preventDefault();
    e.stopImmediatePropagation();
    const item = $(this).closest('.folder-item');
    const folderName = String(item.data('folder'));
    const collapsed  = item.hasClass('folder-collapsed');
    const svg = $(this).find('svg')[0];

    if (collapsed) {
      item.removeClass('folder-collapsed');
      setDescendantsVisible(folderName, true);
      svg.style.transform = 'rotate(90deg)';
    } else {
      item.addClass('folder-collapsed');
      setDescendantsVisible(folderName, false);
      svg.style.transform = 'rotate(0deg)';
    }
  });

  // Folder context menu — abre/fecha ao clicar no botão ⋮
  $(document).on('click', '.folder-menu-btn', function (e) {
    e.preventDefault();
    e.stopImmediatePropagation();
    const dropdown = $(this).siblings('.folder-dropdown');
    $('.folder-dropdown').not(dropdown).addClass('hidden');
    if (dropdown.hasClass('hidden')) {
      const rect = this.getBoundingClientRect();
      dropdown.css({ top: rect.top, left: rect.right + 2 });
    }
    dropdown.toggleClass('hidden');
  });

  // Fecha dropdown ao clicar fora
  $(document).on('click', function () {
    $('.folder-dropdown').addClass('hidden');
  });

  // Nova Subpasta
  $(document).on('click', '.folder-action-subfolder', function (e) {
    e.stopPropagation();
    $('.folder-dropdown').addClass('hidden');
    const item   = $(this).closest('.folder-item');
    const parent = String(item.data('folder'));
    const delim  = String(item.data('delim') || '/');
    const name = prompt('New subfolder name:');
    if (!name || !name.trim()) return;
    $.post('/api/folders', { parent: parent, delim: delim, name: name.trim() })
      .done(function () { location.reload(); })
      .fail(function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : xhr.responseText || 'unknown error';
        alert('Error creating subfolder:\n' + msg);
      });
  });

  // Empty Trash
  $(document).on('click', '.folder-action-empty', function (e) {
    e.stopPropagation();
    $('.folder-dropdown').addClass('hidden');
    const folder = $(this).closest('.folder-item').data('folder');
    if (!confirm('Empty folder "' + folder + '"? This cannot be undone.')) return;
    $.ajax({
      url: '/mail/' + encodeURIComponent(String(folder)),
      method: 'DELETE',
      success: function () { location.reload(); },
      error: function () { alert('Error emptying folder.'); }
    });
  });

  // Rename folder
  $(document).on('click', '.folder-action-rename', function (e) {
    e.stopPropagation();
    $('.folder-dropdown').addClass('hidden');
    const item   = $(this).closest('.folder-item');
    const folder = String(item.data('folder'));
    const delim  = String(item.data('delim') || '/');
    const parts  = folder.split(delim);
    const current = parts[parts.length - 1];
    const newDisplay = prompt('Rename "' + current + '" to:', current);
    if (!newDisplay || !newDisplay.trim() || newDisplay.trim() === current) return;
    parts[parts.length - 1] = newDisplay.trim();
    const newname = parts.join(delim);
    $.post('/api/folders/rename', { name: folder, newname: newname })
      .done(function () { window.location.href = '/mail/' + encodeURIComponent(newname); })
      .fail(function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : 'unknown error';
        alert('Error renaming folder:\n' + msg);
      });
  });

  // Delete folder
  $(document).on('click', '.folder-action-delete', function (e) {
    e.stopPropagation();
    $('.folder-dropdown').addClass('hidden');
    const item   = $(this).closest('.folder-item');
    const folder = String(item.data('folder'));
    if (!confirm('Delete folder "' + folder + '" and all its messages? This cannot be undone.')) return;
    $.post('/api/folders/delete', { name: folder })
      .done(function () { window.location.href = '/mail/INBOX'; })
      .fail(function (xhr) {
        const msg = xhr.responseJSON && xhr.responseJSON.error ? xhr.responseJSON.error : 'unknown error';
        alert('Error deleting folder:\n' + msg);
      });
  });
});
