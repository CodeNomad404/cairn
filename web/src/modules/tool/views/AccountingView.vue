<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { APIRequestError } from '../../../shared/api/http'
import {
  createAccountingBook,
  createAccountingExpense,
  deleteAccountingBook,
  deleteAccountingExpense,
  getAccountingBook,
  listAccountingBooks,
  listAccountingExpenses,
  updateAccountingBook,
  updateAccountingExpense,
} from '../api/accounting'
import type {
  AccountingBookDetail,
  AccountingBookSummary,
  AccountingExpense,
  AccountingExpenseCategory,
} from '../types'

const categories: Array<{ id: AccountingExpenseCategory; label: string; icon: string }> = [
  { id: 1, label: '餐饮', icon: '食' },
  { id: 2, label: '交通', icon: '行' },
  { id: 3, label: '住宿', icon: '住' },
  { id: 4, label: '购物', icon: '购' },
  { id: 5, label: '票务', icon: '票' },
  { id: 6, label: '娱乐', icon: '乐' },
  { id: 7, label: '医疗', icon: '医' },
  { id: 8, label: '其他', icon: '其' },
]

const categoryMap = new Map(categories.map((item) => [item.id, item]))
const books = ref<AccountingBookSummary[]>([])
const selectedBookID = ref<number | null>(null)
const bookDetail = ref<AccountingBookDetail | null>(null)
const expenses = ref<AccountingExpense[]>([])
const page = ref(1)
const hasMore = ref(false)
const booksLoading = ref(true)
const detailLoading = ref(false)
const expenseSaving = ref(false)
const bookSaving = ref(false)
const error = ref('')

const showBookForm = ref(false)
const editingBook = ref(false)
const bookForm = reactive({ name: '', description: '' })

const editingExpenseID = ref<number | null>(null)
const showExpenseForm = ref(false)
const expenseForm = reactive({
  amount: '',
  category: 1 as AccountingExpenseCategory,
  spentAt: localDateTimeValue(),
  note: '',
})
const expenseFormError = ref('')

function localDateTimeValue(value = new Date()): string {
  const local = new Date(value.getTime() - value.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function formatMoney(cents: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
  }).format(cents / 100)
}

function formatExpenseTime(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(value))
}

function categoryLabel(category: AccountingExpenseCategory): string {
  return categoryMap.get(category)?.label ?? '其他'
}

function friendlyError(reason: unknown, fallback: string): string {
  if (reason instanceof APIRequestError) {
    if (reason.status === 401) return '登录状态已失效，请重新登录。'
    return reason.message
  }
  return fallback
}

function parseAmountToCents(value: string): number | null {
  const normalized = value.trim()
  if (!/^\d+(?:\.\d{1,2})?$/.test(normalized)) return null
  const [yuan, fraction = ''] = normalized.split('.')
  const cents = Number(yuan) * 100 + Number(fraction.padEnd(2, '0'))
  return Number.isSafeInteger(cents) && cents > 0 ? cents : null
}

async function loadBooks(preferredID?: number) {
  booksLoading.value = true
  error.value = ''
  try {
    const result = await listAccountingBooks()
    books.value = result.items
    const nextID = preferredID && result.items.some((book) => book.id === preferredID)
      ? preferredID
      : result.items[0]?.id ?? null
    selectedBookID.value = nextID
    if (nextID) await loadBook(nextID, 1)
    else {
      bookDetail.value = null
      expenses.value = []
    }
  } catch (reason) {
    error.value = friendlyError(reason, '账本加载失败，请稍后重试。')
  } finally {
    booksLoading.value = false
  }
}

async function loadBook(bookID: number, nextPage = 1) {
  selectedBookID.value = bookID
  detailLoading.value = true
  error.value = ''
  try {
    const [detail, expenseList] = await Promise.all([
      getAccountingBook(bookID),
      listAccountingExpenses(bookID, nextPage),
    ])
    if (selectedBookID.value !== bookID) return
    bookDetail.value = detail
    expenses.value = expenseList.items
    page.value = expenseList.page
    hasMore.value = expenseList.has_more
    cancelExpenseEdit()
  } catch (reason) {
    if (selectedBookID.value === bookID) {
      error.value = friendlyError(reason, '账本明细加载失败，请稍后重试。')
    }
  } finally {
    if (selectedBookID.value === bookID) detailLoading.value = false
  }
}

