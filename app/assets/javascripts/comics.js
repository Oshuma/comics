// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).ready(function() {
  $('#comic-upload').fileupload({
    limitConcurrentUploads: 3,

    progressall: function(e, data) {
      var progress = parseInt(data.loaded / data.total * 100, 10);
      var progressBar = $('.progress .determinate');
      $('.progress').removeClass('hide');
      progressBar.css('width', progress + '%');
    },

    done: function(e, data) {
      $.each(data.files, function(index, file) {
        $('#upload-results').removeClass('hide');
        var item = $('<li class="collection-item avatar"/>');
        item
          .append(file.name)
          .append($('<i class="material-icons circle green">done</i>'));

        $('#upload-results').append(item);
      });
    },
  });
});
