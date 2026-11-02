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

    <el-dialog
      v-model="sendDialog.visible"
      :title="$t('admin.gameMail.sendTitle')"
      width="820px"
      class="mail-dialog"
      destroy-on-close
    >
      <div class="mail-send">
        <!-- 左侧编辑区 -->
        <section class="mail-edit">
          <el-form label-position="top">
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
              <el-input
                v-model="sendDialog.title"
                :placeholder="$t('admin.gameMail.titlePlaceholder')"
                maxlength="64"
                show-word-limit
              />
            </el-form-item>
            <el-form-item :label="$t('admin.gameMail.contentLabel')">
              <el-input
                v-model="sendDialog.content"
                type="textarea"
                :rows="6"
                :placeholder="$t('admin.gameMail.contentPlaceholder')"
                maxlength="2000"
                show-word-limit
                resize="vertical"
              />
            </el-form-item>

            <el-form-item :label="$t('admin.gameMail.attachments')">
              <div class="attach-list">
                <div
                  v-for="(a, i) in sendDialog.attachments"
                  :key="i"
                  class="attach-card"
                >
                  <div class="attach-card__body">
                    <el-select
                      v-model="a.type"
                      style="width: 110px"
                      @change="onTypeChange(a)"
                    >
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
                      style="flex: 1; min-width: 160px"
                    >
                      <el-option
                        v-for="c in a.options"
                        :key="c.id"
                        :label="`${c.name} #${c.id}`"
                        :value="c.id"
                      />
                    </el-select>
                    <el-input-number
                      v-model="a.count"
                      :min="1"
                      :controls="false"
                      :placeholder="$t('admin.gameMail.attachmentCount')"
                      style="width: 90px"
                    />
                  </div>
                  <el-button
                    link
                    type="danger"
                    class="attach-card__del"
                    :disabled="sendDialog.attachments.length === 1"
                    @click="removeAttachment(i)"
                  >
                    <el-icon><Close /></el-icon>
                  </el-button>
                </div>

                <el-button
                  class="attach-add"
                  link
                  type="primary"
                  @click="addAttachment"
                >
                  <el-icon><Plus /></el-icon>
                  {{ $t('admin.gameMail.addAttachment') }}
                </el-button>
              </div>
            </el-form-item>
          </el-form>
        </section>

        <!-- 右侧预览区 -->
        <aside class="mail-preview">
          <div class="mail-preview__header">
            <span>{{ $t('admin.gameMail.previewTitle') }}</span>
          </div>

          <div class="mail-preview__meta">
            <div class="meta-row">
              <span class="meta-label">{{ $t('admin.gameMail.previewTo') }}</span>
              <div class="meta-value">
                <el-tag
                  v-if="sendDialog.playerIds.length === 0"
                  type="info"
                  effect="plain"
                >{{ $t('admin.gameMail.previewNoRecipient') }}</el-tag>
                <template v-else>
                  <el-tag
                    v-for="pid in sendDialog.playerIds"
                    :key="pid"
                    size="small"
                    class="recipient-tag"
                  >#{{ pid }}</el-tag>
                  <span v-if="sendDialog.playerIds.length > 3" class="recipient-more">
                    {{ $t('admin.gameMail.previewMore', { n: sendDialog.playerIds.length - 3 }) }}
                  </span>
                </template>
              </div>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ $t('admin.gameMail.titleLabel') }}</span>
              <div class="meta-value mail-preview__subject">
                {{ sendDialog.title || $t('admin.gameMail.previewNoSubject') }}
              </div>
            </div>
          </div>

          <div class="mail-preview__body">
            <div class="preview-content">
              {{ sendDialog.content || $t('admin.gameMail.previewEmptyContent') }}
            </div>
          </div>

          <div class="mail-preview__attach">
            <div v-if="attachmentPreviewList.length === 0" class="attach-empty">
              {{ $t('admin.gameMail.noAttachments') }}
            </div>
            <div v-else class="attach-preview-list">
              <div
                v-for="(a, i) in attachmentPreviewList"
                :key="i"
                class="attach-preview-item"
              >
                <el-tag type="warning" effect="light">
                  {{ typeLabel(a.type) }}
                </el-tag>
                <span class="attach-preview-name">#{{ a.cfgId }}</span>
                <span class="attach-preview-count">x{{ a.count }}</span>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <template #footer>
        <div class="mail-footer">
          <div class="mail-footer__summary">
            <span>{{ $t('admin.gameMail.summaryPlayers', { n: sendDialog.playerIds.length }) }}</span>
            <span class="dot" />
            <span>{{ $t('admin.gameMail.summaryAttachments', { n: attachmentPreviewList.length }) }}</span>
          </div>
          <div>
            <el-button @click="sendDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
            <el-button type="primary" :loading="sendDialog.loading" @click="submitSend">
              <el-icon class="el-icon--left"><Promotion /></el-icon>
              {{ $t('admin.gameMail.sendAction') }}
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Close, Plus, Promotion } from '@element-plus/icons-vue'

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
  attachments: [newAttachmentRow()] as AttachmentRow[]
})