function openCreateBook() {
  editingBook.value = false
  bookForm.name = ''
  bookForm.description = ''
  showBookForm.value = true
}

function openEditBook() {
  if (!bookDetail.value) return
  editingBook.value = true
  bookForm.name = bookDetail.value.name
  bookForm.description = bookDetail.value.description
  showBookForm.value = true
}

async function saveBook() {
  const name = bookForm.name.trim()
  if (!name) {
    error.value = '请输入账本名称。'
    return
  }
  bookSaving.value = true
  error.value = ''
  try {
    const saved = editingBook.value && selectedBookID.value
      ? await updateAccountingBook(selectedBookID.value, {
        name,
        description: bookForm.description.trim(),
      })
      : await createAccountingBook({ name, description: bookForm.description.trim() })
    showBookForm.value = false
    await loadBooks(saved.id)
  } catch (reason) {
    error.value = friendlyError(reason, '账本保存失败，请稍后重试。')
  } finally {
    bookSaving.value = false
  }
}

async function removeBook() {
  if (!bookDetail.value) return
  const confirmed = window.confirm(`确定删除“${bookDetail.value.name}”吗？其中的全部支出也会永久删除。`)
  if (!confirmed) return
  error.value = ''
  try {
    await deleteAccountingBook(bookDetail.value.id)
    await loadBooks()
  } catch (reason) {
    error.value = friendlyError(reason, '账本删除失败，请稍后重试。')
  }
}

function resetExpenseForm() {
  editingExpenseID.value = null
  expenseForm.amount = ''
  expenseForm.category = 1
  expenseForm.spentAt = localDateTimeValue()
  expenseForm.note = ''
  expenseFormError.value = ''
}

function cancelExpenseEdit() {
  resetExpenseForm()
  showExpenseForm.value = false
}

function openCreateExpense() {
  resetExpenseForm()
  showExpenseForm.value = true
}

function editExpense(expense: AccountingExpense) {
  editingExpenseID.value = expense.id
  expenseForm.amount = (expense.amount_cents / 100).toFixed(2)
  expenseForm.category = expense.category
  expenseForm.spentAt = localDateTimeValue(new Date(expense.spent_at))
  expenseForm.note = expense.note
  expenseFormError.value = ''
  showExpenseForm.value = true
}

async function saveExpense() {
  if (!selectedBookID.value) return
  const amountCents = parseAmountToCents(expenseForm.amount)
  if (amountCents === null) {
    expenseFormError.value = '请输入大于 0、最多两位小数的金额。'
    return
  }
  const spentAt = new Date(expenseForm.spentAt)
  if (Number.isNaN(spentAt.getTime())) {
    expenseFormError.value = '请选择支出时间。'
    return
  }
  expenseSaving.value = true
  expenseFormError.value = ''
  const bookID = selectedBookID.value
  const input = {
    amount_cents: amountCents,
    category: expenseForm.category,
    spent_at: spentAt.toISOString(),
    note: expenseForm.note.trim(),
  }
  try {
    if (editingExpenseID.value) await updateAccountingExpense(editingExpenseID.value, input)
    else await createAccountingExpense(bookID, input)
    cancelExpenseEdit()
    await loadBooks(bookID)
  } catch (reason) {
    expenseFormError.value = friendlyError(reason, '支出保存失败，请稍后重试。')
  } finally {
    expenseSaving.value = false
  }
}

async function removeExpense(expense: AccountingExpense) {
  if (!window.confirm(`确定删除这笔 ${formatMoney(expense.amount_cents)} 的支出吗？`)) return
  const bookID = selectedBookID.value
  if (!bookID) return
  error.value = ''
  try {
    await deleteAccountingExpense(expense.id)
    await loadBooks(bookID)
  } catch (reason) {
    error.value = friendlyError(reason, '支出删除失败，请稍后重试。')
  }
}

onMounted(() => void loadBooks())
</script>

