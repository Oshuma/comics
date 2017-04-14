// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).on('turbolinks:load', function() {

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
  var previewNode = document.querySelector("#template");
  previewNode.id = "";
  var previewTemplate = previewNode.parentNode.innerHTML;
  previewNode.parentNode.removeChild(previewNode);

  var myDropzone = new Dropzone('#comic_upload', {
    paramName: 'comic',
    thumbnailWidth: 80,
    thumbnailHeight: 80,
    parallelUploads: 5,
    previewTemplate: previewTemplate,
    autoQueue: false, // Make sure the files aren't queued until manually added
    previewsContainer: "#previews", // Define the container to display the previews
    clickable: ".fileinput-button" // Define the element that should be used as click trigger to select files.
  });

  myDropzone.on("addedfile", function(file) {
    // Hookup the start button
    file.previewElement.querySelector(".start").onclick = function() { myDropzone.enqueueFile(file); };
  });

  // Update the total progress bar
  myDropzone.on("totaluploadprogress", function(progress) {
    document.querySelector("#total-progress .progress-bar").style.width = progress + "%";
  });

  myDropzone.on("sending", function(file) {
    // Show the total progress bar when upload starts
    document.querySelector("#total-progress").style.opacity = "1";
    // And disable the start button
    file.previewElement.querySelector(".start").setAttribute("disabled", "disabled");
  });

  // Hide the total progress bar when nothing's uploading anymore
  myDropzone.on("queuecomplete", function(progress) {
    document.querySelector("#total-progress").style.opacity = "0";
  });

  myDropzone.on("success", function(file, serverResponse) {
    if (serverResponse.url) {
      file.previewElement.querySelector(".delete").setAttribute("href", serverResponse.url);
    } else {
      if (serverResponse.errors) {
        file.previewElement.querySelector('.error').innerHTML = serverResponse.errors;
      }
    }
  });

  // Setup the buttons for all transfers
  // The "add files" button doesn't need to be setup because the config
  // `clickable` has already been specified.
  document.querySelector("#actions .start").onclick = function() {
    myDropzone.enqueueFiles(myDropzone.getFilesWithStatus(Dropzone.ADDED));
  };
  document.querySelector("#actions .cancel").onclick = function() {
    myDropzone.removeAllFiles(true);
  };

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

});
