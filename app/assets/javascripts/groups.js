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
});
