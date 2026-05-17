$(function () {
  // Deletar contato com confirmação
  $(document).on('click', '.delete-contact', function () {
    if (!confirm('Remove this contact?')) return;
    const id = $(this).data('id');
    const row = $(this).closest('tr');
    $.ajax({
      url: '/contacts/' + id,
      method: 'DELETE',
      success: function () { row.fadeOut(200, function () { $(this).remove(); }); }
    });
  });
});
