<template>
  <PageContainer>
    <ProTable :table="table" row-key="playerId" :default-sort="{ prop: 'createdAt', order: 'descending' }">
      <template #search>
        <el-form-item :label="$t('admin.player.username')">
          <el-input
            v-model="table.query.username"
            :placeholder="$t('admin.player.usernamePlaceholder')"
            clearable
          />
        </el-form-item>
      </template>

      <template #toolbar>
        <el-button
          v-permisaction="['game:player:list']"
          type="primary"
          @click="openOnline"
        >{{ $t('admin.player.online') }}</el-button>
      </template>

      <el-table-column
        :label="$t('admin.player.playerId')"
        prop="playerId"
        min-width="150"
        show-overflow-tooltip
      />
      <el-table-column
        :label="$t('admin.player.username')"
        prop="username"
        min-width="120"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('admin.player.zone')" prop="zone" min-width="80" />
      <el-table-column :label="$t('admin.player.phoneNumber')" prop="phoneNumber" min-width="120" />
      <el-table-column
        :label="$t('admin.player.createdAt')"
        prop="createdAt"
        min-width="150"
        sortable="custom"
      >
        <template #default="{ row }"><DateCell :value="row.createdAt" pattern="{y}-{m}-{d} {h}:{i}" /></template>
      </el-table-column>
    </ProTable>

    <el-dialog
      v-model="online.visible"
      :title="$t('admin.player.online')"
      width="480px"
      @open="loadOnline"
    >
      <el-table v-loading="online.loading" :data="online.rows" max-height="420">
        <el-table-column
          :label="$t('admin.player.playerId')"
          prop="playerId"
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column :label="$t('admin.player.enterTime')" min-width="150">
          <template #default="{ row }"><DateCell :value="row.enterTime" pattern="{y}-{m}-{d} {h}:{i}:{s}" /></template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" min-width="140">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:player:list']"
              link
              type="primary"
              @click="kick(row, 1)"
            >{{ $t('admin.player.kick') }}</el-button>
            <el-button
              v-permisaction="['game:player:list']"
              link
              type="danger"
              @click="kick(row, 2)"
            >{{ $t('admin.player.banKick') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable } from '@/composables'
import { msgSuccess } from '@/utils/message'

import { listPlayer, listOnlinePlayer, kickPlayer } from '@/api/admin/player'
import type { Player, PlayerQuery, OnlinePlayer } from '@/types/admin'

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
  defaultQuery: () => ({ username: undefined })
})

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
  // The interceptor has already reported any failure; the catch just keeps the
  // rejection from surfacing as an unhandled one on the dialog's open event.
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
  // The interceptor has already reported any failure; null means we stop here.
  const res = await kickPlayer({ playerId: row.playerId!, nodeId: row.nodeId!, kickType }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.player.kickOk'))
  await loadOnline()
}
</script>
