<template>
  <PageContainer>
    <ProTable :table="table" :paginated="false" row-key="id">
      <el-table-column prop="id" :label="$t('admin.activity.id')" width="90" />
      <el-table-column
        prop="name"
        :label="$t('admin.activity.name')"
        min-width="140"
        show-overflow-tooltip
      />
      <el-table-column
        prop="desc"
        :label="$t('admin.activity.desc')"
        min-width="200"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('admin.activity.strategyType')" prop="strategyType" width="120">
        <template #default="{ row }">{{ strategyLabel(row.strategyType) }}</template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.activity.strategyStartValue')"
        prop="strategyStartValue"
        min-width="140"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ row.strategyStartValue || '-' }}</template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.activity.strategyEndValue')"
        prop="strategyEndValue"
        min-width="140"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ row.strategyEndValue || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.roundCount')" prop="roundCount" width="100">
        <template #default="{ row }">
          {{ row.roundCount ? row.roundCount : $t('admin.activity.unlimited') }}
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.triggerLogicType')" prop="triggerLogicType" width="110" />
      <el-table-column
        :label="$t('admin.activity.triggerConditions')"
        min-width="160"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ formatConditions(row.triggerConditions) }}</template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.rewards')" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ formatAssets(row.rewards) }}</template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.taskCount')" prop="taskCount" width="100" />
    </ProTable>
  </PageContainer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import { useTable } from '@/composables'
import { listActivityConfigs } from '@/api/admin/activity'
import type { GmActivityAsset, GmActivityCfg, GmActivityCondition } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'ActivityConfigManage' })

const { t } = useI18n()

const table = useTable<GmActivityCfg>({
  api: listActivityConfigs,
  idKey: 'id',
  paginated: false
})

// 开服策略类型见 tpl.StrategyType
const strategyLabel = (strategyType?: number) => {
  switch (strategyType) {
    case 1:
      return t('admin.activity.strategies.1')
    case 2:
      return t('admin.activity.strategies.2')
    case 3:
      return t('admin.activity.strategies.3')
    case 4:
      return t('admin.activity.strategies.4')
    default:
      return '-'
  }
}

const formatConditions = (conditions?: GmActivityCondition[]) =>
  conditions?.length
    ? conditions.map(item => `${item.type ?? '-'}:${item.args ?? '-'}`).join('，')
    : '-'

const formatAssets = (assets?: GmActivityAsset[]) =>
  assets?.length
    ? assets.map(item => `${item.id ?? '-'}×${item.count ?? '-'}`).join('，')
    : '-'
</script>
