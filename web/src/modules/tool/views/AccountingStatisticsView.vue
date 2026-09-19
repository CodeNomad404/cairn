<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { APIRequestError } from '../../../shared/api/http'
import { getAccountingMonthlyStatistic } from '../api/accounting'
import type {
  AccountingExpenseCategory,
  AccountingMonthlyStatistic,
} from '../types'

const categoryLabels: Record<AccountingExpenseCategory, string> = {
  1: '餐饮',
  2: '交通',
  3: '住宿',
  4: '购物',
  5: '票务',
  6: '娱乐',
  7: '医疗',
  8: '其他',
}

const selectedMonth = ref(currentMonth())
const statistic = ref<AccountingMonthlyStatistic | null>(null)
const mode = ref<'category' | 'book'>('category')
const loading = ref(true)
const error = ref('')

const rows = computed(() => {
  if (!statistic.value) return []
  if (mode.value === 'category') {
    return statistic.value.by_category.map((item) => ({
      key: `category-${item.category}`,
      label: categoryLabels[item.category],
      amountCents: item.amount_cents,
      count: item.expense_count,
    }))
  }
  return statistic.value.by_book.map((item) => ({
    key: `book-${item.book_id}`,
    label: item.book_name,
    amountCents: item.amount_cents,
    count: item.expense_count,
  }))
})

const maxAmount = computed(() => Math.max(1, ...rows.value.map((item) => item.amountCents)))
const totalCount = computed(() => {
  if (!statistic.value) return 0
  return statistic.value.by_category.reduce((total, item) => total + item.expense_count, 0)
})

function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function formatMoney(cents: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
  }).format(cents / 100)
}

function formatMonth(month: string): string {
  const [year, value] = month.split('-')
  return `${year} 年 ${Number(value)} 月`
}

async function loadStatistic() {
  loading.value = true
  error.value = ''
  try {
    statistic.value = await getAccountingMonthlyStatistic(selectedMonth.value)
  } catch (reason) {
    if (reason instanceof APIRequestError && reason.status === 401) {
      error.value = '登录状态已失效，请重新登录。'
    } else {
      error.value = reason instanceof Error ? reason.message : '月度统计加载失败，请稍后重试。'
    }
  } finally {
    loading.value = false
  }
}

