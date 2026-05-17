$(function () {
  // Mostrar/esconder Cc e Bcc
  $('#btn-cc').on('click', function () { $('#cc-row').toggleClass('hidden flex'); });
  $('#btn-bcc').on('click', function () { $('#bcc-row').toggleClass('hidden flex'); });

  // Upload de anexos
  $('#attach-input').on('change', function () {
    const fd = new FormData();
    $.each(this.files, function (i, f) { fd.append('files[]', f); });
    $.ajax({
      url: '/compose/upload',
      method: 'POST',
      data: fd,
      processData: false,
      contentType: false,
      success: function (res) {
        (res.files || []).forEach(function (f) {
          $('#attachment-list').append(
            '<span class="flex items-center gap-1 text-xs border border-gray-300 px-2 py-0.5 bg-gray-50">' +
            f.name +
            ' <button type="button" class="remove-attach text-red-400 hover:text-red-600" data-id="' + f.id + '">✕</button>' +
            '</span>'
          );
        });
      }
    });
    this.value = '';
  });

  // Remover anexo
  $(document).on('click', '.remove-attach', function () {
    $(this).closest('span').remove();
  });

  // Salvar rascunho
  $('#btn-save-draft').on('click', function () {
    const data = {
      to:         $('#compose-form [name=to]').val(),
      subject:    $('#compose-form [name=subject]').val(),
      body_html:  $('#compose-body').val(),
    };
    $.post('/compose/draft', data, function () {
      const btn = $('#btn-save-draft');
      btn.text('✓ Saved').prop('disabled', true);
      setTimeout(function () { btn.text('💾 Draft').prop('disabled', false); }, 2000);
    });
  });
});
