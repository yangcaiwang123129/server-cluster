<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameContainer.title') }}</span>
          <span v-if="container.loaded" class="grid-size">
            {{ containerLabel(query.containerType) }} ·
            {{ $t('admin.gameContainer.gridSize') }} {{ container.gridW }} × {{ container.gridH }}
          </span>
          <span class="hint">{{ $t('admin.gameContainer.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameContainer.playerId')">
          <el-input
            v-model="query.playerId"
            :placeholder="$t('admin.gameContainer.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.containerType')">
          <el-select v-model="query.containerType" style="width: 140px">
            <el-option :label="$t('admin.gameContainer.backpack')" :value="1" />
            <el-option :label="$t('admin.gameContainer.stash')" :value="2" />
            <el-option :label="$t('admin.gameContainer.secure')" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadContainer">
            {{ $t('admin.gameContainer.list') }}
          </el-button>
          <el-button
            v-permisaction="['game:gm:container:list']"
            :disabled="!container.loaded"
            :loading="sorting"
            @click="sortContainer"
          >{{ $t('admin.gameContainer.sort') }}</el-button>
        </el-form-item>
      </el-form>

      <div v-if="container.loaded" ref="gridScrollRef" class="grid-scroll">
        <div ref="gridRef" class="grid" :style="gridStyle">
          <div
            v-if="drag.active && drag.item"
            class="drop-preview"
            :class="{ invalid: !dropValid }"
            :style="dropPreviewStyle"
          />
          <div
            v-for="item in container.items"
            :key="item.itemId"
            class="grid-item"
            :class="{ dragging: drag.item?.itemId === item.itemId && drag.moved }"
            :style="itemStyle(item)"
            :title="itemTitle(item)"
            @pointerdown="onItemPointerDown($event, item)"
          >
            <div class="grid-item-head">
              <GameItemIcon :name="item.name" :rarity="item.rarity" :size="16" />
              <span class="grid-item-name">{{ item.name || '#' + item.itemCfgId }}</span>
            </div>
            <span v-if="(item.count ?? 0) > 1" class="grid-item-count">x{{ item.count }}</span>
          </div>
        </div>
      </div>
      <el-empty v-else :description="$t('admin.gameContainer.emptyHint')" />
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameContainer.itemCatalogTitle') }}</el-divider>

    <GameCatalogList
      type="item"
      permission="game:gm:container:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="actionDialog.visible" :title="actionDialog.name" width="360px">
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('admin.gameContainer.itemId')">
          {{ actionDialog.item?.itemId }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameContainer.itemCfgId')">
          #{{ actionDialog.item?.itemCfgId }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameContainer.count')">
          {{ actionDialog.item?.count }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameContainer.position')">
          ({{ actionDialog.item?.gridX }}, {{ actionDialog.item?.gridY }})
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameContainer.rotated')">
          {{ actionDialog.item?.rotated ? '✓' : '-' }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button
          v-permisaction="['game:gm:container:list']"
          :loading="rotating"
          @click="rotateItem"
        >{{ $t('admin.gameContainer.rotate') }}</el-button>
        <el-button
          v-permisaction="['game:gm:container:list']"
          type="primary"
          @click="openTransfer"
        >{{ $t('admin.gameContainer.transfer') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="transferDialog.visible"
      :title="$t('admin.gameContainer.transfer')"
      width="420px"
    >
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.gameContainer.item')">
          {{ transferDialog.name }}
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.fromType')">
          {{ containerLabel(query.containerType) }}
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.toType')">
          <el-select v-model="transferDialog.toType" style="width: 140px">
            <el-option
              v-for="t in [1, 2, 3].filter(t => t !== query.containerType)"
              :key="t"
              :label="containerLabel(t)"
              :value="t"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.gridX')">
          <el-input-number v-model="transferDialog.toX" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.gridY')">
          <el-input-number v-model="transferDialog.toY" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.rotated')">
          <el-switch v-model="transferDialog.rotated" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="transferDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="primary" :loading="transferDialog.loading" @click="submitTransfer">
          {{ $t('common.dialogConfirm') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="addDialog.visible" :title="$t('admin.gameContainer.addTitle')" width="420px">
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.gameContainer.pickItem')">
          <span>{{ addDialog.name }} (#{{ addDialog.itemCfgId }})</span>
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.count')">
          <el-input-number v-model="addDialog.count" :min="1" :max="999" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.gridX')">
          <el-input-number v-model="addDialog.gridX" :min="0" :max="container.gridW - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.gridY')">
          <el-input-number v-model="addDialog.gridY" :min="0" :max="container.gridH - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameContainer.rotated')">
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
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import PageContainer from '@/components/PageContainer/index.vue'
import GameItemIcon from '@/components/GameItemIcon/index.vue'
import GameCatalogList from '@/components/GameCatalogList/index.vue'
import { msgInfo, msgSuccess } from '@/utils/message'

import {
  addGmContainerItem,
  listGmContainer,
  moveGmContainerItem,
  rotateGmContainerItem,
  sortGmContainer,
  transferGmContainerItem
} from '@/api/admin/gm'
import type { GameCatalogEntry, GmContainer, GmContainerItem } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameContainerManage' })

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

const query = reactive({
  playerId: '',
  containerType: 1
})

const loading = ref(false)
const sorting = ref(false)
const rotating = ref(false)

const container = reactive({
  loaded: false,
  gridW: 0,
  gridH: 0,
  items: [] as GmContainerItem[]
})

const gridRef = ref<HTMLElement>()
const gridScrollRef = ref<HTMLElement>()

/** Drag state: follows cells, not pixels — the item snaps to the hovered cell. */
const drag = reactive({
  active: false,
  moved: false,
  item: null as GmContainerItem | null,
  startClientX: 0,
  startClientY: 0,
  grabDX: 0,
  grabDY: 0,
  hoverX: 0,
  hoverY: 0
})

const actionDialog = reactive({
  visible: false,
  name: '',
  item: null as GmContainerItem | null
})

const transferDialog = reactive({
  visible: false,
  loading: false,
  name: '',
  itemId: '',
  toType: 2,
  toX: 0,
  toY: 0,
  rotated: false
})

const addDialog = reactive({
  visible: false,
  loading: false,
  name: '',
  itemCfgId: undefined as number | undefined,
  count: 1,
  gridX: 0,
  gridY: 0,
  rotated: false
})

const gridStyle = computed(() => ({
  width: `${container.gridW * CELL_SIZE}px`,
  height: `${container.gridH * CELL_SIZE}px`,
  backgroundSize: `${CELL_SIZE}px ${CELL_SIZE}px`
}))

const itemW = (item: GmContainerItem) => item.width ?? 1
const itemH = (item: GmContainerItem) => item.height ?? 1

const itemStyle = (item: GmContainerItem) => {
  const dragging = drag.item?.itemId === item.itemId && drag.moved
  const x = dragging ? drag.hoverX : (item.gridX ?? 0)
  const y = dragging ? drag.hoverY : (item.gridY ?? 0)
  return {
    left: `${x * CELL_SIZE}px`,
    top: `${y * CELL_SIZE}px`,
    width: `${itemW(item) * CELL_SIZE - 2}px`,
    height: `${itemH(item) * CELL_SIZE - 2}px`,
    borderColor: RARITY_COLORS[item.rarity ?? 0] ?? 'var(--el-border-color)'
  }
}

const dropPreviewStyle = computed(() => {
  const item = drag.item
  if (!item) return {}
  return {
    left: `${drag.hoverX * CELL_SIZE}px`,
    top: `${drag.hoverY * CELL_SIZE}px`,
    width: `${itemW(item) * CELL_SIZE - 2}px`,
    height: `${itemH(item) * CELL_SIZE - 2}px`
  }
})

/** Client-side pre-check: inside bounds and not overlapping another item. */
const dropValid = computed(() => {
  const item = drag.item
  if (!item) return false
  const x1 = drag.hoverX
  const y1 = drag.hoverY
  const x2 = x1 + itemW(item) - 1
  const y2 = y1 + itemH(item) - 1
  if (x1 < 0 || y1 < 0 || x2 >= container.gridW || y2 >= container.gridH) return false
  return !container.items.some(other => {
    if (other.itemId === item.itemId) return false
    const ox2 = (other.gridX ?? 0) + itemW(other) - 1
    const oy2 = (other.gridY ?? 0) + itemH(other) - 1
    return x1 <= ox2 && x2 >= (other.gridX ?? 0) && y1 <= oy2 && y2 >= (other.gridY ?? 0)
  })
})

const itemTitle = (item: GmContainerItem) =>
  `${item.name || '#' + item.itemCfgId} x${item.count ?? 0} (${item.gridX}, ${item.gridY})`

const containerLabel = (type: number) => {
  switch (type) {
    case 1:
      return t('admin.gameContainer.backpack')
    case 2:
      return t('admin.gameContainer.stash')
    case 3:
      return t('admin.gameContainer.secure')
    default:
      return '-'
  }
}

const playerIdOrZero = () => {
  const id = Number(query.playerId)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const applyContainers = (list?: GmContainer[]) => {
  const current = (list ?? []).find(c => c.containerType === query.containerType)
  if (!current) return
  container.gridW = current.gridW ?? 0
  container.gridH = current.gridH ?? 0
  container.items = current.items ?? []
  container.loaded = true
}

const loadContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.gameContainer.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmContainer({ playerId, containerType: query.containerType }).catch(() => null)
  loading.value = false
  if (!res) return
  applyContainers(res.data)
}

const sortContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) return
  sorting.value = true
  const res = await sortGmContainer({ playerId, containerType: query.containerType }).catch(() => null)
  sorting.value = false
  if (!res) return
  msgSuccess(t('admin.gameContainer.sortOk'))
  await loadContainer()
}

// ── 拖拽移动 ─────────────────────────────────────────────────────

const onPointerMove = (e: PointerEvent) => {
  if (!drag.active || !drag.item || !gridRef.value) return
  if (!drag.moved) {
    if (Math.abs(e.clientX - drag.startClientX) < 4 && Math.abs(e.clientY - drag.startClientY) < 4) return
    drag.moved = true
  }
  const rect = gridRef.value.getBoundingClientRect()
  const cellX = Math.floor((e.clientX - rect.left) / CELL_SIZE) - drag.grabDX
  const cellY = Math.floor((e.clientY - rect.top) / CELL_SIZE) - drag.grabDY
  drag.hoverX = Math.min(Math.max(0, cellX), container.gridW - itemW(drag.item))
  drag.hoverY = Math.min(Math.max(0, cellY), container.gridH - itemH(drag.item))
}

const onPointerUp = () => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)

  const item = drag.item
  const wasDrag = drag.active && drag.moved
  drag.active = false
  drag.moved = false
  if (!item) return

  if (!wasDrag) {
    openAction(item)
    return
  }
  const toX = drag.hoverX
  const toY = drag.hoverY
  if ((toX === item.gridX && toY === item.gridY) || !dropValid.value) return

  const playerId = playerIdOrZero()
  if (!playerId) return
  moveGmContainerItem({
    playerId,
    containerType: query.containerType,
    itemId: item.itemId ?? '',
    toX,
    toY,
    rotated: item.rotated ?? false
  }).then(() => {
    msgSuccess(t('admin.gameContainer.moveOk'))
    loadContainer()
  }).catch(() => loadContainer())
}

const onItemPointerDown = (e: PointerEvent, item: GmContainerItem) => {
  if (e.button !== 0) return
  const target = e.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  drag.active = true
  drag.moved = false
  drag.item = item
  drag.startClientX = e.clientX
  drag.startClientY = e.clientY
  drag.grabDX = Math.floor((e.clientX - rect.left) / CELL_SIZE)
  drag.grabDY = Math.floor((e.clientY - rect.top) / CELL_SIZE)
  drag.hoverX = item.gridX ?? 0
  drag.hoverY = item.gridY ?? 0
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
})

// ── 点击物品：旋转 / 转移 ────────────────────────────────────────

const openAction = (item: GmContainerItem) => {
  actionDialog.item = item
  actionDialog.name = item.name || `#${item.itemCfgId}`
  actionDialog.visible = true
}

const rotateItem = async() => {
  const item = actionDialog.item
  const playerId = playerIdOrZero()
  if (!item || !playerId) return
  rotating.value = true
  const res = await rotateGmContainerItem({
    playerId,
    containerType: query.containerType,
    itemId: item.itemId ?? ''
  }).catch(() => null)
  rotating.value = false
  if (!res) return
  msgSuccess(t('admin.gameContainer.rotateOk'))
  actionDialog.visible = false
  await loadContainer()
}

const openTransfer = () => {
  const item = actionDialog.item
  if (!item) return
  transferDialog.name = item.name || `#${item.itemCfgId}`
  transferDialog.itemId = item.itemId ?? ''
  transferDialog.toType = query.containerType === 1 ? 2 : 1
  transferDialog.toX = 0
  transferDialog.toY = 0
  transferDialog.rotated = item.rotated ?? false
  actionDialog.visible = false
  transferDialog.visible = true
}

const submitTransfer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) return
  transferDialog.loading = true
  const res = await transferGmContainerItem({
    playerId,
    fromType: query.containerType,
    toType: transferDialog.toType,
    itemId: transferDialog.itemId,
    toX: transferDialog.toX,
    toY: transferDialog.toY,
    rotated: transferDialog.rotated
  }).catch(() => null)
  transferDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameContainer.transferOk'))
  transferDialog.visible = false
  applyContainers(res.data)
}

// ── 添加物品 ─────────────────────────────────────────────────────

/** Bottom catalog row clicked: prefill the add dialog, grid position/count stay editable. */
const addFromCatalog = (entry: GameCatalogEntry) => {
  if (!playerIdOrZero()) {
    msgInfo(t('admin.gameContainer.playerIdInvalid'))
    return
  }
  if (!container.loaded) {
    msgInfo(t('admin.gameContainer.emptyHint'))
    return
  }
  addDialog.name = entry.name ?? ''
  addDialog.itemCfgId = entry.id
  addDialog.count = 1
  addDialog.gridX = 0
  addDialog.gridY = 0
  addDialog.rotated = false
  addDialog.visible = true
}

const submitAdd = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.gameContainer.playerIdInvalid'))
    return
  }
  if (!addDialog.itemCfgId) {
    msgInfo(t('admin.gameContainer.pickItem'))
    return
  }
  addDialog.loading = true
  const res = await addGmContainerItem({
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
  msgSuccess(t('admin.gameContainer.addOk'))
  addDialog.visible = false
  applyContainers(res.data)
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.grid-size,
.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.grid-scroll {
  overflow: auto;
  max-height: 560px;
  padding: 4px;
}

.grid {
  position: relative;
  box-sizing: content-box;
  background-image:
    linear-gradient(to right, var(--el-border-color-lighter) 1px, transparent 1px),
    linear-gradient(to bottom, var(--el-border-color-lighter) 1px, transparent 1px);
  border-right: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.drop-preview {
  position: absolute;
  box-sizing: border-box;
  pointer-events: none;
  background: var(--el-color-success-light-7);
  border: 1px dashed var(--el-color-success);
  border-radius: 4px;
}

.drop-preview.invalid {
  background: var(--el-color-danger-light-7);
  border-color: var(--el-color-danger);
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
  cursor: grab;
  user-select: none;
  touch-action: none;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.grid-item:hover {
  background: var(--el-fill-color-light);
}

.grid-item.dragging {
  z-index: 2;
  opacity: 0.85;
  cursor: grabbing;
}

.grid-item-head {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
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
