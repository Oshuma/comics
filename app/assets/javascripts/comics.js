// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {

  // HACK: We have to wrap the Dropzone shit in this check, so it won't try to attach on other pages.
  if ($('#comic_upload').length) {
    if ($('#select_group').val() !== '') {
      enableUploadButtons();
    }

    $('#select_group').on('change', function() {
      if ($(this).val() === '') {
        disableUploadButtons();
      } else {
        enableUploadButtons();
      }
    });

    // Get the template HTML and remove it from the doument
    var previewNode = $('#template');
    previewNode.removeAttr('id');
    var previewTemplate = previewNode.parent().html();
    previewNode.remove();

    var comicUploadDropzone = new Dropzone('#comic_upload', {
      paramName: 'comic',
      thumbnailWidth: 80,
      thumbnailHeight: 80,
      parallelUploads: 3,
      previewTemplate: previewTemplate,
      autoQueue: false, // Make sure the files aren't queued until manually added
      previewsContainer: "#previews", // Define the container to display the previews
      clickable: ".fileinput-button" // Define the element that should be used as click trigger to select files.
    });

    comicUploadDropzone.on("addedfile", function(file) {
      // Hookup the start button
      $(file.previewElement).find('.start').on('click', function() { comicUploadDropzone.enqueueFile(file); });
    });

    // Update the total progress bar
    comicUploadDropzone.on("totaluploadprogress", function(progress) {
      $('#total-progress .progress-bar').css('width', progress + '%');
    });

    comicUploadDropzone.on("sending", function(file) {
      // Show the total progress bar when upload starts
      $('#total-progress').css('opacity', '1');
      // And disable the start button
      $(file.previewElement).find('.start').prop('disabled', true);
    });

    // Hide the total progress bar when nothing's uploading anymore
    comicUploadDropzone.on("queuecomplete", function(progress) {
      $('#total-progress').css('opacity', '0');
    });

    comicUploadDropzone.on("success", function(file, serverResponse) {
      if (serverResponse.id && serverResponse.url) {
        $(file.previewElement).attr('id', serverResponse.id);
        $(file.previewElement).find('.delete').attr('href', serverResponse.url);
      } else {
        if (serverResponse.errors) {
          $(file.previewElement).find('.error').html(serverResponse.errors);
        }
      }
    });

    // Setup the buttons for all transfers
    // The "add files" button doesn't need to be setup because the config
    // `clickable` has already been specified.
    $('#actions .start').on('click', function(e) {
      comicUploadDropzone.enqueueFiles(comicUploadDropzone.getFilesWithStatus(Dropzone.ADDED));
    });

    $('#actions .cancel').on('click', function() {
      comicUploadDropzone.removeAllFiles(true);
    });

    function enableUploadButtons() {
      $('#actions .start').prop('disabled', false);
      $('.file-row .start').prop('disabled', false);
    }

    function disableUploadButtons() {
      $('#actions .start').prop('disabled', true);
      $('.file-row .start').prop('disabled', true);
    }


    // autofocus new group modal's name field.
    $('#new-group-modal').on('shown.bs.modal', function() {
      $('#group_name').focus();
    });
  }

});
