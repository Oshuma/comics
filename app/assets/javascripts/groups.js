// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).ready(function() {
  $('#manage-groups').on('click', function(e) {
    $('.add-group').toggleClass('hide');
    $('.delete-group').toggleClass('hide');
    $('.empty-group').toggleClass('hide');

    if ($(this).data('manage')) {
      $(this).data('manage', false);
      $(this).text('Manage Groups');
    } else {
      $('#group_name').focus();
      $(this).data('manage', true);
      $(this).text('Cancel');
    }

    e.preventDefault();
  });
});
