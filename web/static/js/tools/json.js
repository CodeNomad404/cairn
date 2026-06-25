function updateJSONLineNumbers(textarea, lineNumbers) {
  var count = Math.max(textarea.value.split('\n').length, 1);
  lineNumbers.textContent = Array.from({ length: count }, function (_, index) {
    return index + 1;
  }).join('\n');
}

function setJSONPreviewWaiting(preview, message) {
  preview.innerHTML = '<p class="mt-20 text-center text-gray-400">' + message + '</p>';
}

function decorateJSONPreview(preview) {
  preview.querySelectorAll('.json-formatter-constructor-name').forEach(function (element) {
    element.style.display = 'inline';
    var text = element.textContent;
    if (text === 'Object') {
      element.textContent = '{...}';
      return;
    }
    if (text.indexOf('Array') === 0) {
      var match = text.match(/\d+/);
      element.textContent = '[' + (match ? match[0] : '') + ']';
    }
  });
}

document.addEventListener('DOMContentLoaded', function () {
  var textarea = document.getElementById('json-input');
  var lineNumbers = document.getElementById('json-line-numbers');
  var preview = document.getElementById('json-preview');
  var error = document.getElementById('input-error');
  var handle = document.getElementById('json-drag-handle');
  var left = document.getElementById('json-panel-left');
  var right = document.getElementById('json-panel-right');
  var panels = document.getElementById('json-panels');
  var debounceTimer;
  var dragging = false;
  var startX = 0;
  var startLeftWidth = 0;

  function formatJSON() {
    var input = textarea.value.trim();
    if (!input) {
      setJSONPreviewWaiting(preview, '粘贴 JSON 后点击格式化');
      error.classList.add('hidden');
      return;
    }

    try {
      var parsed = JSON.parse(input);
      preview.innerHTML = '';
      var formatter = new JSONFormatter(parsed, 2, {
        hoverPreviewEnabled: false,
        animateOpen: true,
        animateClose: true,
      });
      preview.appendChild(formatter.render());
      formatter.openAtDepth(3);
      decorateJSONPreview(preview);
      error.classList.add('hidden');
    } catch (err) {
      setJSONPreviewWaiting(preview, '等待输入...');
      error.textContent = 'JSON 格式错误：' + err.message;
      error.classList.remove('hidden');
    }
  }

  function clearAll() {
    textarea.value = '';
    setJSONPreviewWaiting(preview, '粘贴 JSON 后点击格式化');
    error.classList.add('hidden');
    updateJSONLineNumbers(textarea, lineNumbers);
  }

  function formattedInput() {
    var input = textarea.value.trim();
    if (!input) return '';
    return JSON.stringify(JSON.parse(input), null, 2);
  }

  function copyJSON(event) {
    try {
      var formatted = formattedInput();
      if (!formatted) return;
      navigator.clipboard.writeText(formatted).then(function () {
        var button = event.target;
        var original = button.textContent;
        button.textContent = '已复制';
        window.setTimeout(function () {
          button.textContent = original;
        }, 1500);
      });
    } catch (err) {
      return;
    }
  }

  function exportJSON() {
    try {
      var formatted = formattedInput();
      if (!formatted) return;
      var blob = new Blob([formatted], { type: 'application/json' });
      var url = URL.createObjectURL(blob);
      var anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = 'cairn-export.json';
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      return;
    }
  }

  textarea.addEventListener('input', function () {
    updateJSONLineNumbers(textarea, lineNumbers);
    window.clearTimeout(debounceTimer);
    debounceTimer = window.setTimeout(formatJSON, 500);
  });
  textarea.addEventListener('scroll', function () {
    lineNumbers.scrollTop = textarea.scrollTop;
  });

  document.querySelector('[data-json-format]').addEventListener('click', formatJSON);
  document.querySelector('[data-json-clear]').addEventListener('click', clearAll);
  document.querySelector('[data-json-copy]').addEventListener('click', copyJSON);
  document.querySelector('[data-json-export]').addEventListener('click', exportJSON);

  handle.addEventListener('mousedown', function (event) {
    dragging = true;
    startX = event.clientX;
    startLeftWidth = left.getBoundingClientRect().width;
    handle.classList.add('active');
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  });

  document.addEventListener('mousemove', function (event) {
    if (!dragging) return;
    var nextLeft = startLeftWidth + event.clientX - startX;
    var containerWidth = panels.getBoundingClientRect().width;
    var minWidth = 300;
    var maxWidth = containerWidth - minWidth - 6;
    var clamped = Math.max(minWidth, Math.min(maxWidth, nextLeft));
    left.style.flex = 'none';
    left.style.width = clamped + 'px';
    right.style.flex = '1';
  });

  document.addEventListener('mouseup', function () {
    if (!dragging) return;
    dragging = false;
    handle.classList.remove('active');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  });

  updateJSONLineNumbers(textarea, lineNumbers);
});
