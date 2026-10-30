<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameEquip.title') }}</span>
          <span class="hint">{{ $t('admin.gameEquip.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameEquip.playerId')">
          <el-input
            v-model="playerId"
            :placeholder="$t('admin.gameEquip.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadList">
            {{ $t('admin.gameEquip.list') }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="rows" border>
        <el-table-column prop="name" :label="$t('admin.gameEquip.name')" min-width="180">
          <template #default="{ row }">
            <div class="item-cell">
              <GameItemIcon :name="row.name" tone="#f59e0b" :size="28" />
              <span class="item-name">{{ row.name || '-' }}</span>
              <span class="item-id">#{{ row.equipId }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="level" :label="$t('admin.gameEquip.level')" min-width="90" />
        <el-table-column
          prop="desc"
          :label="$t('admin.gameEquip.desc')"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column :label="$t('admin.gameEquip.equipped')" min-width="90">
          <template #default="{ row }">{{ row.equipped ? '✓' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:gm:equip:list']"
              link
              type="primary"
              @click="openEdit(row)"
            >{{ $t('common.edit') }}</el-button>
            <el-button
              v-permisaction="['game:gm:equip:list']"
              link
              type="warning"
              @click="upgrade(row)"
            >{{ $t('admin.gameEquip.upgrade') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameEquip.catalogTitle') }}</el-divider>

    <GameCatalogList
      type="equip"
      tone="#f59e0b"
      permission="game:gm:equip:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="editDialog.visible" :title="$t('admin.gameEquip.editTitle')" width="420px">
      <el-form label-width="100px">
        <el-form-item :label="$t('admin.gameEquip.equipId')">
          <div class="item-cell">
            <GameItemIcon :name="editDialog.name" tone="#f59e0b" :size="22" />
            <span class="item-name">{{ editDialog.name || '-' }}</span>
            <span class="item-id">#{{ editDialog.equipId }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.level')">
          <el-input-number v-model="editDialog.level" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.equipped')">
          <el-switch v-model="editDialog.equipped" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="editDialog.loading" @click="submitEdit">
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
import GameItemIcon from '@/components/GameItemIcon/index.vue'
import GameCatalogList from '@/components/GameCatalogList/index.vue'
import { msgInfo, msgSuccess } from '@/utils/message'

import { addGmEquip, listGmEquip, updateGmEquip, upgradeGmEquip } from '@/api/admin/gm'
import type { GameCatalogEntry, GmEquip } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameEquipManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmEquip[]>([])

const editDialog = reactive({
  visible: false,
  loading: false,
  equipId: 0,
  name: '',
  level: 1,
  equipped: false
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const loadList = async() => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameEquip.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmEquip({ playerId: id }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data ?? []
}

const addFromCatalog = async(entry: GameCatalogEntry) => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameEquip.playerIdInvalid'))
    return
  }
  const res = await addGmEquip({ playerId: id, equipId: entry.id! }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.addOk'))
  await loadList()
}

const openEdit = (row: GmEquip) => {
  editDialog.equipId = row.equipId ?? 0
  editDialog.name = row.name ?? ''
  editDialog.level = row.level ?? 1
  editDialog.equipped = row.equipped ?? false
  editDialog.visible = true
}

const submitEdit = async() => {
  const id = playerIdOrZero()
  if (!id) return
  editDialog.loading = true
  const res = await updateGmEquip({
    playerId: id,
    equipId: editDialog.equipId,
    level: editDialog.level,
    equipped: editDialog.equipped
  }).catch(() => null)
  editDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameEquip.updateOk'))
  editDialog.visible = false
  await loadList()
}

// Upgrade consumes upgrade materials from the backpack; the interceptor reports failures.
const upgrade = async(row: GmEquip) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await upgradeGmEquip({ playerId: id, cfgId: row.equipId ?? 0 }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.upgradeOk'))
  await loadList()
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

.item-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-id {
  flex: none;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
