<template>
  <PageContainer>
    <ProTable
      :table="table"
      row-key="id"
      :default-sort="{ prop: 'id', order: 'descending' }"
    >
      <template #search>
        <el-form-item :label="$t('admin.activity.cfgId')">
          <el-input-number
            v-model="table.query.cfgId"
            :placeholder="$t('admin.activity.cfgIdPlaceholder')"
            :controls="false"
            :min="1"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.activity.status')">
          <el-select
            v-model="table.query.status"
            :placeholder="$t('admin.activity.statusPlaceholder')"
            clearable
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
      </template>

      <el-table-column prop="id" label="ID" width="80" sortable="custom" />
      <el-table-column
        :label="$t('admin.activity.cfgId')"
        prop="cfgId"
        width="110"
        sortable="custom"
      />
      <el-table-column
        :label="$t('admin.activity.name')"
        prop="cfgName"
        min-width="140"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <el-button
            v-if="cfgOf(row)"
            link
            type="primary"
            class="name-link"
            @click="openStrategy(row)"
          >
            <el-icon class="name-icon"><Calendar /></el-icon>
            {{ row.cfgName || '-' }}
          </el-button>
          <span v-else>{{ row.cfgName || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.round')" prop="round" width="80" />
      <el-table-column
        :label="$t('admin.activity.startTime')"
        prop="startTime"
        min-width="160"
        sortable="custom"
      >
        <template #default="{ row }"><DateCell :value="row.startTime" /></template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.activity.endTime')"
        prop="endTime"
        min-width="160"
        sortable="custom"
      >
        <template #default="{ row }"><DateCell :value="row.endTime" /></template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.status')" prop="status" width="110">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status)" effect="light" round disable-transitions>
            <span class="status-dot" />
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.source')" prop="source" width="100">
        <template #default="{ row }">
          <el-tag
            v-if="row.source === 1"
            type="info"
            effect="plain"
            disable-transitions
          >
            {{ $t('admin.activity.sourceConfig') }}
          </el-tag>
          <span v-else>{{ row.source ?? '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.activity.remark')"
        prop="remark"
        min-width="140"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('common.createdAt')" prop="createdAt" min-width="160">
        <template #default="{ row }"><DateCell :value="row.createdAt" /></template>
      </el-table-column>

      <template #actions="{ row }">
        <el-button
          v-if="row.status === 0"
          v-permisaction="['game:activity:list']"
          link
          type="primary"
          @click="openTime(row)"
        >
          {{ $t('admin.activity.editTime') }}
        </el-button>
        <el-button
          v-if="row.status === 1"
          v-permisaction="['game:activity:list']"
          link
          type="danger"
          @click="endNow(row)"
        >
          {{ $t('admin.activity.endNow') }}
        </el-button>
      </template>
    </ProTable>

    <el-dialog
      v-model="timeDialog.visible"
      :title="$t('admin.activity.timeTitle')"
      width="480px"
      :close-on-click-modal="false"
    >
      <el-form :model="timeDialog" label-width="auto">
        <el-form-item :label="$t('admin.activity.startTime')">
          <el-date-picker
            v-model="timeDialog.startTime"
            type="datetime"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.activity.endTime')">
          <el-date-picker
            v-model="timeDialog.endTime"
            type="datetime"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="timeDialog.visible = false">
          {{ $t('common.dialogCancel') }}
        </el-button>
        <el-button type="primary" :loading="timeDialog.submitting" @click="submitTime">
          {{ $t('common.dialogConfirm') }}
        </el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="strategyDrawer.visible"
      :title="$t('admin.activity.strategyTitle')"
      size="480px"
    >
      <el-descriptions v-if="strategyDrawer.cfg" :column="1" border>
        <el-descriptions-item :label="$t('admin.activity.name')">
          <span class="strategy-name">{{ strategyDrawer.cfg.name || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.desc')">
          {{ strategyDrawer.cfg.desc || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.strategyType')">
          <el-tag type="primary" effect="light" disable-transitions>
            {{ strategyLabel(strategyDrawer.cfg.strategyType) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.strategyStartValue')">
          <span class="strategy-time">
            {{ formatStrategyValue(strategyDrawer.cfg.strategyType, strategyDrawer.cfg.strategyStartValue) }}
          </span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.strategyEndValue')">
          <span class="strategy-time">
            {{ formatStrategyValue(strategyDrawer.cfg.strategyType, strategyDrawer.cfg.strategyEndValue) }}
          </span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.roundCount')">
          <el-tag
            v-if="!strategyDrawer.cfg.roundCount"
            type="info"
            effect="plain"
            disable-transitions
          >
            {{ $t('admin.activity.unlimited') }}
          </el-tag>
          <span v-else>{{ strategyDrawer.cfg.roundCount }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.triggerConditions')">
          <template v-if="strategyDrawer.cfg.triggerConditions?.length">
            <el-tag
              v-for="(c, i) in strategyDrawer.cfg.triggerConditions"
              :key="i"
              type="info"
              effect="plain"
              class="drawer-tag"
              disable-transitions
            >
              {{ c.type ?? '-' }} : {{ c.args ?? '-' }}
            </el-tag>
          </template>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.rewards')">
          <template v-if="strategyDrawer.cfg.rewards?.length">
            <el-tag
              v-for="(r, i) in strategyDrawer.cfg.rewards"
              :key="i"
              type="success"
              effect="plain"
              class="drawer-tag"
              disable-transitions
            >
              {{ r.id ?? '-' }} × {{ r.count ?? '-' }}
            </el-tag>
          </template>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.activity.taskCount')">
          {{ strategyDrawer.cfg.taskCount ?? '-' }}
        </el-descriptions-item>
      </el-descriptions>
      <el-empty v-else :description="$t('common.empty')" />
    </el-drawer>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable } from '@/composables'
import { msgSuccess } from '@/utils/message'
import { listActivities, listActivityConfigs, updateActivityTime, endActivity } from '@/api/admin/activity'
import type { GmActivity, GmActivityAsset, GmActivityCfg, GmActivityQuery } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'ActivityManage' })

const { t } = useI18n()

const statusOptions = computed(() => [
  { label: t('admin.activity.statusNotStarted'), value: 0 },
  { label: t('admin.activity.statusRunning'), value: 1 },
  { label: t('admin.activity.statusEnded'), value: 2 },
  { label: t('admin.activity.statusCancelled'), value: 3 }
])

const statusLabel = (status?: number) => {
  switch (status) {
    case 0:
      return t('admin.activity.statusNotStarted')
    case 1:
      return t('admin.activity.statusRunning')
    case 2:
      return t('admin.activity.statusEnded')
    case 3:
      return t('admin.activity.statusCancelled')
    default:
      return '-'
  }
}

const statusTag = (status?: number): 'success' | 'info' | 'warning' | 'danger' => {
  switch (status) {
    case 1:
      return 'success'
    case 2:
      return 'info'
    case 3:
      return 'danger'
    default:
      return 'warning'
  }
}

const table = useTable<GmActivity, GmActivityQuery>({
  api: listActivities,
  idKey: 'id',
  defaultSort: { prop: 'id', order: 'descending' },
  defaultQuery: () => ({ cfgId: undefined, status: undefined })
})

// 活动模板（开服策略等）按 cfgId 索引，供列表点击查看
const cfgMap = reactive(new Map<number, GmActivityCfg>())

const loadConfigs = async() => {
  const res = await listActivityConfigs().catch(() => null)
  if (!res) return
  cfgMap.clear()
  for (const cfg of res.data || []) {
    if (cfg.id !== undefined) cfgMap.set(cfg.id, cfg)
  }
}

const cfgOf = (row: GmActivity) =>
  row.cfgId !== undefined ? cfgMap.get(row.cfgId) : undefined

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

const weekdayKey = (day: number) => {
  switch (day) {
    case 1:
      return 'admin.activity.weekdays.1'
    case 2:
      return 'admin.activity.weekdays.2'
    case 3:
      return 'admin.activity.weekdays.3'
    case 4:
      return 'admin.activity.weekdays.4'
    case 5:
      return 'admin.activity.weekdays.5'
    case 6:
      return 'admin.activity.weekdays.6'
    default:
      return 'admin.activity.weekdays.7'
  }
}

// 策略值 "前缀 时:分:秒" 转人类可读文案；解析失败原样返回。
const formatStrategyValue = (type?: number, value?: string) => {
  if (!value) return '-'
  const parts = value.split(/[\s_]+/)
  if (parts.length !== 2) return value
  const [prefix, time] = parts
  switch (type) {
    case 2:
      return t('admin.activity.strategyDaily', { time })
    case 3: {
      const day = Number(prefix)
      if (!(day >= 1 && day <= 7)) return value
      return t('admin.activity.strategyWeekly', { day: t(weekdayKey(day)), time })
    }
    case 4: {
      const day = Number(prefix)
      if (day <= 0) return t('admin.activity.strategyMonthlyLast', { time })
      return t('admin.activity.strategyMonthly', { day: prefix, time })
    }
    default:
      return value
  }
}

const strategyDrawer = reactive<{ visible: boolean; cfg: GmActivityCfg | null }>({
  visible: false,
  cfg: null
})

const openStrategy = (row: GmActivity) => {
  strategyDrawer.cfg = cfgOf(row) || null
  strategyDrawer.visible = true
}

const timeDialog = reactive<{
  visible: boolean
  submitting: boolean
  id: number
  startTime: Date | null
  endTime: Date | null
}>({
  visible: false,
  submitting: false,
  id: 0,
  startTime: null,
  endTime: null
})

const openTime = (row: GmActivity) => {
  timeDialog.id = row.id || 0
  timeDialog.startTime = row.startTime ? new Date(row.startTime) : null
  timeDialog.endTime = row.endTime ? new Date(row.endTime) : null
  timeDialog.visible = true
}

const submitTime = async() => {
  const start = timeDialog.startTime?.getTime()
  const end = timeDialog.endTime?.getTime()
  if (!start || !end || start >= end) {
    ElMessage.warning(t('admin.activity.invalidTime'))
    return
  }
  timeDialog.submitting = true
  try {
    const res = await updateActivityTime(timeDialog.id, { startTime: start, endTime: end }).catch(() => null)
    if (!res) return
    msgSuccess(t('admin.activity.timeOk'))
    timeDialog.visible = false
    await table.getList()
  } finally {
    timeDialog.submitting = false
  }
}

const endNow = async(row: GmActivity) => {
  try {
    await ElMessageBox.confirm(t('admin.activity.endConfirm'), t('common.notice'), { type: 'warning' })
  } catch {
    return
  }
  if (!row.id) return
  const res = await endActivity(row.id).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.activity.endOk'))
  await table.getList()
}

onMounted(() => {
  loadConfigs()
})
</script>

<style scoped>
.name-link :deep(.name-icon) {
  margin-right: 4px;
}

.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-right: 4px;
  vertical-align: middle;
  border-radius: 50%;
  background: currentColor;
}

.strategy-name {
  font-weight: 600;
}

.strategy-time {
  font-variant-numeric: tabular-nums;
}

.drawer-tag {
  margin: 0 6px 6px 0;
}
</style>
