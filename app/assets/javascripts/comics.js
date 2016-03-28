// Place all the behaviors and hooks related to the matching controller here.
// All this logic will automatically be available in application.js.

$(document).ready(function() {
  var files = [];

  var getFilename = function(data) {
    return data.files[0].name;
  };

  var cancelUpload = function(index) {
    if (files[index]) {
      files[index].jqXHR.abort();
    }
  };

  var startUpload = function(index) {
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
      var cancelButton = $('<button type="button" data-file="' + index + '">Cancel Upload</button>');
      var startButton = $('<button type="button" data-file="' + index + '">Start Upload</button>');

      cancelButton.on('click', function() {
        cancelUpload($(this).attr('data-file'));
      });

      startButton.on('click', function() {
        startUpload($(this).attr('data-file'));
      });

      var row = $('<li class="collection-item"/>');
      row.append($('<div class="filename"/>'));
      row.append($('<div class="progress"/>').append($('<div class="determinate"/>')));
      row.append($('<span class="start"/>'));
      row.append($('<span class="cancel"/>'));

      row.find('.filename').text(filename);
      row.find('.progress').replaceWith(createProgressBar(progress));
      row.find('.start').append(startButton);
      row.find('.cancel').append(cancelButton);

      row.appendTo('#upload-results');
      $('#upload-results').removeClass('hide');

      data.context = row;
      files.push(data);
    },

    done: function(e, data) {
      console.log('done', data);
    },

    progress: function(e, data) {
      data.context.removeData('retries');
      var progress = calculateProgress(data);
      data.context.find('.progress').replaceWith(createProgressBar(progress));
    },

    progressall: function(e, data) {
      var progress = calculateProgress(data);
      $('#total-progress').text(progress);
    },

    // progressall: function(e, data) {
    //   var progress = parseInt(data.loaded / data.total * 100, 10);
    //   var progressBar = $('.progress .determinate');
    //   $('.progress').removeClass('hide');
    //   progressBar.css('width', progress + '%');
    // },

    // done: function(e, data) {
    //   $.each(data.files, function(index, file) {
    //     $('#upload-results').removeClass('hide');
    //     var item = $('<li class="collection-item avatar"/>');
    //     item
    //       .append(file.name)
    //       .append($('<i class="material-icons circle green">done</i>'));

    //     $('#upload-results').append(item);
    //   });
    // },
  });

  $('#start-upload').on('click', function() {
    startAllUploads();
  });

  $('#stop-uploads').on('click', function() {
    cancelAllUploads();
  });
});
