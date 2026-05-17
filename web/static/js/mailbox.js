$(function () {
  let selectedUID     = null;
  let selectedMailbox = null;

  // Clique em linha de mensagem — carrega painel de leitura via AJAX
  $(document).on('click', '.msg-row', function (e) {
    if ($(e.target).is('input[type=checkbox]')) return;

    $('.msg-row').removeClass('msg-selected').css({'background-color': '', 'border-left': ''});
    $(this).addClass('msg-selected').css({'background-color': '#d4e4f7', 'border-left': '3px solid #37517e'});

    selectedUID     = $(this).data('uid');
    selectedMailbox = $(this).data('mailbox');

    $('#reading-pane').load(
      '/mail/' + selectedMailbox + '/' + selectedUID + ' #message-body',
      function () {
        // marca como lido localmente
        const row = $('[data-uid="' + selectedUID + '"]');
        row.removeClass('bg-white font-semibold').addClass('bg-[#f8f9fa] text-gray-600');
      }
    );

    // Marca como lido no servidor
    $.post('/mail/' + selectedMailbox + '/' + selectedUID + '/flag',
           { flag: 'seen', value: 1 });

    updateToolbar();
  });

  // Selecionar todos
  $('#check-all').on('change', function () {
    $('.msg-check').prop('checked', this.checked);
    updateToolbar();
  });

  // Checkbox individual
  $(document).on('change', '.msg-check', function () {
    updateToolbar();
  });

  // Deletar (único ou em lote via checkboxes)
  $('#btn-delete').on('click', function () {
    const checked = $('.msg-check:checked').closest('.msg-row');
    const targets = checked.length > 0 ? checked : (selectedUID ? $('[data-uid="' + selectedUID + '"]') : $());
    if (!targets.length) return;

    const requests = targets.map(function () {
      const row  = $(this);
      const uid  = row.data('uid');
      const mbox = row.data('mailbox');
      return $.ajax({ url: '/mail/' + mbox + '/' + uid, method: 'DELETE' })
        .then(function () {
          row.fadeOut(200, function () { $(this).remove(); });
          if (uid == selectedUID) {
            selectedUID = null;
            $('#reading-pane').html('<div class="flex items-center justify-center h-full text-gray-300 text-sm">Select a message</div>');
          }
        });
    }).get();

    $.when.apply($, requests).always(function () { updateToolbar(); });
  });

  // Mark Read (single or bulk via checkboxes)
  $('#btn-mark-read').on('click', function () {
    const checked = $('.msg-check:checked').closest('.msg-row');
    const targets = checked.length > 0 ? checked : (selectedUID ? $('[data-uid="' + selectedUID + '"]') : $());
    if (!targets.length) return;

    const requests = targets.map(function () {
      const row  = $(this);
      const uid  = row.data('uid');
      const mbox = row.data('mailbox');
      return $.post('/mail/' + mbox + '/' + uid + '/flag', { flag: 'seen', value: 1 })
        .then(function () {
          row.removeClass('bg-white font-semibold').addClass('bg-[#f8f9fa] text-gray-600');
          row.find('.msg-check').prop('checked', false);
          row.find('span.truncate').removeClass('text-[#37517e]');
        });
    }).get();

    $.when.apply($, requests).always(function () { updateToolbar(); });
  });


  // Mover (único ou em lote via checkboxes)
  $('#move-select').on('change', function () {
    const dest = $(this).val();
    if (!dest) return;
    $(this).val('');

    const checked  = $('.msg-check:checked').closest('.msg-row');
    const targets  = checked.length > 0 ? checked : (selectedUID ? $('[data-uid="' + selectedUID + '"]') : $());
    if (!targets.length) return;

    const requests = targets.map(function () {
      const row  = $(this);
      const uid  = row.data('uid');
      const mbox = row.data('mailbox');
      return $.post('/mail/' + mbox + '/' + uid + '/move', { dest: dest })
        .then(function () {
          row.fadeOut(200, function () { $(this).remove(); });
          if (uid == selectedUID) {
            selectedUID = null;
            $('#reading-pane').html('<div class="flex items-center justify-center h-full text-gray-300 text-sm">Selecione uma mensagem</div>');
          }
        });
    }).get();

    $.when.apply($, requests).always(function () { updateToolbar(); });
  });

  // Paginação por scroll infinito
  let page = 1;
  let loading = false;
  $('#msg-list').on('scroll', function () {
    if (loading) return;
    if (this.scrollTop + this.clientHeight >= this.scrollHeight - 40) {
      loading = true;
      page++;
      $.get(location.pathname, { page: page }, function (html) {
        const rows = $(html).find('.msg-row');
        if (rows.length) {
          $('#msg-rows').append(rows);
        }
        loading = false;
      });
    }
  });

  function updateToolbar() {
    const hasSelected   = selectedUID !== null;
    const checkedCount  = $('.msg-check:checked').length;
    $('#btn-reply, #btn-forward').prop('disabled', !hasSelected);
    $('#btn-delete, #btn-mark-read, #move-select').prop('disabled', !hasSelected && checkedCount === 0);
    $('#btn-delete').prop('disabled', !hasSelected && checkedCount === 0);
  }

  function openCompose(to, subject, bodyHTML, title) {
    $('#compose-modal h3').text(title || 'New Message');
    $('#modal-to').val(to || '');
    $('#modal-subject').val(subject || '');
    const editor = tinymce.get('modal-body');
    if (editor) {
      editor.setContent(bodyHTML || '');
    } else {
      $('#modal-body').val(bodyHTML || '');
    }
    $('#attachments-list').empty();
    $('#modal-attach-label').text('No files selected');
    $('#compose-modal').removeClass('hidden');
  }

  function quotedBody() {
    const mb    = $('#message-body');
    const from  = mb.data('from-name') || mb.data('from-email') || '';
    const date  = mb.data('date') || '';
    const body  = $('#msg-body').html() || '';
    return '<br><br><hr style="border:none;border-top:1px solid #ccc;margin:8px 0">'
         + '<p style="color:#555;font-size:0.85em">On ' + date + ', ' + from + ' wrote:</p>'
         + '<blockquote style="border-left:3px solid #ccc;margin:4px 0;padding-left:12px;color:#555">'
         + body + '</blockquote>';
  }

  // Compose — New message
  $('#btn-novo-modal').on('click', function () {
    openCompose('', '', '', 'New Message');
  });

  // Reply
  $('#btn-reply').on('click', function () {
    const mb      = $('#message-body');
    const to      = mb.data('from-email') || '';
    const subject = 'Re: ' + (mb.data('subject') || '');
    openCompose(to, subject, quotedBody(), 'Reply');
  });

  // Forward
  $('#btn-forward').on('click', function () {
    const mb      = $('#message-body');
    const subject = 'Fwd: ' + (mb.data('subject') || '');
    openCompose('', subject, quotedBody(), 'Forward');
  });

  $('.close-compose').on('click', function () {
    $('#compose-modal').addClass('hidden');
  });
});
