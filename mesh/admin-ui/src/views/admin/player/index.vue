<template>
  <PageContainer>
    <ProTable
      :table="table"
      row-key="playerId"
      :default-sort="{ prop: 'createdAt', order: 'descending' }"
    >
      <template #search>
        <el-form-item :label="$t('admin.player.playerId')">
          <el-input
            v-model.number="table.query.playerId"
            :placeholder="$t('admin.player.playerIdPlaceholder')"
            clearable
            @keyup.enter="table.search"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.player.username')">
          <el-input
            v-model="table.query.username"
            :placeholder="$t('admin.player.usernamePlaceholder')"
            clearable
            @keyup.enter="table.search"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.player.phoneNumber')">
          <el-input
            v-model="table.query.phoneNumber"
            :placeholder="$t('admin.player.phoneNumberPlaceholder')"
            clearable
            @keyup.enter="table.search"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.player.status')">
          <el-select
            v-model="table.query.status"
            placeholder=""
            clearable
            style="width: 140px"
          >
            <el-option label="全部" :value="0" />
            <el-option :label="$t('admin.player.statusNormal')" :value="1" />
            <el-option :label="$t('admin.player.statusBanned')" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.player.createdAt')">
          <el-date-picker
            v-model="createdAtRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始"
            end-placeholder="结束"
            value-format="x"
            style="width: 340px"
            @change="onCreatedAtChange"
          />
        </el-form-item>
      </template>

      <template #toolbar>
        <el-tooltip :content="$t('admin.player.onlineTip')" placement="top">
          <el-button
            v-permisaction="['game:player:list']"
            type="primary"
            @click="openOnline"
          >
            <el-icon class="el-icon--left"><Connection /></el-icon>
            {{ $t('admin.player.online') }}
            <el-badge v-if="online.rows.length > 0" :value="online.rows.length" class="online-badge" />
          </el-button>
        </el-tooltip>
        <el-tooltip :content="$t('common.refresh')" placement="top">
          <el-button
            v-permisaction="['game:player:list']"
            circle
            :loading="table.loading"
            @click="table.search"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
      </template>

      <template #empty>
        <el-empty :description="$t('admin.player.emptyTip')" :image-size="120">
          <el-button type="primary" @click="resetSearch">
            {{ $t('common.reset') }}
          </el-button>
        </el-empty>
      </template>

      <el-table-column :label="$t('admin.player.playerId')" prop="playerId" width="180">
        <template #default="{ row }">
          <div class="cell-pid">
            <el-tag effect="light" type="info" class="pid-tag">
              {{ row.playerId }}
            </el-tag>
            <el-tooltip :content="$t('admin.player.copyPlayerId')" placement="top">
              <el-button
                link
                class="copy-btn"
                @click.stop="copyText(String(row.playerId))"
              >
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>

      <el-table-column
        :label="$t('admin.player.username')"
        prop="username"
        min-width="130"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span>{{ row.username || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column
        :label="$t('admin.player.nickname')"
        prop="nickname"
        min-width="130"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span :class="{ muted: !row.nickname }">{{ row.nickname || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.player.zone')" prop="zone" width="72" align="center">
        <template #default="{ row }">
          <el-tag size="small" effect="plain" class="zone-tag" :type="zoneType(row.zone)">
            +{{ row.zone ?? 0 }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.player.phoneNumber')" prop="phoneNumber" width="110">
        <template #default="{ row }">
          <span v-if="row.phoneNumber">{{ row.phoneNumber }}</span>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.player.level')" prop="level" width="70" align="center">
        <template #default="{ row }">
          <span class="level-badge">Lv.{{ row.level ?? 1 }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.player.vipLevel')" prop="vipLevel" width="78" align="center">
        <template #default="{ row }">
          <el-tag
            v-if="(row.vipLevel ?? 1) > 1"
            type="warning"
            size="small"
            effect="light"
          >VIP{{ row.vipLevel }}</el-tag>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>

      <el-table-column
        :label="$t('admin.player.currencies')"
        min-width="200"
        class-name="col-currencies"
      >
        <template #default="{ row }">
          <div class="cell-currencies">
            <span class="cur-item cur-item--coupon">
              <el-icon><Present /></el-icon>
              {{ formatNumber(row.coupon) }}
            </span>
            <span class="cur-item cur-item--gold">
              <el-icon><Coin /></el-icon>
              {{ formatNumber(row.gold) }}
            </span>
            <span class="cur-item cur-item--coin">
              <el-icon><Medal /></el-icon>
              {{ formatNumber(row.coin) }}
            </span>
          </div>
        </template>
      </el-table-column>

      <el-table-column
        :label="$t('admin.player.registerIp')"
        prop="registerIp"
        min-width="150"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <div v-if="row.registerIp" class="cell-ip">
            <el-icon class="cell-ip__icon"><Location /></el-icon>
            <span class="cell-ip__value">{{ row.registerIp }}</span>
            <el-tooltip :content="$t('admin.player.copyIp')" placement="top">
              <el-button link class="copy-btn" @click.stop="copyText(row.registerIp!)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>

      <el-table-column
        :label="$t('admin.player.createdAt')"
        prop="createdAt"
        min-width="160"
        sortable="custom"
      >
        <template #default="{ row }">
          <DateCell :value="row.createdAt" pattern="{y}-{m}-{d} {h}:{i}" />
        </template>
      </el-table-column>

      <el-table-column :label="$t('common.operation')" width="220" fixed="right" align="center">
        <template #default="{ row }">
          <div class="cell-op">
            <el-button
              v-if="(row.status ?? 1) !== 2"
              v-permisaction="['game:player:list']"
              link
              type="danger"
              size="small"
              @click="openBan(row)"
            >{{ $t('admin.player.ban') }}</el-button>
            <el-button
              v-else
              v-permisaction="['game:player:list']"
              link
              type="success"
              size="small"
              @click="submitUnban(row)"
            >{{ $t('admin.player.unban') }}</el-button>
            <el-button
              v-permisaction="['game:player:list']"
              link
              type="primary"
              size="small"
              @click="openResetPwd(row)"
            >{{ $t('admin.player.resetPwd') }}</el-button>
          </div>
        </template>
      </el-table-column>
    </ProTable>

    <!-- 在线玩家弹窗：卡片化展示 + 批量踢下线 -->
    <el-dialog
      v-model="online.visible"
      :title="$t('admin.player.online')"
      width="680px"
      class="player-online-dialog"
      destroy-on-close
      @open="loadOnline"
    >
      <template #header>
        <div class="online-dialog-header">
          <span class="online-dialog-title">{{ $t('admin.player.online') }}</span>
          <el-badge
            v-if="!online.loading"
            :value="online.rows.length"
            :max="9999"
            type="success"
            class="online-badge-header"
          />
          <el-button
            link
            class="online-refresh"
            :loading="online.loading"
            @click="loadOnline"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </template>

      <div v-if="online.rows.length === 0 && !online.loading" class="online-empty">
        <el-empty :description="$t('admin.player.onlineEmpty')" :image-size="100" />
      </div>

      <div v-else class="online-list">
        <div
          v-for="p in online.rows"
          :key="p.playerId"
          class="online-card"
        >
          <div class="online-card__body">
            <div class="online-card__pid">
              <el-tag type="success" effect="light" round size="small">#{{ p.playerId }}</el-tag>
            </div>
            <div class="online-card__enter">
              <el-icon class="muted"><Clock /></el-icon>
              <DateCell :value="p.enterTime" pattern="{y}-{m}-{d} {h}:{i}:{s}" />
            </div>
            <div class="online-card__node">
              <el-icon class="muted"><Monitor /></el-icon>
              <span class="mono">{{ p.lobbyNodeId || '-' }}</span>
            </div>
            <div v-if="p.loginIp" class="online-card__ip">
              <el-icon class="muted"><Location /></el-icon>
              <span class="mono">{{ p.loginIp }}</span>
            </div>
          </div>
          <div class="online-card__actions">
            <el-button
              link
              type="warning"
              size="small"
              @click="kick(p, 1)"
            >{{ $t('admin.player.kick') }}</el-button>
            <el-button
              link
              type="danger"
              size="small"
              @click="kick(p, 2)"
            >{{ $t('admin.player.banKick') }}</el-button>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="online.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button
          v-permisaction="['game:player:list']"
          type="primary"
          :disabled="online.rows.length === 0"
          :loading="online.loading"
          @click="kickAll"
        >
          {{ $t('admin.player.kickAll', { n: online.rows.length }) }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 封号弹窗 -->
    <el-dialog
      v-model="banDialog.visible"
      :title="$t('admin.player.ban')"
      width="420px"
      destroy-on-close
    >
      <el-form
        ref="banFormRef"
        :model="banForm"
        :rules="banRules"
        label-width="80px"
      >
        <el-form-item :label="$t('admin.player.playerId')">
          <el-tag effect="light" type="info">{{ banForm.playerId }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('admin.player.banDuration')" prop="duration">
          <el-select
            v-model="banForm.duration"
            placeholder=""
            style="width: 100%"
          >
            <el-option
              v-for="opt in banDurationOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.player.banReason')" prop="reason">
          <el-input
            v-model="banForm.reason"
            type="textarea"
            :rows="3"
            :placeholder="$t('admin.player.banReasonPlaceholder')"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="banDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="danger" @click="submitBan">{{ $t('admin.player.ban') }}</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码弹窗 -->
    <el-dialog
      v-model="pwdDialog.visible"
      :title="$t('admin.player.resetPwd')"
      width="400px"
      destroy-on-close
    >
      <el-form
        ref="pwdFormRef"
        :model="pwdForm"
        :rules="pwdRules"
        label-width="80px"
      >
        <el-form-item :label="$t('admin.player.playerId')">
          <el-tag effect="light" type="info">{{ pwdForm.playerId }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('admin.player.newPassword')" prop="newPassword">
          <el-input
            v-model="pwdForm.newPassword"
            type="password"
            show-password
            :placeholder="$t('admin.player.newPasswordPlaceholder')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" @click="submitResetPwd">{{ $t('admin.player.resetPwd') }}</el-button>
      </template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  Clock,
  Coin,
  Connection,
  CopyDocument,
  Location,
  Medal,
  Monitor,
  Present,
  Refresh
} from '@element-plus/icons-vue'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable } from '@/composables'
import { msgSuccess } from '@/utils/message'

import { banPlayer, kickPlayer, listOnlinePlayer, listPlayer, resetPlayerPassword, unbanPlayer } from '@/api/admin/player'
import type { Player, PlayerQuery, OnlinePlayer, PlayerBanReq, PlayerResetPasswordReq } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'PlayerManage' })

const { t } = useI18n()

// Mirrors pb.KickType: 1 active kick, 2 ban kick.
const KICK_TYPE_BAN = 2

/**
 * The admin views accounts and live presence, and can kick a player offline
 * (plain or ban). One dialog for the online list, fetched fresh on each open
 * -- presence goes stale fast, so there is nothing to cache.
 */
const table = useTable<Player, PlayerQuery>({
  api: listPlayer,
  idKey: 'playerId',
  defaultSort: { prop: 'createdAt', order: 'descending' },
  defaultQuery: () => ({ playerId: undefined, username: undefined, phoneNumber: undefined, status: 0, createdAtStart: undefined, createdAtEnd: undefined })
})

// 创建时间范围筛选：el-date-picker datetimerange 的 value-format="x" 给毫秒字符串
const createdAtRange = ref<string[]>([])
const onCreatedAtChange = (val: string[] | null) => {
  if (!val || val.length === 0) {
    table.query.createdAtStart = undefined
    table.query.createdAtEnd = undefined
  } else {
    table.query.createdAtStart = Number(val[0])
    table.query.createdAtEnd = Number(val[1])
  }
}

const online = reactive({
  visible: false,
  loading: false,
  rows: [] as OnlinePlayer[]
})

const openOnline = () => {
  online.visible = true
}

const loadOnline = async() => {
  online.loading = true
  const res = await listOnlinePlayer().catch(() => null)
  online.rows = res?.data ?? []
  online.loading = false
}

const kick = async(row: OnlinePlayer, kickType: number) => {
  const message = kickType === KICK_TYPE_BAN
    ? t('admin.player.banKickConfirm', { playerId: row.playerId })
    : t('admin.player.kickConfirm', { playerId: row.playerId })
  try {
    await ElMessageBox.confirm(message, t('common.notice'), { type: 'warning' })
  } catch {
    return
  }
  const res = await kickPlayer({ playerId: row.playerId!, nodeId: row.lobbyNodeId!, kickType }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.player.kickOk'))
  await loadOnline()
}

const kickAll = async() => {
  try {
    await ElMessageBox.confirm(t('admin.player.kickAllConfirm', { n: online.rows.length }), t('common.notice'), { type: 'warning' })
  } catch {
    return
  }
  // 并行踢下线：每个玩家独立请求，收集成功数给前端反馈
  let ok = 0
  await Promise.all(online.rows.map(p =>
    kickPlayer({ playerId: p.playerId!, nodeId: p.lobbyNodeId!, kickType: 1 }).then(r => r && ok++).catch(() => null)
  ))
  if (ok === online.rows.length) {
    msgSuccess(t('admin.player.kickAllOk'))
  } else {
    ElMessage.warning(t('admin.player.kickAllPartial', { ok, total: online.rows.length }))
  }
  await loadOnline()
}

// ── 辅助展示函数 ──────────────────────────────────────────────

const zoneType = (zone?: number): 'primary' | 'success' | 'warning' | 'info' => {
  if (!zone || zone === 0) return 'info'
  if (zone <= 10) return 'primary'
  if (zone <= 50) return 'success'
  return 'warning'
}

// 货币千分位 + 10^4/10^8 简写（大数字友好）
const formatNumber = (v?: number) => {
  const n = v ?? 0
  const abs = Math.abs(n)
  if (abs >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (n / 1e4).toFixed(1) + '万'
  return n.toLocaleString()
}

const copyText = async(text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('admin.player.copyOk'))
  } catch {
    // 降级方案：动态创建 textarea
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      ElMessage.success(t('admin.player.copyOk'))
    } catch {
      ElMessage.error(t('admin.player.copyFailed'))
    }
    document.body.removeChild(ta)
  }
}

const resetSearch = () => {
  table.resetQuery()
}

// ── 封号 / 解封 / 改密 状态 ────────────────────────────────────

// 封号弹窗
const banFormRef = ref<FormInstance>()
const banDialog = reactive({ visible: false, player: null as Player | null })
const banForm = reactive<PlayerBanReq>({ playerId: 0, duration: 0, reason: '' })
const banRules: FormRules = {
  reason: [{ required: true, message: () => t('admin.player.banReasonRequired'), trigger: 'blur' }]
}
// duration 预设选项
const banDurationOptions = [
  { label: '1 小时', value: 3600 },
  { label: '24 小时', value: 86400 },
  { label: '7 天', value: 7 * 86400 },
  { label: '30 天', value: 30 * 86400 },
  { label: '永久', value: 0 }
]

const openBan = (row: Player) => {
  banDialog.player = row
  banForm.playerId = row.playerId!
  banForm.duration = 86400
  banForm.reason = ''
  banDialog.visible = true
}

const submitBan = async() => {
  if (!banFormRef.value || !banDialog.player) return
  await banFormRef.value.validate()
  const res = await banPlayer({ ...banForm }).catch(() => null)
  if (!res) return
  banDialog.visible = false
  msgSuccess(t('admin.player.banOk'))
  await table.search()
}

// 解封
const submitUnban = async(row: Player) => {
  try {
    await ElMessageBox.confirm(
      t('admin.player.unbanConfirm', { playerId: row.playerId }),
      t('common.notice'),
      { type: 'warning' }
    )
  } catch {
    return
  }
  const res = await unbanPlayer({ playerId: row.playerId! }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.player.unbanOk'))
  await table.search()
}

// 重置密码弹窗
const pwdFormRef = ref<FormInstance>()
const pwdDialog = reactive({ visible: false, player: null as Player | null })
const pwdForm = reactive<PlayerResetPasswordReq>({ playerId: 0, newPassword: '' })
const pwdRules: FormRules = {
  newPassword: [
    { required: true, message: () => t('admin.player.pwdRequired'), trigger: 'blur' },
    { min: 6, max: 32, message: () => t('admin.player.pwdLength'), trigger: 'blur' }
  ]
}

const openResetPwd = (row: Player) => {
  pwdDialog.player = row
  pwdForm.playerId = row.playerId!
  pwdForm.newPassword = ''
  pwdDialog.visible = true
}

const submitResetPwd = async() => {
  if (!pwdFormRef.value || !pwdDialog.player) return
  await pwdFormRef.value.validate()
  const res = await resetPlayerPassword({ ...pwdForm }).catch(() => null)
  if (!res) return
  pwdDialog.visible = false
  msgSuccess(t('admin.player.pwdOk'))
}
</script>

<style scoped>
/* ------- 工具栏 badge ------- */
.online-badge {
  margin-left: 4px;
}

/* ------- 列内展示 ------- */
.cell-pid {
  display: flex;
  align-items: center;
  gap: 4px;
}
.pid-tag {
  font-family: var(--el-font-family-mono, monospace);
  padding: 0 6px;
  height: 24px;
  line-height: 22px;
  font-size: 12px;
}
.copy-btn {
  padding: 2px;
  color: var(--el-text-color-placeholder);
  transition: color 0.15s ease, transform 0.15s ease;
}
.copy-btn:hover {
  color: var(--el-color-primary);
  transform: scale(1.1);
}

.cell-name {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}
.cell-name__primary {
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cell-name__secondary {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 1;
  max-width: 120px;
}

.level-badge {
  font-weight: 600;
  color: var(--el-color-primary);
  font-variant-numeric: tabular-nums;
}

.zone-tag {
  font-variant-numeric: tabular-nums;
}

.cell-ip {
  display: flex;
  align-items: center;
  gap: 4px;
  font-family: var(--el-font-family-mono, monospace);
  font-size: 12.5px;
}
.cell-ip__icon {
  color: var(--el-text-color-placeholder);
}
.cell-ip__value {
  color: var(--el-text-color-primary);
}

.muted {
  color: var(--el-text-color-placeholder);
}

/* 货币列：三个紧凑 chip */
.cell-currencies {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.cur-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.cur-item__icon {
  font-size: 12px;
}
.cur-item--coupon {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}
.cur-item--gold {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
.cur-item--coin {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.cell-op {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
  justify-content: center;
}
.player-online-dialog :deep(.el-dialog__body) {
  padding-top: 8px;
}

.online-dialog-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.online-dialog-title {
  font-weight: 600;
  font-size: 15px;
}
.online-badge-header {
  margin-left: 4px;
}
.online-refresh {
  margin-left: auto;
}

.online-empty {
  padding: 24px 0;
}

.online-list {
  max-height: 420px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 2px;
}

.online-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}
.online-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.06);
}
.online-card__body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
}
.online-card__enter {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12.5px;
  color: var(--el-text-color-secondary);
}
.online-card__node {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.online-card__ip {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.mono {
  font-family: var(--el-font-family-mono, monospace);
}
.online-card__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* 卡片列表滚动条美化 */
.online-list::-webkit-scrollbar {
  width: 6px;
}
.online-list::-webkit-scrollbar-thumb {
  background: var(--el-border-color);
  border-radius: 3px;
}
.online-list::-webkit-scrollbar-thumb:hover {
  background: var(--el-border-color-darker);
}
</style>
