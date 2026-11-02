<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameMail.title') }}</span>
          <span class="hint">{{ $t('admin.gameMail.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameMail.playerId')">
          <el-input
            v-model="playerId"
            :placeholder="$t('admin.gameMail.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadList">
            {{ $t('admin.gameMail.list') }}
          </el-button>
          <el-button type="success" @click="openSend">
            {{ $t('admin.gameMail.send') }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="rows" row-key="id" border>
        <el-table-column :label="$t('admin.gameMail.titleLabel')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ row.title }}</template>
        </el-table-column>
        <el-table-column :label="$t('admin.gameMail.contentLabel')" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">{{ row.content || '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('admin.gameMail.attachments')" min-width="200">
          <template #default="{ row }">
            <span v-if="!row.attachments?.length">{{ $t('admin.gameMail.noAttachments') }}</span>
            <span v-else>
              <el-tag
                v-for="(a, i) in row.attachments"
                :key="i"
                class="att-tag"
                type="warning"
              >{{ a.type }}-{{ a.cfgId }} x{{ a.count }}</el-tag>
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('admin.gameMail.status')" min-width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('admin.gameMail.createdAt')" min-width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="sendDialog.visible" :title="$t('admin.gameMail.sendTitle')" width="560px">
      <el-form label-width="100px">
        <el-form-item :label="$t('admin.gameMail.playerId')">
          <el-input-number v-model="sendDialog.playerId" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameMail.titleLabel')" required>
          <el-input v-model="sendDialog.title" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameMail.contentLabel')">
          <el-input v-model="sendDialog.content" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameMail.attachments')">
          <div class="attachments">
            <div v-for="(a, i) in sendDialog.attachments" :key="i" class="attach-row">
              <el-input-number v-model="a.type" :min="0" :controls="false" :placeholder="$t('admin.gameMail.attachmentType')" style="width: 110px" />
              <el-input-number v-model="a.cfgId" :min="0" :controls="false" :placeholder="$t('admin.gameMail.attachmentCfgId')" style="width: 130px" />
              <el-input-number v-model="a.count" :min="1" :controls="false" :placeholder="$t('admin.gameMail.attachmentCount')" style="width: 100px" />
              <el-button link type="danger" @click="removeAttachment(i)">{{ $t('admin.gameMail.removeAttachment') }}</el-button>
            </div>
            <el-button link type="primary" @click="addAttachment">+ {{ $t('admin.gameMail.addAttachment') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sendDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="sendDialog.loading" @click="submitSend">
          {{ $t('common.dialogConfirm') }}
        </el-button>
      </template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import PageContainer from '@/components/PageContainer/index.vue'
import { msgInfo, msgSuccess } from '@/utils/message'

import { listGmMail, sendGmMail } from '@/api/admin/gm'
import type { GmMail, GmMailAttachment } from '@/types/admin'

defineOptions({ name: 'GameMailManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmMail[]>([])

const sendDialog = reactive({
  visible: false,
  loading: false,
  playerId: 0,
  title: '',
  content: '',
  attachments: [] as GmMailAttachment[]
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const loadList = async() => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameMail.playerIdInvalid'))
    return
  }
  loading.value = true
  const res = await listGmMail({ playerId: id }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data ?? []
}

const openSend = () => {
  sendDialog.playerId = playerIdOrZero()
  sendDialog.title = ''
  sendDialog.content = ''
  sendDialog.attachments = []
  sendDialog.visible = true
}

const addAttachment = () => {
  sendDialog.attachments.push({ type: 0, cfgId: 0, count: 1 })
}

const removeAttachment = (i: number) => {
  sendDialog.attachments.splice(i, 1)
}

const submitSend = async() => {
  if (!sendDialog.playerId || !sendDialog.title) {
    msgInfo(t('admin.gameMail.playerIdInvalid'))
    return
  }
  sendDialog.loading = true
  const res = await sendGmMail({
    playerId: sendDialog.playerId,
    title: sendDialog.title,
    content: sendDialog.content,
    attachments: sendDialog.attachments.filter(a => (a.cfgId ?? 0) > 0)
  }).catch(() => null)
  sendDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameMail.sendOk'))
  sendDialog.visible = false
  if (playerIdOrZero() === sendDialog.playerId) {
    await loadList()
  }
}

const statusText = (s?: number) => {
  switch (s) {
    case 1: return t('admin.gameMail.statusUnread')
    case 2: return t('admin.gameMail.statusRead')
    case 3: return t('admin.gameMail.statusClaimed')
    default: return '-'
  }
}

const statusType = (s?: number) => {
  switch (s) {
    case 1: return 'danger'
    case 2: return 'info'
    case 3: return 'success'
    default: return ''
  }
}

const formatTime = (ms?: number) => {
  if (!ms) return '-'
  const d = new Date(ms)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
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

.att-tag {
  margin-right: 4px;
}

.attachments {
  width: 100%;
}

.attach-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
</style>
