// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {
  $('#manage-groups').on('click', function(e) {
    $('.add-group').toggleClass('hidden');
    $('.empty-group').toggleClass('hidden');
    $('.manage-link').toggleClass('hidden');

    if ($(this).data('manage')) {
      $(this).data('manage', false);
      $(this).find('i').removeClass('fa-ban').addClass('fa-cog');
      $(this).find('span').text('Manage Groups');
    } else {
      $(this).data('manage', true);
      $(this).find('i').removeClass('fa-cog').addClass('fa-ban');
      $(this).find('span').text('Cancel');
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
