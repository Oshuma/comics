// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).ready(function() {
  $('#manage-groups').on('click', function(e) {
    $('.add-group').toggleClass('hide');
    $('.empty-group').toggleClass('hide');
    $('.manage-link').toggleClass('hide');

    if ($(this).data('manage')) {
      $(this).data('manage', false);
      $(this).text('Manage Groups');
    } else {
      $(this).data('manage', true);
      $(this).text('Cancel');
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
});
