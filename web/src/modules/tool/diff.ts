// Cairn — tool module: pure text diff engine.
// Line-level LCS diff; modified line pairs additionally get word-level
// highlights via a second LCS over whitespace-separated tokens.

export type RowType = 'equal' | 'delete' | 'insert' | 'modify'

export interface Span {
  t: 'same' | 'del' | 'add'
  text: string
}

export interface DiffRow {
  type: RowType
  /** 1-based line numbers in left / right text. */
  leftNo?: number
  rightNo?: number
  left?: string
  right?: string
  /** Word-level spans for modify rows. */
  leftSpans?: Span[]
  rightSpans?: Span[]
}

type LcsOp = 'eq' | 'del' | 'ins'

// Guards: keep the DP tables small so typing stays responsive on huge inputs.
const MAX_DP_CELLS = 1_000_000
const MAX_WORD_CHARS = 2000
const MAX_WORD_TOKENS = 400

export function diffTexts(a: string, b: string): DiffRow[] {
  const aLines = toLines(a)
  const bLines = toLines(b)

  // Trim common prefix and suffix lines before the expensive part.
  let start = 0
  while (start < aLines.length && start < bLines.length && aLines[start] === bLines[start]) start++

  let end = 0
  while (
    end < aLines.length - start &&
    end < bLines.length - start &&
    aLines[aLines.length - 1 - end] === bLines[bLines.length - 1 - end]
  ) {
    end++
  }

  const rows: DiffRow[] = []
  for (let k = 0; k < start; k++) {
    rows.push({ type: 'equal', leftNo: k + 1, rightNo: k + 1, left: aLines[k], right: bLines[k] })
  }

  const aMid = aLines.slice(start, aLines.length - end)
  const bMid = bLines.slice(start, bLines.length - end)
  rows.push(...diffLineRanges(aMid, bMid, aLines, bLines, start))

  for (let k = 0; k < end; k++) {
    const la = aLines.length - end + k
    const lb = bLines.length - end + k
    rows.push({ type: 'equal', leftNo: la + 1, rightNo: lb + 1, left: aLines[la], right: bLines[lb] })
  }

  return rows
}

export function diffStats(rows: DiffRow[]): { added: number; removed: number; same: number } {
  let added = 0
  let removed = 0
  let same = 0
  for (const r of rows) {
    if (r.type === 'insert' || r.type === 'modify') added++
    if (r.type === 'delete' || r.type === 'modify') removed++
    if (r.type === 'equal') same++
  }
  return { added, removed, same }
}

/** Split into lines; a trailing newline is not a phantom empty line ("a\n" === "a"). */
function toLines(s: string): string[] {
  if (s === '') return []
  const lines = s.split(/\r?\n/)
  if (lines[lines.length - 1] === '') lines.pop()
  return lines
}

function diffLineRanges(
  aMid: string[],
  bMid: string[],
  aAll: string[],
  bAll: string[],
  offset: number
): DiffRow[] {
  if (aMid.length === 0 && bMid.length === 0) return []

  let ops: LcsOp[]
  if (aMid.length === 0) {
    ops = Array.from({ length: bMid.length }, () => 'ins' as LcsOp)
  } else if (bMid.length === 0) {
    ops = Array.from({ length: aMid.length }, () => 'del' as LcsOp)
  } else {
    ops = lcsOps(aMid, bMid)
  }

  const rows: DiffRow[] = []
  let ai = offset
  let bi = offset
  let k = 0
  while (k < ops.length) {
    const t = ops[k]
    let run = 1
    while (k + run < ops.length && ops[k + run] === t) run++

    if (t === 'eq') {
      for (let r = 0; r < run; r++) {
        rows.push({ type: 'equal', leftNo: ai + r + 1, rightNo: bi + r + 1, left: aAll[ai + r], right: bAll[bi + r] })
      }
      ai += run
      bi += run
      k += run
    } else if (t === 'del') {
      // The del-first tie-break in lcsOps makes the insert run follow its delete run.
      let insRun = 0
      if (k + run < ops.length && ops[k + run] === 'ins') {
        insRun = 1
        while (k + run + insRun < ops.length && ops[k + run + insRun] === 'ins') insRun++
      }
      const paired = Math.min(run, insRun)
      for (let r = 0; r < paired; r++) {
        rows.push(modifyRow(aAll[ai + r], bAll[bi + r], ai + r + 1, bi + r + 1))
      }
      for (let r = paired; r < run; r++) {
        rows.push({ type: 'delete', leftNo: ai + r + 1, left: aAll[ai + r] })
      }
      for (let r = paired; r < insRun; r++) {
        rows.push({ type: 'insert', rightNo: bi + r + 1, right: bAll[bi + r] })
      }
      ai += run
      bi += insRun
      k += run + insRun
    } else {
      // Standalone insert run (kept for totality; normally consumed by pairing).
      for (let r = 0; r < run; r++) {
        rows.push({ type: 'insert', rightNo: bi + r + 1, right: bAll[bi + r] })
      }
      bi += run
      k += run
    }
  }

  return rows
}

