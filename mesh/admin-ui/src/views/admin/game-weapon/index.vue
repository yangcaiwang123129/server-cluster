<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameWeapon.title') }}</span>
          <span class="hint">{{ $t('admin.gameWeapon.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameWeapon.playerId')">
          <el-input
            v-model="playerId"
            :placeholder="$t('admin.gameWeapon.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadList">
            {{ $t('admin.gameWeapon.list') }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="rows" border>
        <el-table-column prop="name" :label="$t('admin.gameWeapon.name')" min-width="180">
          <template #default="{ row }">
            <div class="item-cell">
              <GameItemIcon :name="row.name" tone="#0ea5e9" :size="28" />
              <span class="item-name">{{ row.name || '-' }}</span>
              <span class="item-id">#{{ row.weaponId }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="level" :label="$t('admin.gameWeapon.level')" min-width="90" />
        <el-table-column prop="durability" :label="$t('admin.gameWeapon.durability')" min-width="100" />
        <el-table-column :label="$t('admin.gameWeapon.equipped')" min-width="90">
          <template #default="{ row }">{{ row.equipped ? '✓' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:gm:weapon:list']"
              link
              type="primary"
              @click="openEdit(row)"
            >{{ $t('common.edit') }}</el-button>
            <el-button
              v-permisaction="['game:gm:weapon:list']"
              link
              type="warning"
              @click="upgrade(row)"
            >{{ $t('admin.gameWeapon.upgrade') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameWeapon.catalogTitle') }}</el-divider>

    <GameCatalogList
      type="weapon"
      tone="#0ea5e9"
      permission="game:gm:weapon:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="editDialog.visible" :title="$t('admin.gameWeapon.editTitle')" width="420px">
      <el-form label-width="100px">
        <el-form-item :label="$t('admin.gameWeapon.weaponId')">
          <div class="item-cell">
            <GameItemIcon :name="editDialog.name" tone="#0ea5e9" :size="22" />
            <span class="item-name">{{ editDialog.name || '-' }}</span>
            <span class="item-id">#{{ editDialog.weaponId }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.gameWeapon.level')">
          <el-input-number v-model="editDialog.level" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameWeapon.durability')">
          <el-input-number v-model="editDialog.durability" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameWeapon.equipped')">
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

import { addGmWeapon, listGmWeapon, updateGmWeapon, upgradeGmWeapon } from '@/api/admin/gm'
import type { GameCatalogEntry, GmWeapon } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameWeaponManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmWeapon[]>([])

const editDialog = reactive({
  visible: false,
  loading: false,
  weaponId: 0,
  name: '',
  level: 1,
  durability: 0,
  equipped: false
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const loadList = async() => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameWeapon.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmWeapon({ playerId: id }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data ?? []
}

const addFromCatalog = async(entry: GameCatalogEntry) => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameWeapon.playerIdInvalid'))
    return
  }
  const res = await addGmWeapon({ playerId: id, weaponId: entry.id! }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameWeapon.addOk'))
  await loadList()
}

const openEdit = (row: GmWeapon) => {
  editDialog.weaponId = row.weaponId ?? 0
  editDialog.name = row.name ?? ''
  editDialog.level = row.level ?? 1
  editDialog.durability = row.durability ?? 0
  editDialog.equipped = row.equipped ?? false
  editDialog.visible = true
}

const submitEdit = async() => {
  const id = playerIdOrZero()
  if (!id) return
  editDialog.loading = true
  const res = await updateGmWeapon({
    playerId: id,
    weaponId: editDialog.weaponId,
    level: editDialog.level,
    durability: editDialog.durability,
    equipped: editDialog.equipped
  }).catch(() => null)
  editDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameWeapon.updateOk'))
  editDialog.visible = false
  await loadList()
}

// Upgrade consumes upgrade materials from the backpack; the interceptor reports failures.
const upgrade = async(row: GmWeapon) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await upgradeGmWeapon({ playerId: id, cfgId: row.weaponId ?? 0 }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameWeapon.upgradeOk'))
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
