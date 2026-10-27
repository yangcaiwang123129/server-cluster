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
      </el-table>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { reactive } from 'vue'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import DateCell from '@/components/DateCell/index.vue'
import { useTable } from '@/composables'

import { listPlayer, listOnlinePlayer } from '@/api/admin/player'
import type { Player, PlayerQuery, OnlinePlayer } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'PlayerManage' })

/**
 * Read-only module: the admin views accounts and live presence, it does not
 * write them. One dialog for the online list, fetched fresh on each open --
 * presence goes stale fast, so there is nothing to cache.
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
</script>