<template>
  <div class="page">
    <main class="tool-page accounting-page">
      <RouterLink class="back" to="/tools">← 返回工具</RouterLink>

      <header class="accounting-heading">
        <div>
          <span class="accounting-eyebrow">个人账本</span>
          <h1>每一程，都有一本账</h1>
          <p class="desc">用具体事务整理支出，金额统一按人民币记录。</p>
        </div>
        <div class="heading-actions">
          <RouterLink class="statistic-link" to="/tools/accounting/statistics">月度统计</RouterLink>
          <button class="primary-button" type="button" @click="openCreateBook">＋ 新建账本</button>
        </div>
      </header>

      <p v-if="error" class="accounting-alert" role="alert">{{ error }}</p>

      <div class="accounting-workspace">
        <aside class="book-panel">
          <div class="panel-title">
            <span>我的账本</span>
            <small>{{ books.length }} 本</small>
          </div>
          <div v-if="booksLoading && !books.length" class="accounting-loading"><span class="spinner" /></div>
          <div v-else-if="books.length" class="book-list">
            <button
              v-for="book in books"
              :key="book.id"
              type="button"
              class="book-item"
              :class="{ active: selectedBookID === book.id }"
              @click="loadBook(book.id)"
            >
              <span>
                <strong>{{ book.name }}</strong>
                <small>{{ book.expense_count }} 笔支出</small>
              </span>
              <b>{{ formatMoney(book.total_amount_cents) }}</b>
            </button>
          </div>
          <div v-else class="book-empty">
            <span>空白页</span>
            <p>新建一本账，开始记录这次行程。</p>
            <button type="button" @click="openCreateBook">新建第一本账</button>
          </div>
        </aside>

        <section class="ledger-panel">
          <div v-if="detailLoading && !bookDetail" class="accounting-loading large">
            <span class="spinner" /> 正在翻开账本
          </div>
          <template v-else-if="bookDetail">
            <header class="ledger-heading">
              <div>
                <span>当前账本</span>
                <h2>{{ bookDetail.name }}</h2>
                <p v-if="bookDetail.description">{{ bookDetail.description }}</p>
              </div>
              <div class="ledger-actions">
                <button type="button" @click="openEditBook">编辑</button>
                <button type="button" class="danger-link" @click="removeBook">删除</button>
              </div>
            </header>

            <div class="book-overview">
              <div><small>累计支出</small><strong>{{ formatMoney(bookDetail.total_amount_cents) }}</strong></div>
              <div><small>支出笔数</small><strong>{{ bookDetail.expense_count }}</strong></div>
              <div><small>主要分类</small><strong>{{ bookDetail.categories[0] ? categoryLabel(bookDetail.categories[0].category) : '—' }}</strong></div>
            </div>

            <section class="expense-section" aria-labelledby="expense-list-title">
              <div class="panel-title">
                <div>
                  <span id="expense-list-title">支出明细</span>
                  <small>第 {{ page }} 页</small>
                </div>
                <button class="add-expense-button" type="button" @click="openCreateExpense">＋ 新增支出</button>
              </div>
              <div v-if="detailLoading" class="accounting-loading"><span class="spinner" /></div>
              <div v-else-if="expenses.length" class="expense-list">
                <article v-for="expense in expenses" :key="expense.id" class="expense-item">
                  <span class="expense-category">{{ categoryMap.get(expense.category)?.icon }}</span>
                  <div class="expense-copy">
                    <strong>{{ expense.note || categoryLabel(expense.category) }}</strong>
                    <small>{{ categoryLabel(expense.category) }} · {{ formatExpenseTime(expense.spent_at) }}</small>
                  </div>
                  <b>{{ formatMoney(expense.amount_cents) }}</b>
                  <div class="expense-actions">
                    <button type="button" @click="editExpense(expense)">编辑</button>
                    <button type="button" @click="removeExpense(expense)">删除</button>
                  </div>
                </article>
              </div>
              <p v-else class="compact-empty">还没有支出，先记下第一笔吧。</p>
              <div v-if="page > 1 || hasMore" class="pagination">
                <button type="button" :disabled="page <= 1" @click="loadBook(bookDetail.id, page - 1)">上一页</button>
                <span>{{ page }}</span>
                <button type="button" :disabled="!hasMore" @click="loadBook(bookDetail.id, page + 1)">下一页</button>
              </div>
            </section>
          </template>
          <div v-else class="ledger-empty">
            <span>从一本账开始</span>
            <h2>把旅途里的每一笔，记在它该在的地方。</h2>
            <button class="primary-button" type="button" @click="openCreateBook">新建账本</button>
          </div>
        </section>
      </div>

      <div v-if="showExpenseForm" class="modal-backdrop" @click.self="cancelExpenseEdit">
        <form class="expense-modal expense-editor" @submit.prevent="saveExpense">
          <div class="modal-heading">
            <div>
              <span>{{ editingExpenseID ? '修改记录' : '记一笔' }}</span>
              <h2>{{ editingExpenseID ? '编辑支出' : '新增支出' }}</h2>
            </div>
            <button type="button" aria-label="关闭" @click="cancelExpenseEdit">×</button>
          </div>
          <div class="expense-fields">
            <label class="amount-field">
              <span>金额</span>
              <div><b>¥</b><input v-model="expenseForm.amount" inputmode="decimal" placeholder="0.00" required autofocus></div>
            </label>
            <label>
              <span>时间</span>
              <input v-model="expenseForm.spentAt" type="datetime-local" required>
            </label>
          </div>
          <fieldset class="category-field">
            <legend>分类 <small>必选</small></legend>
            <div>
              <label v-for="category in categories" :key="category.id">
                <input v-model="expenseForm.category" type="radio" :value="category.id">
                <span class="category-icon">{{ category.icon }}</span>
                <span>{{ category.label }}</span>
              </label>
            </div>
          </fieldset>
          <label class="note-field">
            <span>备注 <small>选填</small></span>
            <input v-model="expenseForm.note" maxlength="200" placeholder="例如：机场到酒店的出租车">
          </label>
          <div class="editor-footer">
            <p v-if="expenseFormError" role="alert">{{ expenseFormError }}</p>
            <span v-else />
            <div class="editor-actions">
              <button type="button" @click="cancelExpenseEdit">取消</button>
              <button class="primary-button" type="submit" :disabled="expenseSaving">
                {{ expenseSaving ? '保存中…' : editingExpenseID ? '保存修改' : '记下这笔' }}
              </button>
            </div>
          </div>
        </form>
      </div>

      <div v-if="showBookForm" class="modal-backdrop" @click.self="showBookForm = false">
        <form class="book-modal" @submit.prevent="saveBook">
          <div class="modal-heading">
            <div><span>账本</span><h2>{{ editingBook ? '编辑账本' : '新建账本' }}</h2></div>
            <button type="button" aria-label="关闭" @click="showBookForm = false">×</button>
          </div>
          <label>
            <span>名称</span>
            <input v-model="bookForm.name" maxlength="100" placeholder="例如：云南旅行" required autofocus>
          </label>
          <label>
            <span>描述 <small>选填</small></span>
            <textarea v-model="bookForm.description" maxlength="500" placeholder="记录这本账的用途" />
          </label>
          <div class="modal-actions">
            <button type="button" @click="showBookForm = false">取消</button>
            <button class="primary-button" type="submit" :disabled="bookSaving">
              {{ bookSaving ? '保存中…' : '保存账本' }}
            </button>
          </div>
        </form>
      </div>
    </main>
  </div>
