<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>{{ title }}</span>
        <span class="hint">{{ $t('admin.gameRecord.hint') }}</span>
      </div>
    </template>

    <el-tabs v-model="module" @tab-change="handleSearch">
      <el-tab-pane
        v-for="m in modules"
        :key="m.value"
        :label="m.label"
        :name="m.value"
      />
    </el-tabs>

    <el-form :inline="true" @submit.prevent>
      <el-form-item :label="$t('admin.gameRecord.playerId')">
        <el-input
          v-model="playerId"
          :placeholder="$t('admin.gameRecord.playerIdPlaceholder')"
          clearable
          style="width: 180px"
        />
      </el-form-item>
      <el-form-item :label="$t('admin.gameRecord.cfgId')">
        <el-input
          v-model="cfgId"
          :placeholder="$t('admin.gameRecord.cfgIdPlaceholder')"
          clearable
          style="width: 160px"
        />
      </el-form-item>
      <el-form-item :label="$t('admin.gameRecord.range')">
        <el-date-picker
          v-model="range"
          type="datetimerange"
          value-format="x"
          range-separator="-"
          :start-placeholder="$t('admin.gameRecord.rangeStart')"
          :end-placeholder="$t('admin.gameRecord.rangeEnd')"
          style="width: 380px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="handleSearch">
          {{ $t('admin.gameRecord.search') }}
        </el-button>
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="rows" border>
      <el-table-column prop="playerId" :label="$t('admin.gameRecord.playerId')" min-width="160" />
      <el-table-column :label="$t('admin.gameRecord.target')" min-width="220">
        <template #default="{ row }">
          <div class="item-cell">
            <span class="item-name">{{ row.name || '-' }}</span>
            <span class="item-id">#{{ row.cfgId }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="count" :label="$t('admin.gameRecord.count')" width="100" />
      <el-table-column :label="$t('admin.gameRecord.createdAt')" width="180">
        <template #default="{ row }">
          <DateCell :value="row.createdAt" pattern="{y}-{m}-{d} {h}:{i}:{s}" />
        </template>
      </el-table-column>
    </el-table>

    <Pagination
      v-model:page="page"
      v-model:limit="limit"
      :total="total"
      @pagination="load"
    />
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import DateCell from '@/components/DateCell/index.vue'
import Pagination from '@/components/Pagination/index.vue'

import { listGmRecord } from '@/api/admin/gm'
import type { GmRecord } from '@/types/admin'

const props = defineProps<{
  /** 1 = 获得记录，2 = 消耗记录 */
  action: 1 | 2
  title: string
}>()

defineOptions({ name: 'GameRecordTable' })

const { t } = useI18n()

const modules = computed(() => {
  const list = ['item', 'equip', 'weapon', 'skill', 'hero'] as const
  return list.map((value) => ({ value, label: t(`admin.gameRecord.module.${value}`) }))
})

const module = ref<'item' | 'equip' | 'weapon' | 'skill' | 'hero'>('item')
const playerId = ref('')
const cfgId = ref('')
const range = ref<[string, string] | null>(null)

const rows = ref<GmRecord[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const loading = ref(false)

const toNumber = (value: string) => {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : undefined
}

async function load() {
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmRecord({
    action: props.action,
    module: module.value,
    playerId: toNumber(playerId.value),
    cfgId: toNumber(cfgId.value),
    begin: range.value ? Number(range.value[0]) : undefined,
    end: range.value ? Number(range.value[1]) : undefined,
    pageIndex: page.value,
    pageSize: limit.value
  }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data?.list ?? []
  total.value = res.data?.count ?? 0
}

const handleSearch = () => {
  page.value = 1
  load()
}

load()
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.item-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-name {
  font-weight: 500;
}

.item-id {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
