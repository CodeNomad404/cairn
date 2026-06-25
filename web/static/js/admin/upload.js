document.addEventListener('DOMContentLoaded', function () {
  var dropzone = document.querySelector('[data-upload-dropzone]');
  var input = document.querySelector('[data-upload-input]');
  var hint = document.getElementById('file-name-hint');
  if (!dropzone || !input || !hint) return;

  dropzone.addEventListener('click', function () {
    input.click();
  });

  input.addEventListener('change', function () {
    hint.textContent = input.files[0] ? input.files[0].name : '支持 .md 和 .markdown 格式';
  });
});
