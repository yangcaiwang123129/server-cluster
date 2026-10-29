<template>
  <PageContainer>
    <ProTable
      :table="table"
      selection
      row-key="versionId"
      :default-sort="{ prop: 'createdAt', order: 'descending' }"
    >
      <template #search>
        <el-form-item :label="$t('admin.clientVersion.version')">
          <el-input
            v-model="table.query.version"
            :placeholder="$t('admin.clientVersion.versionPlaceholder')"
            clearable
          />
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.platform')">
          <el-select
            v-model="table.query.platform"
            :placeholder="$t('admin.clientVersion.platformPlaceholder')"
            clearable
          >
            <el-option
              v-for="item in platformOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.status')">
          <el-select
            v-model="table.query.status"
            :placeholder="$t('common.selectPlaceholder')"
            clearable
          >
            <el-option
              v-for="item in sys_normal_disable"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
      </template>

      <template #toolbar>
        <el-button v-permisaction="['client:version:list']" type="primary" @click="form.openCreate()">
          {{ $t('common.add') }}
        </el-button>
        <el-button
          v-permisaction="['client:version:list']"
          type="danger"
          plain
          :disabled="table.multiple"
          @click="remove(table.selectedIds)"
        >{{ $t('common.delete') }}</el-button>
      </template>

      <el-table-column
        :label="$t('admin.clientVersion.version')"
        prop="version"
        min-width="120"
        sortable="custom"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('admin.clientVersion.platform')" prop="platform" min-width="100">
        <template #default="{ row }">{{ platformLabel(row.platform) }}</template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.clientVersion.downloadUrl')"
        prop="downloadUrl"
        min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <el-link v-if="row.downloadUrl" :href="row.downloadUrl" target="_blank" type="primary">
            {{ row.downloadUrl }}
          </el-link>
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.clientVersion.forceUpdate')" prop="forceUpdate" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.forceUpdate" type="warning" disable-transitions>
            {{ $t('admin.clientVersion.yes') }}
          </el-tag>
          <el-tag v-else disable-transitions>{{ $t('admin.clientVersion.no') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('admin.clientVersion.status')" prop="status" width="90">
        <template #default="{ row }">
          <el-switch
            :model-value="row.status === '2'"
            :loading="statusLoading === row.versionId"
            @change="(value: boolean | string | number) => toggleStatus(row, value)"
          />
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('admin.clientVersion.releaseNote')"
        prop="releaseNote"
        min-width="180"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('common.createdAt')" prop="createdAt" min-width="110" sortable="custom">
        <template #default="{ row }"><DateCell :value="row.createdAt" /></template>
      </el-table-column>

      <template #actions="{ row }">
        <el-button v-permisaction="['client:version:list']" link type="primary" @click="form.openEdit(row)">
          {{ $t('common.edit') }}
        </el-button>
        <el-button v-permisaction="['client:version:list']" link type="danger" @click="remove(row.versionId)">
          {{ $t('common.delete') }}
        </el-button>
      </template>
    </ProTable>

    <el-dialog
      v-model="form.visible"
      :title="form.title"
      width="520px"
      :close-on-click-modal="false"
      @closed="form.reset"
    >
      <el-form
        :ref="form.bindFormRef"
        v-loading="form.loading"
        :model="form.model"
        :rules="form.rules"
        label-width="auto"
      >
        <el-form-item :label="$t('admin.clientVersion.version')" prop="version">
          <el-input
            v-model="form.model.version"
            :placeholder="$t('admin.clientVersion.versionPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.platform')" prop="platform">
          <el-select
            v-model="form.model.platform"
            :placeholder="$t('admin.clientVersion.platformPlaceholder')"
            style="width: 100%"
          >
            <el-option
              v-for="item in platformOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.downloadUrl')" prop="downloadUrl">
          <el-input
            v-model="form.model.downloadUrl"
            :placeholder="$t('admin.clientVersion.downloadUrlPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.forceUpdate')" prop="forceUpdate">
          <el-switch v-model="form.model.forceUpdate" />
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.status')" prop="status">
          <el-radio-group v-model="form.model.status">
            <el-radio
              v-for="item in sys_normal_disable"
              :key="item.value"
              :value="item.value"
            >{{ item.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('admin.clientVersion.releaseNote')" prop="releaseNote">
          <el-input
            v-model="form.model.releaseNote"
            type="textarea"
            :rows="3"
            :placeholder="$t('admin.clientVersion.releaseNotePlaceholder')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="form.close">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="form.submitting" @click="form.submit">
          {{ $t('common.dialogConfirm') }}
        </el-button>
      </template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FormRules } from 'element-plus'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable, useForm, useRemove, useDict } from '@/composables'
import { msgSuccess } from '@/utils/message'

import {
  listClientVersion,
  getClientVersion,
  addClientVersion,
  updateClientVersion,
  delClientVersion,
  updateClientVersionStatus
} from '@/api/admin/client-version'
import type { ClientVersion, ClientVersionQuery } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'ClientVersionManage' })

const { t } = useI18n()

const { sys_normal_disable } = useDict('sys_normal_disable')

const platformOptions = computed(() => [
  { label: t('admin.clientVersion.platformAndroid'), value: 'android' },
  { label: t('admin.clientVersion.platformIOS'), value: 'ios' },
  { label: t('admin.clientVersion.platformWindows'), value: 'windows' }
])

const platformLabel = (platform?: string) =>
  platformOptions.value.find(item => item.value === platform)?.label || platform || '-'

const table = useTable<ClientVersion, ClientVersionQuery>({
  api: listClientVersion,
  idKey: 'versionId',
  defaultSort: { prop: 'createdAt', order: 'descending' },
  defaultQuery: () => ({ version: undefined, platform: undefined, status: undefined })
})

const rules = computed<FormRules>(() => ({
  version: [{ required: true, message: t('admin.clientVersion.rules.version'), trigger: 'blur' }],
  platform: [{ required: true, message: t('admin.clientVersion.rules.platform'), trigger: 'change' }],
  downloadUrl: [{ required: true, message: t('admin.clientVersion.rules.downloadUrl'), trigger: 'blur' }]
}))

const form = useForm<ClientVersion, number>({
  defaultModel: () => ({
    versionId: undefined,
    version: undefined,
    platform: 'android',
    downloadUrl: undefined,
    forceUpdate: false,
    status: '2',
    releaseNote: undefined
  }),
  idKey: 'versionId',
  rules,
  title: {
    create: computed(() => t('admin.clientVersion.addTitle')),
    edit: computed(() => t('admin.clientVersion.editTitle'))
  },
  api: { get: getClientVersion, add: addClientVersion, update: updateClientVersion },
  onSuccess: () => table.getList()
})

const { remove } = useRemove({
  api: delClientVersion,
  onSuccess: () => table.getList()
})

const statusLoading = ref<number | null>(null)

const toggleStatus = async(row: ClientVersion, value: boolean | string | number) => {
  if (!row.versionId) return
  const status = value ? '2' : '1'
  statusLoading.value = row.versionId
  const res = await updateClientVersionStatus({ versionId: row.versionId, status }).catch(() => null)
  statusLoading.value = null
  if (!res) return
  row.status = status
  msgSuccess(t('admin.clientVersion.statusOk'))
}
</script>
