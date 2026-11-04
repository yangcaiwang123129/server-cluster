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
            style="width: 160px"
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
      />
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
      <el-table-column :label="$t('admin.activity.status')" prop="status" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status)" disable-transitions>
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.activity.source')" prop="source" width="80" />
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
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable } from '@/composables'
import { msgSuccess } from '@/utils/message'
import { listActivities, updateActivityTime, endActivity } from '@/api/admin/activity'
import type { GmActivity, GmActivityQuery } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'ActivityInstanceManage' })

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
</script>
