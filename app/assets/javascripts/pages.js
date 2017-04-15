// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {

  // Swipe to next page.
  $('.viewer').on('swipeleft', function(e) {
    var url = $('#next_page').attr('href');
    $.ajax({
      url: url,
      method: 'PUT',
    });
  });

  // Swipe to previous page.
  $('.viewer').on('swiperight', function(e) {
    var url = $('#previous_page').attr('href');
    $.ajax({
      url: url,
      method: 'PUT',
    });
  });

});
