// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {

  $('#toggle-empty-groups').on('click', function(e) {
    var label = $(this).find('span');

    if ($(this).data('hidden-groups-visible') === 'true') {
      $(this).find('i.fa').removeClass('fa-toggle-on').addClass('fa-toggle-off');
      label.text(label.text().replace(/Hide/, 'Show'));
      $(this).data('hidden-groups-visible', 'false');
      $('.empty-group').addClass('hidden');
    } else {
      $(this).find('i.fa').removeClass('fa-toggle-off').addClass('fa-toggle-on');
      label.text(label.text().replace(/Show/, 'Hide'));
      $(this).data('hidden-groups-visible', 'true');
      $('.empty-group').removeClass('hidden');
    }
    e.preventDefault();
  });

  $('.move-to-group').on('change', function() {
    var submit_button = $(this).closest('form').find(':submit');

    if ($(this).val() == '') {
      submit_button.closest('.btn').addClass('disabled');
      submit_button.prop('disabled', true);
    } else {
      submit_button.closest('.btn').removeClass('disabled');
      submit_button.prop('disabled', false);
    }
  });

  // autofocus and select text in group modal's name field.
  $('.edit-group-modal').on('shown.bs.modal', function() {
    $(this).find('#group_name').select();
  });

});
