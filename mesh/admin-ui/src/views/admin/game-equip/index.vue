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
        <el-form-item :label="$t('admin.gameEquip.heroId')">
          <el-input
            v-model="heroId"
            :placeholder="$t('admin.gameEquip.heroIdPlaceholder')"
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

      <el-table v-loading="loading" :data="rows" row-key="id" border>
        <el-table-column prop="name" :label="$t('admin.gameEquip.name')" min-width="180">
          <template #default="{ row }">
            <div class="item-cell">
              <GameItemIcon :name="row.name" tone="#f59e0b" :size="28" />
              <span class="item-name">{{ row.name || '-' }}</span>
              <span class="item-id">#{{ row.equipId }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" :label="$t('admin.gameEquip.instanceId')" min-width="180" show-overflow-tooltip />
        <el-table-column :label="$t('admin.gameEquip.heroId')" min-width="110">
          <template #default="{ row }">{{ row.heroId || '-' }}</template>
        </el-table-column>
        <el-table-column prop="level" :label="$t('admin.gameEquip.level')" min-width="90" />
        <el-table-column prop="quality" :label="$t('admin.gameEquip.quality')" min-width="90" />
        <el-table-column prop="breakthrough" :label="$t('admin.gameEquip.breakthrough')" min-width="90" />
        <el-table-column
          prop="desc"
          :label="$t('admin.gameEquip.desc')"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column :label="$t('admin.gameEquip.equipped')" min-width="90">
          <template #default="{ row }">{{ row.equipped ? '✓' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:gm:equip:list']"
              link
              type="primary"
              @click="openEdit(row)"
            >{{ $t('common.edit') }}</el-button>
            <el-button
              v-if="row.equipped"
              v-permisaction="['game:gm:equip:list']"
              link
              type="warning"
              @click="unwear(row)"
            >{{ $t('admin.gameEquip.unwear') }}</el-button>
            <el-button
              v-else
              v-permisaction="['game:gm:equip:list']"
              link
              type="success"
              @click="wear(row)"
            >{{ $t('admin.gameEquip.wear') }}</el-button>
            <el-dropdown
              v-permisaction="['game:gm:equip:list']"
              trigger="click"
              @command="(cmd: string) => grow(row, cmd as GrowKind)"
            >
              <el-button link type="primary">{{ $t('admin.gameEquip.more') }}</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="upgrade">{{ $t('admin.gameEquip.upgrade') }}</el-dropdown-item>
                  <el-dropdown-item command="upQuality">{{ $t('admin.gameEquip.upQuality') }}</el-dropdown-item>
                  <el-dropdown-item command="breakthrough">
                    {{ $t('admin.gameEquip.breakthrough') }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
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
        <el-form-item :label="$t('admin.gameEquip.instanceId')">
          <div class="item-cell">
            <span class="item-id">{{ editDialog.id }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.level')">
          <el-input-number v-model="editDialog.level" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.quality')">
          <el-input-number v-model="editDialog.quality" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.breakthrough')">
          <el-input-number v-model="editDialog.breakthrough" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameEquip.heroId')">
          <el-input-number v-model="editDialog.heroId" :min="0" />
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

import {
  addGmEquip,
  breakthroughGmEquip,
  listGmEquip,
  unwearGmEquip,
  updateGmEquip,
  upQualityGmEquip,
  upgradeGmEquip,
  wearGmEquip
} from '@/api/admin/gm'
import type { GameCatalogEntry, GmEquip } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameEquipManage' })

const { t } = useI18n()

const playerId = ref('')
const heroId = ref('')
const loading = ref(false)
const rows = ref<GmEquip[]>([])

const editDialog = reactive({
  visible: false,
  loading: false,
  id: '',
  equipId: 0,
  heroId: 0,
  name: '',
  level: 1,
  quality: 1,
  breakthrough: 0,
  equipped: false
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

// 0 表示由服务端挂到当前出战英雄
const heroIdOrZero = () => {
  const id = Number(heroId.value)
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
  const res = await addGmEquip({
    playerId: id,
    equipId: entry.id!,
    heroId: heroIdOrZero()
  }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.addOk'))
  await loadList()
}

const openEdit = (row: GmEquip) => {
  editDialog.id = row.id ?? ''
  editDialog.equipId = row.equipId ?? 0
  editDialog.heroId = row.heroId ?? 0
  editDialog.name = row.name ?? ''
  editDialog.level = row.level ?? 1
  editDialog.quality = row.quality ?? 1
  editDialog.breakthrough = row.breakthrough ?? 0
  editDialog.equipped = row.equipped ?? false
  editDialog.visible = true
}

const submitEdit = async() => {
  const id = playerIdOrZero()
  if (!id) return
  editDialog.loading = true
  const res = await updateGmEquip({
    playerId: id,
    id: editDialog.id,
    heroId: editDialog.heroId,
    level: editDialog.level,
    quality: editDialog.quality,
    breakthrough: editDialog.breakthrough,
    equipped: editDialog.equipped
  }).catch(() => null)
  editDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameEquip.updateOk'))
  editDialog.visible = false
  await loadList()
}

// 维度成长：升级/升品/突破都消耗背包素材，拦截器已统一报错。
const growApi = {
  upgrade: upgradeGmEquip,
  upQuality: upQualityGmEquip,
  breakthrough: breakthroughGmEquip
} as const

type GrowKind = keyof typeof growApi

const grow = async(row: GmEquip, kind: GrowKind) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await growApi[kind]({ playerId: id, equipId: row.id ?? '' }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.growOk'))
  await loadList()
}

// 穿戴到顶部英雄输入指定的英雄（0 = 出战英雄），同英雄同配置自动顶替。
const wear = async(row: GmEquip) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await wearGmEquip({
    playerId: id,
    id: row.id ?? '',
    heroId: heroIdOrZero()
  }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.wearOk'))
  await loadList()
}

// 卸载只清穿戴状态，装备归属英雄保留。
const unwear = async(row: GmEquip) => {
  const id = playerIdOrZero()
  if (!id) return
  const res = await unwearGmEquip({ playerId: id, id: row.id ?? '' }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameEquip.unwearOk'))
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
