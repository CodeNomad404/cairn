(function (root) {
  'use strict';

  function bind(textarea, gutter) {
    var mirror = document.createElement('div');
    mirror.setAttribute('aria-hidden', 'true');
    Object.assign(mirror.style, {
      position: 'fixed',
      left: '-10000px',
      top: '0',
      visibility: 'hidden',
      whiteSpace: 'pre-wrap',
      overflowWrap: 'break-word',
      wordBreak: 'normal',
      boxSizing: 'content-box',
      padding: '0',
      border: '0',
    });
    document.body.appendChild(mirror);

    function update() {
      var style = window.getComputedStyle(textarea);
      var contentWidth = textarea.clientWidth
        - parseFloat(style.paddingLeft)
        - parseFloat(style.paddingRight);

      mirror.style.width = Math.max(contentWidth, 1) + 'px';
      mirror.style.fontFamily = style.fontFamily;
      mirror.style.fontSize = style.fontSize;
      mirror.style.fontStyle = style.fontStyle;
      mirror.style.fontWeight = style.fontWeight;
      mirror.style.fontStretch = style.fontStretch;
      mirror.style.fontVariant = style.fontVariant;
      mirror.style.letterSpacing = style.letterSpacing;
      mirror.style.lineHeight = style.lineHeight;
      mirror.style.tabSize = style.tabSize;

      gutter.style.paddingTop = style.paddingTop;
      gutter.style.paddingBottom = style.paddingBottom;
      gutter.style.fontFamily = style.fontFamily;
      gutter.style.fontSize = style.fontSize;
      gutter.style.fontWeight = style.fontWeight;
      gutter.style.lineHeight = style.lineHeight;

      var lineHeight = parseFloat(style.lineHeight);
      var lines = textarea.value.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
      var fragment = document.createDocumentFragment();

      lines.forEach(function (line, index) {
        mirror.textContent = line || '\u200b';
        var number = document.createElement('span');
        var visualLines = Math.max(Math.round(mirror.getBoundingClientRect().height / lineHeight), 1);
        number.textContent = index + 1;
        number.style.height = visualLines * lineHeight + 'px';
        fragment.appendChild(number);
      });

      gutter.replaceChildren(fragment);
      gutter.scrollTop = textarea.scrollTop;
    }

    textarea.addEventListener('input', update);
    textarea.addEventListener('scroll', function () {
      gutter.scrollTop = textarea.scrollTop;
    });

    var resizeObserver = new ResizeObserver(update);
    resizeObserver.observe(textarea);
    update();

    return { update: update };
  }

  root.CairnLineNumbers = { bind: bind };
})(window);
