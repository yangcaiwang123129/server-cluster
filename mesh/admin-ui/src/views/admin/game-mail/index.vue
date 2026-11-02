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
              >{{ typeLabel(a.type) }} #{{ a.cfgId }} x{{ a.count }}</el-tag>
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
        <el-form-item :label="$t('admin.gameMail.playerSelect')" required>
          <el-select
            v-model="sendDialog.playerIds"
            multiple
            filterable
            remote
            reserve-keyword
            :remote-method="searchPlayers"
            :loading="playerLoading"
            :placeholder="$t('admin.gameMail.playerSearchPlaceholder')"
            style="width: 100%"
          >
            <el-option
              v-for="p in playerOptions"
              :key="p.playerId"
              :label="`${p.nickname || p.username} #${p.playerId}`"
              :value="p.playerId"
            />
          </el-select>
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
              <el-select v-model="a.type" style="width: 110px" @change="onTypeChange(a)">
                <el-option
                  v-for="opt in attachmentTypes"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                />
              </el-select>
              <el-select
                v-model="a.cfgId"
                filterable
                remote
                reserve-keyword
                :remote-method="(kw: string) => searchAsset(a, kw)"
                :loading="a.loading"
                :placeholder="$t('admin.gameMail.assetSearchPlaceholder')"
                style="width: 220px"
              >
                <el-option
                  v-for="c in a.options"
                  :key="c.id"
                  :label="`${c.name} #${c.id}`"
                  :value="c.id"
                />
              </el-select>
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
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import PageContainer from '@/components/PageContainer/index.vue'
import { msgInfo, msgSuccess } from '@/utils/message'

import { listGameCatalog, listGmMail, sendGmMail } from '@/api/admin/gm'
import { listPlayer } from '@/api/admin/player'
import type { GameCatalogEntry, GmMail, GmMailAttachment, Player } from '@/types/admin'

defineOptions({ name: 'GameMailManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmMail[]>([])

/** 发送弹窗里的附件行：options/loading 只服务于下拉搜索，不下发给后端。 */
interface AttachmentRow extends GmMailAttachment {
  options: GameCatalogEntry[]
  loading: boolean
}

const playerOptions = ref<Player[]>([])
const playerLoading = ref(false)

const sendDialog = reactive({
  visible: false,
  loading: false,
  playerIds: [] as number[],
  title: '',
  content: '',
  attachments: [] as AttachmentRow[]
})

const newAttachmentRow = (): AttachmentRow => ({
  type: 1,
  cfgId: undefined,
  count: 1,
  options: [],
  loading: false
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

// 玩家下拉远程搜索：纯数字按玩家ID精确匹配，其余按账号模糊匹配。
const searchPlayers = async(keyword: string) => {
  playerLoading.value = true
  const params: { pageIndex: number; pageSize: number; playerId?: number; username?: string } = {
    pageIndex: 1,
    pageSize: 20
  }
  if (/^\d+$/.test(keyword)) params.playerId = Number(keyword)
  else if (keyword) params.username = keyword
  const res = await listPlayer(params).catch(() => null)
  playerLoading.value = false
  playerOptions.value = res?.data?.list ?? []
}

const assetCatalogType = (type: number): 'item' | 'equip' | 'hero' =>
  type === 2 ? 'equip' : type === 3 ? 'hero' : 'item'

// 资产下拉远程搜索：第二列的内容随第一列的资产类型切换。
const searchAsset = async(row: AttachmentRow, keyword: string) => {
  row.loading = true
  const res = await listGameCatalog({
    type: assetCatalogType(row.type ?? 1),
    keyword,
    pageIndex: 1,
    pageSize: 20
  }).catch(() => null)
  row.loading = false
  row.options = res?.data?.list ?? []
}

const onTypeChange = (row: AttachmentRow) => {
  row.cfgId = undefined
  row.options = []
  void searchAsset(row, '')
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
  const id = playerIdOrZero()
  sendDialog.playerIds = id ? [id] : []
  sendDialog.title = ''
  sendDialog.content = ''
  sendDialog.attachments = [newAttachmentRow()]
  sendDialog.visible = true
  void searchPlayers('')
}

const addAttachment = () => {
  sendDialog.attachments.push(newAttachmentRow())
}

// 附件类型下拉：只开放可发放的资产类型（1 物品 / 2 装备 / 3 英雄）。
const attachmentTypes = computed(() => [
  { value: 1, label: t('admin.gameMail.attachmentTypeItem') },
  { value: 2, label: t('admin.gameMail.attachmentTypeEquip') },
  { value: 3, label: t('admin.gameMail.attachmentTypeHero') }
])

// 列表展示用：历史邮件可能带武器/技能附件，展示名称不随发送下拉收窄。
const attachmentTypeLabels = computed(() => [
  ...attachmentTypes.value,
  { value: 4, label: t('admin.gameMail.attachmentTypeWeapon') },
  { value: 5, label: t('admin.gameMail.attachmentTypeSkill') }
])

const typeLabel = (type?: number) =>
  attachmentTypeLabels.value.find(o => o.value === type)?.label ?? `#${type}`

const removeAttachment = (i: number) => {
  sendDialog.attachments.splice(i, 1)
}

const submitSend = async() => {
  if (!sendDialog.playerIds.length) {
    msgInfo(t('admin.gameMail.playersRequired'))
    return
  }
  if (!sendDialog.title) {
    msgInfo(t('admin.gameMail.titleRequired'))
    return
  }
  const attachments: GmMailAttachment[] = sendDialog.attachments
    .filter(a => (a.cfgId ?? 0) > 0)
    .map(a => ({ type: a.type, cfgId: a.cfgId, count: a.count }))

  sendDialog.loading = true
  let ok = 0
  for (const pid of sendDialog.playerIds) {
    const res = await sendGmMail({
      playerId: pid,
      title: sendDialog.title,
      content: sendDialog.content,
      attachments
    }).catch(() => null)
    if (res) ok++
  }
  sendDialog.loading = false

  if (ok === 0) {
    msgInfo(t('admin.gameMail.sendFailed'))
    return
  }
  if (ok < sendDialog.playerIds.length) {
    msgInfo(t('admin.gameMail.sendPartial', { ok, total: sendDialog.playerIds.length }))
  } else {
    msgSuccess(t('admin.gameMail.sendOk'))
  }
  sendDialog.visible = false
  if (senderIncludesCurrentPlayer()) {
    await loadList()
  }
}

const senderIncludesCurrentPlayer = () => {
  const id = playerIdOrZero()
  return id > 0 && sendDialog.playerIds.includes(id)
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
