// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).ready(function() {
  var files = [];

  var getFilename = function(data) {
    return data.files[0].name;
  };

  var cancelUpload = function(index) {
    if (files[index]) {
      if (files[index].jqXHR) files[index].jqXHR.abort();
      files[index].context.fadeOut();
      files[index] = null;
    }
  };

  var startUpload = function(index) {
    if (files[index] == null) return;

    var data = files[index];
    var context = data.context;

    data.uploadedBytes = parseInt($(context).attr('uploadedBytes'), 10);
    data.data = null;
    $(data).submit();
  };

  var cancelAllUploads = function() {
    $(files).each(function(index, file) {
      cancelUpload(index);
    });

    $('#upload-results').addClass('hide');
  };

  var startAllUploads = function() {
    $(files).each(function(index, file) {
      startUpload(index);
    });
  };

  var calculateProgress = function(data) {
    var value = parseInt(data.loaded / data.total * 100, 10) || 0;
    return value + '%';
  };

  var createProgressBar = function(progress) {
    var progressBar = $('<div class="progress"/>');
    progressBar.append($('<div class="determinate" style="width: ' + progress + ';"/>'));
    return progressBar;
  };

  $('#comic-upload').fileupload({
    limitConcurrentUploads: 3,

    add: function(e, data) {
      var progress = calculateProgress(data);
      var filename = getFilename(data);

      var index = $('#upload-results li').length;
      var startButton = $('<button type="button" class="btn" data-file="' + index + '">Start</button>');
      var cancelButton = $('<button type="button" class="btn grey" data-file="' + index + '">Cancel</button>');

      startButton.on('click', function() {
        startUpload($(this).attr('data-file'));
      });

      cancelButton.on('click', function() {
        cancelUpload($(this).attr('data-file'));
      });

      var row = $('<li class="collection-item"/>');
      row.append($('<div class="filename"/>'));
      row.append($('<div class="progress"/>').append($('<div class="determinate"/>')));
      row.append($('<div class="actions"/>'));

      row.find('.filename').text(filename);
      row.find('.progress').replaceWith(createProgressBar(progress));
      row.find('.actions').append(startButton);
      row.find('.actions').append('&nbsp;');
      row.find('.actions').append(cancelButton);

      row.appendTo('#upload-results');
      $('#upload-results').removeClass('hide');

      data.context = row;
      files.push(data);
    },

    done: function(e, data) {
      data.context.find('.actions').remove();
      data.context.find('.progress').after('<i class="material-icons green-text">done</i>');
    },

    progress: function(e, data) {
      data.context.removeData('retries');
      var progress = calculateProgress(data);
      data.context.find('.progress').replaceWith(createProgressBar(progress));
    },

    progressall: function(e, data) {
      var progress = calculateProgress(data);
      $('#total-progress').css('width', progress);
    },
  });

  $('#start-upload').on('click', function() {
    startAllUploads();
  });

  $('#stop-uploads').on('click', function() {
    cancelAllUploads();
  });
});
