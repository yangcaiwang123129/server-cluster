<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameSkill.title') }}</span>
          <span class="hint">{{ $t('admin.gameSkill.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameSkill.playerId')">
          <el-input
            v-model="playerId"
            :placeholder="$t('admin.gameSkill.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadList">
            {{ $t('admin.gameSkill.list') }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="rows" border>
        <el-table-column prop="name" :label="$t('admin.gameSkill.name')" min-width="180">
          <template #default="{ row }">
            <div class="item-cell">
              <GameItemIcon :name="row.name" tone="#a855f7" :size="28" />
              <span class="item-name">{{ row.name || '-' }}</span>
              <span class="item-id">#{{ row.skillId }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="level" :label="$t('admin.gameSkill.level')" min-width="90" />
        <el-table-column
          prop="desc"
          :label="$t('admin.gameSkill.desc')"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column :label="$t('admin.gameSkill.equipped')" min-width="90">
          <template #default="{ row }">{{ row.equipped ? '✓' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:gm:skill:list']"
              link
              type="primary"
              @click="openEdit(row)"
            >{{ $t('common.edit') }}</el-button>
            <el-button
              v-permisaction="['game:gm:skill:list']"
              link
              type="warning"
              @click="upgrade(row)"
            >{{ $t('admin.gameSkill.upgrade') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameSkill.catalogTitle') }}</el-divider>

    <GameCatalogList
      type="skill"
      tone="#a855f7"
      permission="game:gm:skill:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="editDialog.visible" :title="$t('admin.gameSkill.editTitle')" width="420px">
      <el-form label-width="100px">
        <el-form-item :label="$t('admin.gameSkill.skillId')">
          <div class="item-cell">
            <GameItemIcon :name="editDialog.name" tone="#a855f7" :size="22" />
            <span class="item-name">{{ editDialog.name || '-' }}</span>
            <span class="item-id">#{{ editDialog.skillId }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.gameSkill.level')">
          <el-input-number v-model="editDialog.level" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameSkill.equipped')">
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

import { addGmSkill, listGmSkill, updateGmSkill, upgradeGmSkill } from '@/api/admin/gm'
import type { GameCatalogEntry, GmSkill } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameSkillManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmSkill[]>([])

const editDialog = reactive({
  visible: false,
  loading: false,
  skillId: 0,
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
    msgInfo(t('admin.gameSkill.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmSkill({ playerId: id }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data ?? []
}

const addFromCatalog = async(entry: GameCatalogEntry) => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameSkill.playerIdInvalid'))
    return
  }
  const res = await addGmSkill({ playerId: id, skillId: entry.id! }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameSkill.addOk'))
  await loadList()
}

const openEdit = (row: GmSkill) => {
  editDialog.skillId = row.skillId ?? 0
  editDialog.name = row.name ?? ''
  editDialog.level = row.level ?? 1
  editDialog.equipped = row.equipped ?? false
  editDialog.visible = true
}

const submitEdit = async() => {
  const id = playerIdOrZero()
  if (!id) return
  editDialog.loading = true
  const res = await updateGmSkill({
    playerId: id,
    skillId: editDialog.skillId,
    level: editDialog.level,
    equipped: editDialog.equipped
  }).catch(() => null)
  editDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameSkill.updateOk'))
  editDialog.visible = false
  await loadList()
}

// Upgrade consumes upgrade materials from the backpack; the interceptor reports failures.
const upgrade = async(row: GmSkill) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await upgradeGmSkill({ playerId: id, cfgId: row.skillId ?? 0 }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameSkill.upgradeOk'))
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