function moveMonth(offset: number) {
  const [year, month] = selectedMonth.value.split('-').map(Number)
  const next = new Date(year, month - 1 + offset, 1)
  selectedMonth.value = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}`
  void loadStatistic()
}

onMounted(() => void loadStatistic())
</script>

<template>
  <div class="page">
    <main class="tool-page accounting-statistics-page">
      <RouterLink class="back" to="/tools/accounting">← 返回我的账本</RouterLink>

      <header class="statistics-heading">
        <div>
          <span>月度回顾</span>
          <h1>支出统计</h1>
          <p class="desc">查看全部账本在一个自然月内的人民币支出。</p>
        </div>
        <div class="month-picker">
          <button type="button" aria-label="上一个月" @click="moveMonth(-1)">←</button>
          <input v-model="selectedMonth" type="month" aria-label="统计月份" @change="loadStatistic">
          <button type="button" aria-label="下一个月" @click="moveMonth(1)">→</button>
        </div>
      </header>

      <p v-if="error" class="statistics-alert" role="alert">{{ error }}</p>

      <section class="total-card" aria-label="月度支出汇总">
        <div>
          <span>{{ formatMonth(selectedMonth) }}</span>
          <strong>{{ formatMoney(statistic?.total_amount_cents ?? 0) }}</strong>
          <small>共 {{ totalCount }} 笔支出</small>
        </div>
        <p>这里统计你名下所有账本当月发生的支出。</p>
      </section>

      <section class="breakdown-card">
        <div class="breakdown-heading">
          <div>
            <span>支出构成</span>
            <h2>{{ mode === 'category' ? '按分类统计' : '按账本统计' }}</h2>
          </div>
          <div class="statistics-tabs" role="tablist" aria-label="统计方式">
            <button
              type="button"
              role="tab"
              :aria-selected="mode === 'category'"
              :class="{ active: mode === 'category' }"
              @click="mode = 'category'"
            >按分类</button>
            <button
              type="button"
              role="tab"
              :aria-selected="mode === 'book'"
              :class="{ active: mode === 'book' }"
              @click="mode = 'book'"
            >按账本</button>
          </div>
        </div>

        <div v-if="loading" class="statistics-loading"><span class="spinner" /> 正在统计</div>
        <div v-else-if="rows.length" class="breakdown-list">
          <article v-for="(row, index) in rows" :key="row.key" class="breakdown-row">
            <span class="row-number">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="row-copy">
              <div><strong>{{ row.label }}</strong><small>{{ row.count }} 笔</small></div>
              <div class="amount-track" aria-hidden="true">
                <span :style="{ width: `${Math.max(3, row.amountCents / maxAmount * 100)}%` }" />
              </div>
            </div>
            <b>{{ formatMoney(row.amountCents) }}</b>
          </article>
        </div>
        <div v-else class="statistics-empty">
          <span>本月无账</span>
          <p>这个月还没有记录支出。</p>
          <RouterLink to="/tools/accounting">去记一笔</RouterLink>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.accounting-statistics-page { max-width: calc(var(--measure) + 14rem); padding-right: clamp(0rem, 2vw, 2rem); }
.statistics-heading { display: flex; align-items: end; justify-content: space-between; gap: 2rem; margin-bottom: 2rem; }
.statistics-heading > div:first-child > span, .total-card span, .breakdown-heading span {
  color: var(--blaze); font-family: var(--mono); font-size: .68rem; letter-spacing: .13em; text-transform: uppercase;
}
.statistics-heading .desc { margin-bottom: 0; }
.month-picker { display: flex; align-items: center; border: 1px solid var(--binding); border-radius: 4px; background: color-mix(in srgb, var(--paper) 88%, white); }
.month-picker button { border: 0; padding: .55rem .7rem; background: transparent; color: var(--faded); cursor: pointer; }
.month-picker button:hover { color: var(--blaze); }
.month-picker input { border: 0; border-inline: 1px solid var(--rule); padding: .55rem .7rem; background: transparent; color: var(--ink); font: inherit; font-size: .82rem; }
.statistics-alert { padding: .75rem 1rem; border-left: 3px solid var(--blaze); background: color-mix(in srgb, var(--blaze) 8%, transparent); color: #8b3e23; font-size: .88rem; }
.total-card { display: flex; align-items: end; justify-content: space-between; gap: 2rem; min-height: 12rem; padding: 1.7rem 2rem; border-radius: 5px; background: var(--pine); color: var(--paper); }
.total-card > div { display: grid; }
.total-card span { color: #dcb297; }
.total-card strong { margin: .2rem 0; font-family: var(--serif); font-size: clamp(2.5rem, 8vw, 4.5rem); font-weight: 400; line-height: 1.15; }
.total-card small { color: color-mix(in srgb, var(--paper) 70%, transparent); }
.total-card p { max-width: 22ch; margin: 0; color: color-mix(in srgb, var(--paper) 68%, transparent); font-size: .8rem; text-align: right; }
.breakdown-card { margin-top: 1.3rem; padding: 1.5rem 2rem; border: 1px solid var(--binding); border-radius: 5px; background: color-mix(in srgb, var(--paper) 88%, white); }
.breakdown-heading { display: flex; align-items: end; justify-content: space-between; gap: 1rem; padding-bottom: 1rem; border-bottom: 1px solid var(--rule); }
.breakdown-heading h2 { margin: .1rem 0 0; font-family: var(--serif); font-size: 1.45rem; font-weight: 500; }
.statistics-tabs { display: flex; gap: .3rem; }
.statistics-tabs button { border: 0; border-radius: 2rem; padding: .4rem .8rem; background: transparent; color: var(--faded); cursor: pointer; font: inherit; font-size: .75rem; }
.statistics-tabs button.active { background: var(--ink); color: var(--paper); }
.statistics-loading { display: flex; align-items: center; gap: .6rem; padding: 3rem 0; color: var(--faded); font-size: .82rem; }
.breakdown-list { display: grid; }
.breakdown-row { display: grid; grid-template-columns: 2rem minmax(0, 1fr) auto; gap: .9rem; align-items: center; padding: 1rem 0; border-bottom: 1px solid var(--rule); }
.breakdown-row:last-child { border-bottom: 0; }
.row-number { color: var(--binding); font-family: var(--mono); font-size: .68rem; }
.row-copy { display: grid; gap: .45rem; }
.row-copy > div:first-child { display: flex; justify-content: space-between; gap: .8rem; }
.row-copy strong { font-family: var(--serif); font-size: 1rem; font-weight: 500; }
.row-copy small { color: var(--faded); font-size: .7rem; }
.amount-track { height: 5px; overflow: hidden; border-radius: 4px; background: var(--paper-2); }
.amount-track span { display: block; height: 100%; border-radius: inherit; background: var(--blaze); }
.breakdown-row > b { min-width: 7rem; font-family: var(--mono); font-size: .8rem; font-weight: 500; text-align: right; }
.statistics-empty { display: grid; justify-items: start; padding: 3rem 0; color: var(--faded); }
.statistics-empty > span { color: var(--blaze); font-family: var(--hand); font-size: 1.35rem; }
.statistics-empty p { margin: .2rem 0 .8rem; }
.statistics-empty a { border-bottom: 1px solid var(--pine); color: var(--pine); text-decoration: none; font-size: .82rem; }

@media (max-width: 620px) {
  .accounting-statistics-page { padding-left: 0; padding-right: 0; }
  .statistics-heading { align-items: start; flex-direction: column; gap: 1rem; }
  .total-card { align-items: start; flex-direction: column; min-height: 0; padding: 1.3rem; }
  .total-card p { text-align: left; }
  .breakdown-card { padding: 1.1rem; }
  .breakdown-heading { align-items: start; flex-direction: column; }
  .breakdown-row { grid-template-columns: 1.5rem minmax(0, 1fr); }
  .breakdown-row > b { grid-column: 2; min-width: 0; text-align: left; }
}
</style>