function modifyRow(a: string, b: string, leftNo: number, rightNo: number): DiffRow {
  const row: DiffRow = { type: 'modify', leftNo, rightNo, left: a, right: b }

  if (a.length > MAX_WORD_CHARS || b.length > MAX_WORD_CHARS) {
    row.leftSpans = [{ t: 'del', text: a }]
    row.rightSpans = [{ t: 'add', text: b }]
    return row
  }

  const aTok = tokenize(a)
  const bTok = tokenize(b)
  if (aTok.length > MAX_WORD_TOKENS || bTok.length > MAX_WORD_TOKENS) {
    row.leftSpans = [{ t: 'del', text: a }]
    row.rightSpans = [{ t: 'add', text: b }]
    return row
  }

  const ops = lcsOps(aTok, bTok)
  row.leftSpans = spansFor(ops, aTok, bTok, 'left')
  row.rightSpans = spansFor(ops, aTok, bTok, 'right')
  return row
}

/** Whitespace is a token too, so re-joining the spans reproduces the line exactly. */
function tokenize(s: string): string[] {
  return s.split(/(\s+)/).filter((t) => t !== '')
}

function spansFor(ops: LcsOp[], x: string[], y: string[], side: 'left' | 'right'): Span[] {
  const spans: Span[] = []
  let xi = 0
  let yi = 0
  for (const op of ops) {
    let t: Span['t']
    let text: string
    if (op === 'eq') {
      t = 'same'
      text = x[xi]
      xi++
      yi++
    } else if (op === 'del') {
      t = 'del'
      text = x[xi]
      xi++
    } else {
      t = 'add'
      text = y[yi]
      yi++
    }
    if (side === 'left' && t === 'add') continue
    if (side === 'right' && t === 'del') continue

    const last = spans[spans.length - 1]
    if (last && last.t === t) {
      last.text += text
    } else {
      spans.push({ t, text })
    }
  }
  return spans
}

function lcsOps(x: string[], y: string[]): LcsOp[] {
  const m = x.length
  const n = y.length

  // Fallback without alignment info: everything deleted, then everything inserted.
  if (m * n > MAX_DP_CELLS) {
    return [
      ...Array.from({ length: m }, () => 'del' as LcsOp),
      ...Array.from({ length: n }, () => 'ins' as LcsOp),
    ]
  }

  const w = n + 1
  const dp = new Uint32Array((m + 1) * w)
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i * w + j] =
        x[i - 1] === y[j - 1]
          ? dp[(i - 1) * w + (j - 1)] + 1
          : Math.max(dp[(i - 1) * w + j], dp[i * w + (j - 1)])
    }
  }

  const ops: LcsOp[] = []
  let i = m
  let j = n
  while (i > 0 && j > 0) {
    if (x[i - 1] === y[j - 1]) {
      ops.push('eq')
      i--
      j--
    } else if (dp[i * w + (j - 1)] >= dp[(i - 1) * w + j]) {
      // Backward walk prefers ins on ties, so after reversal the forward op
      // stream emits each divergence's delete run before its insert run —
      // which is what modify pairing relies on.
      ops.push('ins')
      j--
    } else {
      ops.push('del')
      i--
    }
  }
  while (i > 0) {
    ops.push('del')
    i--
  }
  while (j > 0) {
    ops.push('ins')
    j--
  }
  ops.reverse()
  return ops
}