function newAttachmentRow(): AttachmentRow {
  return { type: 1, cfgId: undefined, count: 1, options: [], loading: false }
}

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

const removeAttachment = (i: number) => {
  sendDialog.attachments.splice(i, 1)
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

/** 预览区与汇总计数只算填了 cfgId 的有效附件。 */
const attachmentPreviewList = computed(() =>
  sendDialog.attachments.filter(a => (a.cfgId ?? 0) > 0)
)

const submitSend = async() => {
  if (!sendDialog.playerIds.length) {
    msgInfo(t('admin.gameMail.playersRequired'))
    return
  }
  if (!sendDialog.title) {
    msgInfo(t('admin.gameMail.titleRequired'))
    return
  }
  const attachments: GmMailAttachment[] = attachmentPreviewList.value.map(a => ({
    type: a.type,
    cfgId: a.cfgId,
    count: a.count
  }))

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

/* ------- 发送弹窗 ------- */
.mail-dialog :deep(.el-dialog__body) {
  padding-top: 8px;
  padding-bottom: 4px;
}

.mail-send {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 20px;
  min-height: 420px;
}

/* 左侧编辑区 */
.mail-edit {
  min-width: 0;
}
.mail-edit :deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

/* 附件卡片化 */
.attach-list {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.attach-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.attach-card:hover {
  border-color: var(--el-color-primary-light-5);
}
.attach-card__body {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.attach-card__del {
  flex-shrink: 0;
}

.attach-add {
  align-self: flex-start;
  gap: 4px;
}

/* 右侧预览区 */
.mail-preview {
  display: flex;
  flex-direction: column;
  border: 1px dashed var(--el-border-color);
  border-radius: 10px;
  background: var(--el-bg-color);
  min-height: 420px;
  overflow: hidden;
}
.mail-preview__header {
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border-bottom: 1px dashed var(--el-border-color);
}
.mail-preview__meta {
  padding: 12px 16px 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.meta-row {
  display: flex;
  gap: 8px;
  font-size: 13px;
  line-height: 1.5;
}
.meta-label {
  flex-shrink: 0;
  width: 46px;
  color: var(--el-text-color-secondary);
}
.meta-value {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  color: var(--el-text-color-primary);
}
.mail-preview__subject {
  font-weight: 500;
  word-break: break-all;
}
.recipient-tag {
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.recipient-more {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.mail-preview__body {
  flex: 1;
  padding: 14px 16px;
  overflow: auto;
  background:
    linear-gradient(var(--el-mask-color-extra-light), var(--el-mask-color-extra-light)),
    var(--el-bg-color);
}
.preview-content {
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.7;
}

.mail-preview__attach {
  padding: 10px 16px 12px;
  border-top: 1px dashed var(--el-border-color-lighter);
  min-height: 48px;
}
.attach-empty {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.attach-preview-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.attach-preview-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.attach-preview-name {
  color: var(--el-text-color-primary);
}
.attach-preview-count {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

/* footer */
.mail-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}
.mail-footer__summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.mail-footer__summary .dot {
  display: inline-block;
  width: 4px;
  height: 4px;
  margin: 0 8px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
  vertical-align: middle;
}

/* 窄屏：两栏改单栏 */
@media (max-width: 900px) {
  .mail-send {
    grid-template-columns: 1fr;
  }
  .mail-preview {
    order: -1;
  }
}
</style>