</template>

<style scoped>
.accounting-page { max-width: none; padding-right: clamp(0rem, 2vw, 2rem); }
.accounting-heading { display: flex; justify-content: space-between; gap: 2rem; align-items: end; margin-bottom: 2rem; }
.accounting-heading .desc { margin-bottom: 0; }
.accounting-eyebrow, .panel-title > span, .panel-title > div > span, .ledger-heading > div > span,
.modal-heading span { color: var(--blaze); font-family: var(--mono); font-size: .68rem; letter-spacing: .13em; text-transform: uppercase; }
.heading-actions { display: flex; align-items: center; gap: .7rem; }
.statistic-link { border-bottom: 1px solid var(--pine); color: var(--pine); text-decoration: none; font-size: .78rem; }
.statistic-link:hover { color: var(--blaze); border-color: var(--blaze); }
button, input, textarea { font: inherit; }
.primary-button { border: 0; border-radius: 3px; padding: .7rem 1.1rem; background: var(--ink); color: var(--paper); cursor: pointer; white-space: nowrap; }
.primary-button:hover { background: var(--pine); }
.primary-button:disabled { cursor: wait; opacity: .6; }
.accounting-alert { padding: .75rem 1rem; border-left: 3px solid var(--blaze); background: color-mix(in srgb, var(--blaze) 8%, transparent); color: #8b3e23; font-size: .88rem; }
.accounting-workspace { display: grid; grid-template-columns: minmax(14rem, 28%) 1fr; min-height: 36rem; border: 1px solid var(--binding); border-radius: 5px; background: color-mix(in srgb, var(--paper) 90%, white); }
.book-panel { border-right: 1px solid var(--binding); }
.panel-title { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 1rem 1.1rem; border-bottom: 1px solid var(--rule); }
.panel-title > div { display: flex; align-items: baseline; gap: .7rem; }
.panel-title small { color: var(--faded); font-size: .72rem; }
.book-list { display: grid; padding: .45rem; gap: .2rem; }
.book-item { display: flex; width: 100%; justify-content: space-between; gap: .8rem; align-items: center; border: 0; border-radius: 3px; padding: .8rem; background: transparent; color: var(--ink); text-align: left; cursor: pointer; }
.book-item:hover { background: var(--paper-2); }
.book-item.active { background: var(--pine); color: var(--paper); }
.book-item > span { min-width: 0; display: grid; }
.book-item strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--serif); font-size: 1rem; font-weight: 500; }
.book-item small { opacity: .68; font-size: .68rem; }
.book-item b { font-family: var(--mono); font-size: .7rem; font-weight: 400; white-space: nowrap; }
.book-empty, .ledger-empty { padding: 3rem 1.2rem; text-align: center; color: var(--faded); }
.book-empty > span, .ledger-empty > span { font-family: var(--hand); font-size: 1.4rem; color: var(--blaze); }
.book-empty p { font-size: .82rem; }
.book-empty button { border: 0; border-bottom: 1px solid var(--pine); padding: .2rem; background: transparent; color: var(--pine); cursor: pointer; }
.ledger-panel { min-width: 0; padding: 1.5rem; }
.ledger-heading { display: flex; justify-content: space-between; gap: 1rem; }
.ledger-heading h2 { margin: .1rem 0 0; font-family: var(--serif); font-size: 1.65rem; font-weight: 500; }
.ledger-heading p { margin: .15rem 0 0; color: var(--faded); font-size: .82rem; }
.ledger-actions { display: flex; gap: .4rem; align-items: start; }
.ledger-actions button, .editor-title button, .expense-actions button { border: 0; padding: .25rem .4rem; background: transparent; color: var(--faded); cursor: pointer; font-size: .72rem; }
.ledger-actions button:hover, .editor-title button:hover, .expense-actions button:hover { color: var(--ink); }
.ledger-actions .danger-link:hover { color: var(--blaze); }
.book-overview { display: grid; grid-template-columns: 1.4fr 1fr 1fr; margin: 1.2rem 0; padding: 1rem 0; border-block: 1px solid var(--rule); }
.book-overview > div { display: grid; gap: .1rem; padding: 0 1rem; border-right: 1px solid var(--rule); }
.book-overview > div:first-child { padding-left: 0; }
.book-overview > div:last-child { border-right: 0; }
.book-overview small { color: var(--faded); font-size: .68rem; }
.book-overview strong { font-family: var(--serif); font-size: 1.25rem; font-weight: 500; }
.expense-editor { padding: 1.5rem; border-radius: 5px; background: var(--paper); }
.expense-fields { display: grid; grid-template-columns: 1fr 1fr; gap: .8rem; }
.expense-editor label > span, .expense-editor legend, .book-modal label > span { display: block; margin-bottom: .3rem; color: var(--faded); font-size: .72rem; }
.expense-editor input, .book-modal input, .book-modal textarea { width: 100%; border: 1px solid var(--binding); border-radius: 3px; padding: .65rem .75rem; background: var(--paper); color: var(--ink); }
.amount-field > div { position: relative; }
.amount-field b { position: absolute; left: .75rem; top: .54rem; color: var(--faded); font-weight: 400; }
.amount-field input { padding-left: 1.7rem; font-family: var(--mono); }
.category-field { margin: .8rem 0; padding: 0; border: 0; }
.category-field legend small, .note-field small, .book-modal small { opacity: .75; }
.category-field > div { display: grid; grid-template-columns: repeat(8, 1fr); gap: .35rem; }
.category-field label { position: relative; display: grid; justify-items: center; gap: .15rem; border: 1px solid transparent; border-radius: 3px; padding: .4rem .15rem; color: var(--faded); cursor: pointer; font-size: .68rem; }
.category-field label:has(input:checked) { border-color: var(--pine); background: color-mix(in srgb, var(--pine) 9%, transparent); color: var(--pine); }
.category-field input { position: absolute; opacity: 0; }
.category-icon, .expense-category { display: grid; place-items: center; width: 1.7rem; height: 1.7rem; border-radius: 50%; background: var(--paper); font-family: var(--serif); font-size: .8rem; }
.editor-footer { display: flex; justify-content: space-between; align-items: center; gap: 1rem; margin-top: .8rem; }
.editor-footer p { margin: 0; color: var(--blaze); font-size: .75rem; }
.editor-actions { display: flex; align-items: center; gap: .6rem; }
.editor-actions > button:first-child { border: 0; background: transparent; color: var(--faded); cursor: pointer; }
.expense-section { margin-top: 1.3rem; }
.expense-section .panel-title { padding-inline: 0; }
.add-expense-button { border: 0; border-radius: 3px; padding: .5rem .8rem; background: var(--ink); color: var(--paper); cursor: pointer; font-size: .75rem; }
.add-expense-button:hover { background: var(--pine); }
.expense-item { display: grid; grid-template-columns: auto minmax(0, 1fr) auto auto; gap: .8rem; align-items: center; padding: .75rem 0; border-bottom: 1px solid var(--rule); }
.expense-category { background: var(--paper-2); color: var(--pine); }
.expense-copy { display: grid; min-width: 0; }
.expense-copy strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: .87rem; font-weight: 500; }
.expense-copy small { color: var(--faded); font-size: .68rem; }
.expense-item > b { font-family: var(--mono); font-size: .8rem; font-weight: 500; }
.expense-actions { display: flex; }
.expense-actions button:last-child:hover { color: var(--blaze); }
.compact-empty { margin: 0; padding: 1.5rem 0; color: var(--faded); font-family: var(--hand); font-size: 1.05rem; }
.pagination { display: flex; justify-content: center; align-items: center; gap: .7rem; margin-top: 1rem; }
.pagination button { border: 1px solid var(--binding); border-radius: 3px; padding: .3rem .7rem; background: transparent; color: var(--ink); cursor: pointer; font-size: .72rem; }
.pagination button:disabled { opacity: .35; cursor: default; }
.pagination span { color: var(--faded); font-family: var(--mono); font-size: .72rem; }
.accounting-loading { display: flex; align-items: center; gap: .5rem; padding: 1rem 0; color: var(--faded); font-size: .8rem; }
.accounting-loading.large { justify-content: center; min-height: 20rem; }
.ledger-empty { display: grid; justify-items: center; min-height: 28rem; align-content: center; }
.ledger-empty h2 { max-width: 20ch; font-family: var(--serif); font-size: 1.4rem; font-weight: 500; }
.modal-backdrop { position: fixed; z-index: 20; inset: 0; display: grid; place-items: center; padding: 1rem; background: rgb(42 38 32 / .45); }
.book-modal { width: min(30rem, 100%); padding: 1.5rem; border: 1px solid var(--binding); border-radius: 5px; background: var(--paper); box-shadow: 0 20px 70px rgb(42 38 32 / .2); }
.expense-modal { width: min(42rem, 100%); border: 1px solid var(--binding); box-shadow: 0 20px 70px rgb(42 38 32 / .2); }
.modal-heading { display: flex; justify-content: space-between; align-items: start; margin-bottom: 1rem; }
.modal-heading h2 { margin: 0; font-family: var(--serif); font-weight: 500; }
.modal-heading button { border: 0; background: transparent; color: var(--faded); cursor: pointer; font-size: 1.5rem; }
.book-modal label { display: block; margin-bottom: .8rem; }
.book-modal textarea { min-height: 6rem; resize: vertical; }
.modal-actions { display: flex; justify-content: flex-end; gap: .6rem; margin-top: 1rem; }
.modal-actions > button:first-child { border: 0; background: transparent; color: var(--faded); cursor: pointer; }

@media (max-width: 850px) {
  .accounting-workspace { grid-template-columns: 1fr; }
  .book-panel { border-right: 0; border-bottom: 1px solid var(--binding); }
  .book-list { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .category-field > div { grid-template-columns: repeat(4, 1fr); }
}

@media (max-width: 620px) {
  .accounting-page { padding-left: 0; padding-right: 0; }
  .accounting-heading { align-items: start; }
  .accounting-heading .primary-button { padding-inline: .8rem; }
  .ledger-panel { padding: 1rem; }
  .book-list { grid-template-columns: 1fr; }
  .expense-fields { grid-template-columns: 1fr; }
  .book-overview > div { padding-inline: .6rem; }
  .book-overview strong { font-size: 1rem; }
  .expense-item { grid-template-columns: auto minmax(0, 1fr) auto; }
  .expense-actions { grid-column: 2 / -1; justify-content: flex-end; margin-top: -.5rem; }
}
</style>
