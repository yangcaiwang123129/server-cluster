<template>
  <PageContainer>
    <el-card shadow="never" class="container-card">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.playerItem.containerTitle') }}</span>
          <span v-if="container.loaded" class="grid-size">
            {{ $t('admin.playerItem.gridSize', { w: container.gridW, h: container.gridH }) }}
          </span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.playerItem.playerId')">
          <el-input
            v-model="query.playerId"
            :placeholder="$t('admin.playerItem.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.containerType')">
          <el-select v-model="query.containerType" style="width: 140px">
            <el-option :label="$t('admin.playerItem.backpack')" :value="1" />
            <el-option :label="$t('admin.playerItem.stash')" :value="2" />
            <el-option :label="$t('admin.playerItem.secure')" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadContainer">
            {{ $t('common.search') }}
          </el-button>
          <el-button
            v-permisaction="['game:player:item:list']"
            :disabled="!container.loaded"
            :loading="sorting"
            @click="sortContainer"
          >{{ $t('admin.playerItem.sort') }}</el-button>
        </el-form-item>
      </el-form>

      <div v-if="container.loaded" class="grid-scroll">
        <div class="grid" :style="gridStyle">
          <div v-for="n in cellCount" :key="n" class="grid-cell" />
          <div
            v-for="item in container.items"
            :key="item.itemId"
            class="grid-item"
            :style="itemStyle(item)"
            :title="itemTitle(item)"
            @click="openMove(item)"
          >
            <span class="grid-item-name">{{ item.name || '#' + item.itemCfgId }}</span>
            <span v-if="(item.count ?? 0) > 1" class="grid-item-count">x{{ item.count }}</span>
          </div>
        </div>
      </div>
      <el-empty v-else :description="$t('admin.playerItem.emptyHint')" />
    </el-card>

    <ProTable :table="table" row-key="id">
      <template #search>
        <el-form-item :label="$t('admin.playerItem.keyword')">
          <el-input
            v-model="table.query.keyword"
            :placeholder="$t('admin.playerItem.keywordPlaceholder')"
            clearable
          />
        </el-form-item>
      </template>

      <el-table-column :label="$t('admin.playerItem.itemId')" prop="id" min-width="100" />
      <el-table-column
        :label="$t('admin.playerItem.itemName')"
        prop="name"
        min-width="180"
        show-overflow-tooltip
      />
      <el-table-column :label="$t('admin.playerItem.rarity')" prop="rarity" min-width="80" />
      <el-table-column :label="$t('admin.playerItem.size')" min-width="90">
        <template #default="{ row }">{{ row.width }} × {{ row.height }}</template>
      </el-table-column>
      <el-table-column :label="$t('admin.playerItem.maxCount')" prop="maxCount" min-width="100" />
      <el-table-column :label="$t('common.actions')" min-width="100" fixed="right">
        <template #default="{ row }">
          <el-button
            v-permisaction="['game:player:item:list']"
            link
            type="primary"
            @click="openAdd(row)"
          >{{ $t('admin.playerItem.add') }}</el-button>
        </template>
      </el-table-column>
    </ProTable>

    <el-dialog
      v-model="addDialog.visible"
      :title="$t('admin.playerItem.addTitle')"
      width="420px"
    >
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.playerItem.item')">
          {{ addDialog.name }} (#{{ addDialog.itemCfgId }})
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.count')">
          <el-input-number v-model="addDialog.count" :min="1" :max="99" />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.gridX')">
          <el-input-number v-model="addDialog.gridX" :min="0" :max="container.gridW - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.gridY')">
          <el-input-number v-model="addDialog.gridY" :min="0" :max="container.gridH - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.rotated')">
          <el-switch v-model="addDialog.rotated" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="addDialog.loading" @click="submitAdd">
          {{ $t('common.dialogConfirm') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="moveDialog.visible"
      :title="$t('admin.playerItem.moveTitle')"
      width="420px"
    >
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.playerItem.item')">
          {{ moveDialog.name }} (#{{ moveDialog.itemId }})
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.gridX')">
          <el-input-number v-model="moveDialog.gridX" :min="0" :max="container.gridW - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.gridY')">
          <el-input-number v-model="moveDialog.gridY" :min="0" :max="container.gridH - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.playerItem.rotated')">
          <el-switch v-model="moveDialog.rotated" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="moveDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="moveDialog.loading" @click="submitMove">
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
import ProTable from '@/components/ProTable/index.vue'
import { useTable } from '@/composables'
import { msgInfo, msgSuccess } from '@/utils/message'

import {
  addContainerItem,
  getPlayerContainer,
  listItem,
  moveContainerItem,
  sortPlayerContainer
} from '@/api/admin/item'
import type { ContainerItem, ItemCfg, ItemCfgQuery, PlayerContainer } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'PlayerItemManage' })

const { t } = useI18n()

// One grid cell in pixels; item blocks are positioned by cell count.
const CELL_SIZE = 44

const RARITY_COLORS: Record<number, string> = {
  1: '#9ca3af',
  2: '#22c55e',
  3: '#3b82f6',
  4: '#a855f7',
  5: '#f59e0b'
}

/**
 * Two panels: the player's container grid (left/top) and the Luban item catalog
 * (below). The catalog is the source for adding items; every mutation answers
 * with the updated container, so the grid never needs a second round trip.
 */
const table = useTable<ItemCfg, ItemCfgQuery>({
  api: listItem,
  idKey: 'id',
  defaultQuery: () => ({ keyword: undefined })
})

const query = reactive({
  playerId: '',
  containerType: 1
})

const loading = ref(false)
const sorting = ref(false)

const container = reactive({
  loaded: false,
  gridW: 0,
  gridH: 0,
  items: [] as ContainerItem[]
})

const addDialog = reactive({
  visible: false,
  loading: false,
  itemCfgId: 0,
  name: '',
  count: 1,
  gridX: 0,
  gridY: 0,
  rotated: false
})

const moveDialog = reactive({
  visible: false,
  loading: false,
  itemId: 0,
  name: '',
  gridX: 0,
  gridY: 0,
  rotated: false
})

const gridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${container.gridW}, ${CELL_SIZE}px)`,
  gridTemplateRows: `repeat(${container.gridH}, ${CELL_SIZE}px)`
}))

const cellCount = computed(() => Math.max(0, container.gridW * container.gridH))

const itemStyle = (item: ContainerItem) => ({
  left: `${(item.gridX ?? 0) * CELL_SIZE}px`,
  top: `${(item.gridY ?? 0) * CELL_SIZE}px`,
  width: `${(item.width ?? 1) * CELL_SIZE - 2}px`,
  height: `${(item.height ?? 1) * CELL_SIZE - 2}px`,
  borderColor: RARITY_COLORS[item.rarity ?? 0] ?? 'var(--el-border-color)'
})

const itemTitle = (item: ContainerItem) =>
  `${item.name || '#' + item.itemCfgId} x${item.count ?? 0} (${item.gridX}, ${item.gridY})`

const playerIdOrZero = () => {
  const playerId = Number(query.playerId)
  return Number.isInteger(playerId) && playerId > 0 ? playerId : 0
}

const applyContainer = (data?: PlayerContainer) => {
  if (!data) return
  container.gridW = data.gridW ?? 0
  container.gridH = data.gridH ?? 0
  container.items = data.items ?? []
  container.loaded = true
}

const loadContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.playerItem.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await getPlayerContainer({ playerId, containerType: query.containerType }).catch(() => null)
  loading.value = false
  if (!res) return
  applyContainer(res.data)
}

const sortContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.playerItem.playerIdInvalid'))
    return
  }
  sorting.value = true
  const res = await sortPlayerContainer({ playerId, containerType: query.containerType }).catch(() => null)
  sorting.value = false
  if (!res) return
  msgSuccess(t('admin.playerItem.sortOk'))
  applyContainer(res.data)
}

const openAdd = (row: ItemCfg) => {
  if (!container.loaded) {
    msgInfo(t('admin.playerItem.loadFirst'))
    return
  }
  addDialog.itemCfgId = row.id ?? 0
  addDialog.name = row.name ?? ''
  addDialog.count = 1
  addDialog.gridX = 0
  addDialog.gridY = 0
  addDialog.rotated = false
  addDialog.visible = true
}

const submitAdd = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.playerItem.playerIdInvalid'))
    return
  }
  addDialog.loading = true
  const res = await addContainerItem({
    playerId,
    containerType: query.containerType,
    itemCfgId: addDialog.itemCfgId,
    count: addDialog.count,
    gridX: addDialog.gridX,
    gridY: addDialog.gridY,
    rotated: addDialog.rotated
  }).catch(() => null)
  addDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.playerItem.addOk'))
  addDialog.visible = false
  applyContainer(res.data)
}

const openMove = (item: ContainerItem) => {
  moveDialog.itemId = item.itemId ?? 0
  moveDialog.name = item.name || `#${item.itemCfgId}`
  moveDialog.gridX = item.gridX ?? 0
  moveDialog.gridY = item.gridY ?? 0
  moveDialog.rotated = item.rotated ?? false
  moveDialog.visible = true
}

const submitMove = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.playerItem.playerIdInvalid'))
    return
  }
  moveDialog.loading = true
  const res = await moveContainerItem({
    playerId,
    containerType: query.containerType,
    itemId: moveDialog.itemId,
    gridX: moveDialog.gridX,
    gridY: moveDialog.gridY,
    rotated: moveDialog.rotated
  }).catch(() => null)
  moveDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.playerItem.moveOk'))
  moveDialog.visible = false
  applyContainer(res.data)
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.grid-size {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.grid-scroll {
  overflow: auto;
  max-height: 460px;
  padding: 4px;
}

.grid {
  position: relative;
  display: grid;
  width: max-content;
}

.grid-cell {
  width: 44px;
  height: 44px;
  box-sizing: border-box;
  border: 1px dashed var(--el-border-color-lighter);
}

.grid-item {
  position: absolute;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 2px 4px;
  margin: 1px;
  overflow: hidden;
  font-size: 12px;
  line-height: 1.2;
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.grid-item:hover {
  background: var(--el-fill-color-light);
}

.grid-item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.grid-item-count {
  align-self: flex-end;
  color: var(--el-text-color-secondary);
}
</style>
