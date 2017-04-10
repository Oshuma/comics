// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {

  $('#toggle-empty-groups').on('click', function(e) {
    $('.empty-group').toggleClass('hidden');
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
