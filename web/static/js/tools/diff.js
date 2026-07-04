(function (root, factory) {
  if (typeof module === 'object' && module.exports) {
    module.exports = factory;
    return;
  }
  root.CairnDiffTool = factory(root.difflib);
})(typeof window !== 'undefined' ? window : globalThis, function (diffLib) {
  'use strict';

  function splitLines(text) {
    return text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
  }

  function isBlankInput(lines) {
    return lines.length === 1 && lines[0] === '';
  }

  function escapeHTML(value) {
    return String(value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function visibleWhitespace(value) {
    var chars = Array.from(String(value));
    var trailingStart = chars.length;

    while (trailingStart > 0 && chars[trailingStart - 1] === ' ') {
      trailingStart -= 1;
    }

    return chars.map(function (char, index) {
      if (char === '\t') {
        return '<span class="diff-ws" title="tab">→</span>';
      }
      if (char === ' ' && index >= trailingStart) {
        return '<span class="diff-ws" title="trailing space">·</span>';
      }
      if (char === ' ' && (chars[index - 1] === ' ' || chars[index + 1] === ' ')) {
        return '<span class="diff-ws" title="space">·</span>';
      }
      return escapeHTML(char);
    }).join('');
  }

  function tokenize(line) {
    if (!line) return [];
    return line.match(/\s+|[\u4e00-\u9fff]|[A-Za-z0-9_]+|[^\sA-Za-z0-9_\u4e00-\u9fff]+/g) || [];
  }

  function sequenceMatcher(a, b) {
    return new diffLib.SequenceMatcher(a, b);
  }

  function inlinePieces(left, right) {
    var leftTokens = tokenize(left);
    var rightTokens = tokenize(right);
    var matcher = sequenceMatcher(leftTokens, rightTokens);
    var ops = matcher.get_opcodes();
    var leftHTML = '';
    var rightHTML = '';

    ops.forEach(function (op) {
      var tag = op[0];
      var i1 = op[1];
      var i2 = op[2];
      var j1 = op[3];
      var j2 = op[4];
      var leftText = leftTokens.slice(i1, i2).join('');
      var rightText = rightTokens.slice(j1, j2).join('');

      if (tag === 'equal') {
        leftHTML += visibleWhitespace(leftText);
        rightHTML += visibleWhitespace(rightText);
        return;
      }
      if (tag === 'delete') {
        leftHTML += wrapInline('diff-inline-del', leftText);
        return;
      }
      if (tag === 'insert') {
        rightHTML += wrapInline('diff-inline-add', rightText);
        return;
      }
      leftHTML += wrapInline('diff-inline-del', leftText);
      rightHTML += wrapInline('diff-inline-add', rightText);
    });

    return { left: leftHTML, right: rightHTML };
  }

  function wrapInline(className, text) {
    if (!text) return '';
    return '<span class="' + className + '">' + visibleWhitespace(text) + '</span>';
  }

  function cell(className, html) {
    return '<td class="' + className + '">' + html + '</td>';
  }

  function row(left, right) {
    return '<tr>' + left.join('') + right.join('') + '</tr>';
  }

  function lineNumber(value) {
    return value == null ? '' : String(value);
  }

  function lineCells(side, lineNo, sign, codeHTML, stateClass) {
    var signClass = sign === '+' ? 'diff-add-sign' : sign === '-' ? 'diff-del-sign' : '';
    var classes = stateClass ? ' ' + stateClass : '';

    return [
      cell('gutter' + classes, lineNumber(lineNo)),
      cell('sign' + classes + (signClass ? ' ' + signClass : ''), sign),
      cell('code diff-code-' + side + classes, codeHTML),
    ];
  }

  function emptyCells(side, stateClass) {
    return lineCells(side, null, '', '', stateClass);
  }

  function calculateStats(ops) {
    var stats = { additions: 0, deletions: 0 };

    ops.forEach(function (op) {
      var tag = op[0];
      if (tag === 'insert') stats.additions += op[4] - op[3];
      if (tag === 'delete') stats.deletions += op[2] - op[1];
      if (tag === 'replace') {
        stats.deletions += op[2] - op[1];
        stats.additions += op[4] - op[3];
      }
    });

    return stats;
  }

  function renderStats(stats) {
    return [
      '<span class="diff-stat-add">+' + stats.additions + '</span>',
      '<span class="diff-stat-del">-' + stats.deletions + '</span>',
    ].join(' ');
  }

  function renderRows(leftLines, rightLines, ops) {
    var html = '';

    ops.forEach(function (op) {
      var tag = op[0];
      var i1 = op[1];
      var i2 = op[2];
      var j1 = op[3];
      var j2 = op[4];

      if (tag === 'equal') {
        for (var equalIndex = 0; equalIndex < i2 - i1; equalIndex += 1) {
          html += row(
            lineCells('left', i1 + equalIndex + 1, '', visibleWhitespace(leftLines[i1 + equalIndex]), 'diff-muted'),
            lineCells('right', j1 + equalIndex + 1, '', visibleWhitespace(rightLines[j1 + equalIndex]), 'diff-muted')
          );
        }
        return;
      }

      if (tag === 'delete') {
        for (var deleteIndex = 0; deleteIndex < i2 - i1; deleteIndex += 1) {
          html += row(
            lineCells('left', i1 + deleteIndex + 1, '-', visibleWhitespace(leftLines[i1 + deleteIndex]), 'diff-deleted'),
            emptyCells('right', 'diff-empty diff-deleted')
          );
        }
        return;
      }

      if (tag === 'insert') {
        for (var insertIndex = 0; insertIndex < j2 - j1; insertIndex += 1) {
          html += row(
            emptyCells('left', 'diff-empty diff-added'),
            lineCells('right', j1 + insertIndex + 1, '+', visibleWhitespace(rightLines[j1 + insertIndex]), 'diff-added')
          );
        }
        return;
      }

      if (tag === 'replace') {
        html += renderReplaceRows(leftLines, rightLines, i1, i2, j1, j2);
      }
    });

    return html;
  }

  function renderReplaceRows(leftLines, rightLines, i1, i2, j1, j2) {
    var html = '';
    var max = Math.max(i2 - i1, j2 - j1);

    for (var index = 0; index < max; index += 1) {
      var hasLeft = i1 + index < i2;
      var hasRight = j1 + index < j2;
      var leftLine = hasLeft ? leftLines[i1 + index] : '';
      var rightLine = hasRight ? rightLines[j1 + index] : '';
      var inline = hasLeft && hasRight ? inlinePieces(leftLine, rightLine) : null;

      html += row(
        hasLeft
          ? lineCells('left', i1 + index + 1, '-', inline ? inline.left : visibleWhitespace(leftLine), 'diff-deleted')
          : emptyCells('left', 'diff-empty diff-deleted'),
        hasRight
          ? lineCells('right', j1 + index + 1, '+', inline ? inline.right : visibleWhitespace(rightLine), 'diff-added')
          : emptyCells('right', 'diff-empty diff-added')
      );
    }

    return html;
  }

  function buildDiff(leftText, rightText) {
    var leftLines = splitLines(leftText);
    var rightLines = splitLines(rightText);
    var empty = isBlankInput(leftLines) && isBlankInput(rightLines);

    if (empty) {
      return { empty: true, same: true, stats: { additions: 0, deletions: 0 }, rowsHTML: '', statsHTML: renderStats({ additions: 0, deletions: 0 }) };
    }

    var ops = sequenceMatcher(leftLines, rightLines).get_opcodes();
    var stats = calculateStats(ops);
    var same = stats.additions === 0 && stats.deletions === 0;

    return {
      empty: false,
      same: same,
      stats: stats,
      rowsHTML: renderRows(leftLines, rightLines, ops),
      statsHTML: renderStats(stats),
    };
  }

  function syncLineNumbers(textarea, numsEl) {
    if (!window.CairnLineNumbers) {
      throw new Error('CairnLineNumbers is required before diff.js');
    }
    window.CairnLineNumbers.bind(textarea, numsEl);
  }

  function bindDOM() {
    if (typeof document === 'undefined') return;

    var leftInput = document.getElementById('left-input');
    var rightInput = document.getElementById('right-input');
    var result = document.getElementById('diff-result');
    var output = document.getElementById('diff-output');
    var stats = document.getElementById('diff-stats');
    var emptyState = document.getElementById('diff-empty-state');

    function render() {
      var diff = buildDiff(leftInput.value, rightInput.value);

      if (diff.empty) {
        output.innerHTML = '';
        stats.innerHTML = '';
        result.classList.add('hidden');
        result.classList.remove('flex');
        return;
      }

      output.innerHTML = diff.rowsHTML;
      stats.innerHTML = diff.same ? diff.statsHTML + ' <span class="diff-stat-same">无差异</span>' : diff.statsHTML;
      emptyState.classList.toggle('hidden', !diff.same);
      result.classList.remove('hidden');
      result.classList.add('flex');
    }

    syncLineNumbers(leftInput, document.getElementById('left-nums'));
    syncLineNumbers(rightInput, document.getElementById('right-nums'));

    document.querySelector('[data-diff-run]').addEventListener('click', render);
    document.querySelector('[data-diff-clear]').addEventListener('click', function () {
      leftInput.value = '';
      rightInput.value = '';
      output.innerHTML = '';
      stats.innerHTML = '';
      result.classList.add('hidden');
      result.classList.remove('flex');
      leftInput.dispatchEvent(new Event('input'));
      rightInput.dispatchEvent(new Event('input'));
    });
    document.querySelector('[data-diff-swap]').addEventListener('click', function () {
      var nextLeft = rightInput.value;
      rightInput.value = leftInput.value;
      leftInput.value = nextLeft;
      leftInput.dispatchEvent(new Event('input'));
      rightInput.dispatchEvent(new Event('input'));
      render();
    });
  }

  if (typeof document !== 'undefined') {
    document.addEventListener('DOMContentLoaded', bindDOM);
  }

  return {
    buildDiff: buildDiff,
    escapeHTML: escapeHTML,
    inlinePieces: inlinePieces,
    splitLines: splitLines,
    visibleWhitespace: visibleWhitespace,
  };
});
