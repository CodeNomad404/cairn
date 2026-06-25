document.addEventListener('click', function (event) {
  var replyButton = event.target.closest('[data-reply-to]');
  if (!replyButton) return;

  var form = document.getElementById('comment-form');
  var parentInput = document.getElementById('comment-parent-id');
  if (!form || !parentInput) return;

  parentInput.value = replyButton.getAttribute('data-reply-to');

  var textarea = form.querySelector('textarea');
  if (textarea) {
    textarea.placeholder = '写下回复...';
    textarea.focus();
  }
  form.scrollIntoView({ behavior: 'smooth' });
});
