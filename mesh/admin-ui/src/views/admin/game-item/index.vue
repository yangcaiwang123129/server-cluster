<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameItem.title') }}</span>
          <span v-if="itemContainer.loaded" class="grid-size">
            {{ containerTypeLabel(query.containerType) }} ·
            {{ $t('admin.gameItem.gridSize') }} {{ itemContainer.gridW }} × {{ itemContainer.gridH }}
          </span>
          <span class="hint">{{ $t('admin.gameItem.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameItem.playerId')">
          <el-input
            v-model="query.playerId"
            :placeholder="$t('admin.gameItem.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.containerType')">
          <el-select v-model="query.containerType" style="width: 140px">
            <el-option :label="$t('admin.gameItem.backpack')" :value="1" />
            <el-option :label="$t('admin.gameItem.stash')" :value="2" />
            <el-option :label="$t('admin.gameItem.secure')" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadItemContainer">
            {{ $t('admin.gameItem.list') }}
          </el-button>
          <el-button
            v-permisaction="['game:gm:item:list']"
            :disabled="!itemContainer.loaded"
            :loading="sorting"
            @click="sortItemContainer"
          >{{ $t('admin.gameItem.sort') }}</el-button>
        </el-form-item>
      </el-form>

      <div v-if="itemContainer.loaded" ref="gridScrollRef" class="grid-scroll">
        <div ref="gridRef" class="grid" :style="gridStyle">
          <div
            v-if="drag.active && drag.item"
            class="drop-preview"
            :class="{ invalid: !dropValid }"
            :style="dropPreviewStyle"
          />
          <div
            v-for="item in itemContainer.items"
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
      <el-empty v-else :description="$t('admin.gameItem.emptyHint')" />
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameItem.itemCatalogTitle') }}</el-divider>

    <GameCatalogList
      type="item"
      permission="game:gm:item:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="actionDialog.visible" :title="actionDialog.name" width="360px">
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('admin.gameItem.itemId')">
          {{ actionDialog.item?.itemId }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameItem.itemCfgId')">
          #{{ actionDialog.item?.itemCfgId }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameItem.count')">
          {{ actionDialog.item?.count }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameItem.position')">
          ({{ actionDialog.item?.gridX }}, {{ actionDialog.item?.gridY }})
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameItem.rotated')">
          {{ actionDialog.item?.rotated ? '✓' : '-' }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button
          v-permisaction="['game:gm:item:list']"
          :loading="rotating"
          @click="rotateItem"
        >{{ $t('admin.gameItem.rotate') }}</el-button>
        <el-button
          v-permisaction="['game:gm:item:list']"
          type="primary"
          @click="openTransfer"
        >{{ $t('admin.gameItem.transfer') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="transferDialog.visible"
      :title="$t('admin.gameItem.transfer')"
      width="420px"
    >
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.gameItem.item')">
          {{ transferDialog.name }}
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.fromType')">
          {{ containerTypeLabel(query.containerType) }}
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.toType')">
          <el-select v-model="transferDialog.toType" style="width: 140px">
            <el-option
              v-for="t in [1, 2, 3].filter(t => t !== query.containerType)"
              :key="t"
              :label="containerTypeLabel(t)"
              :value="t"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.gridX')">
          <el-input-number v-model="transferDialog.toX" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.gridY')">
          <el-input-number v-model="transferDialog.toY" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.rotated')">
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

    <el-dialog v-model="addDialog.visible" :title="$t('admin.gameItem.addTitle')" width="420px">
      <el-form label-width="90px">
        <el-form-item :label="$t('admin.gameItem.pickItem')">
          <span>{{ addDialog.name }} (#{{ addDialog.itemCfgId }})</span>
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.count')">
          <el-input-number v-model="addDialog.count" :min="1" :max="999" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.gridX')">
          <el-input-number v-model="addDialog.gridX" :min="0" :max="itemContainer.gridW - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.gridY')">
          <el-input-number v-model="addDialog.gridY" :min="0" :max="itemContainer.gridH - 1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameItem.rotated')">
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
  addGmItem,
  listGmItem,
  moveGmItem,
  rotateGmItem,
  sortGmItem,
  transferGmItem
} from '@/api/admin/gm'
import type { GameCatalogEntry, GmItem, GmItemEntry } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameItemManage' })

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

const itemContainer = reactive({
  loaded: false,
  gridW: 0,
  gridH: 0,
  items: [] as GmItemEntry[]
})

const gridRef = ref<HTMLElement>()
const gridScrollRef = ref<HTMLElement>()

/** Drag state: follows cells, not pixels — the item snaps to the hovered cell. */
const drag = reactive({
  active: false,
  moved: false,
  item: null as GmItemEntry | null,
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
  item: null as GmItemEntry | null
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
  width: `${itemContainer.gridW * CELL_SIZE}px`,
  height: `${itemContainer.gridH * CELL_SIZE}px`,
  backgroundSize: `${CELL_SIZE}px ${CELL_SIZE}px`
}))

const itemW = (item: GmItemEntry) => item.width ?? 1
const itemH = (item: GmItemEntry) => item.height ?? 1

const itemStyle = (item: GmItemEntry) => {
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

/** 物品 other 是否能整体放到 (x,y)（ignoreIds 中的物品视为不占用）。 */
const canPlaceAt = (other: GmItemEntry, x: number, y: number, ignoreIds: string[]) => {
  const w = itemW(other)
  const h = itemH(other)
  if (x < 0 || y < 0 || x + w > itemContainer.gridW || y + h > itemContainer.gridH) return false
  return !itemContainer.items.some(o => {
    if (ignoreIds.includes(o.itemId ?? '')) return false
    return (
      x < (o.gridX ?? 0) + itemW(o) && x + w > (o.gridX ?? 0) &&
      y < (o.gridY ?? 0) + itemH(o) && y + h > (o.gridY ?? 0)
    )
  })
}

/** 客户端预检：目标区域必须为空，或仅被一件能与之换位的物品占据。 */
const dropValid = computed(() => {
  const item = drag.item
  if (!item) return false
  const x1 = drag.hoverX
  const y1 = drag.hoverY
  const w = itemW(item)
  const h = itemH(item)
  if (x1 < 0 || y1 < 0 || x1 + w > itemContainer.gridW || y1 + h > itemContainer.gridH) return false
  const hits = itemContainer.items.filter(other => {
    if (other.itemId === item.itemId) return false
    return (
      x1 < (other.gridX ?? 0) + itemW(other) && x1 + w > (other.gridX ?? 0) &&
      y1 < (other.gridY ?? 0) + itemH(other) && y1 + h > (other.gridY ?? 0)
    )
  })
  if (hits.length === 0) return true
  if (hits.length > 1) return false
  const other = hits[0]
  const ox = other.gridX ?? 0
  const oy = other.gridY ?? 0
  // 拖拽区域必须整块落在对方物品上，且对方要能整体放回原位置，才允许交换
  const covered = x1 >= ox && x1 + w <= ox + itemW(other) && y1 >= oy && y1 + h <= oy + itemH(other)
  return covered && canPlaceAt(other, item.gridX ?? 0, item.gridY ?? 0, [item.itemId ?? '', other.itemId ?? ''])
})

const itemTitle = (item: GmItemEntry) =>
  `${item.name || '#' + item.itemCfgId} x${item.count ?? 0} (${item.gridX}, ${item.gridY})`

const containerTypeLabel = (type: number) => {
  switch (type) {
    case 1:
      return t('admin.gameItem.backpack')
    case 2:
      return t('admin.gameItem.stash')
    case 3:
      return t('admin.gameItem.secure')
    default:
      return '-'
  }
}

const playerIdOrZero = () => {
  const id = Number(query.playerId)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const applyItemContainers = (list?: GmItem[]) => {
  const current = (list ?? []).find(c => c.containerType === query.containerType)
  if (!current) return
  itemContainer.gridW = current.gridW ?? 0
  itemContainer.gridH = current.gridH ?? 0
  itemContainer.items = current.items ?? []
  itemContainer.loaded = true
}

const loadItemContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) {
    msgInfo(t('admin.gameItem.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmItem({ playerId, containerType: query.containerType }).catch(() => null)
  loading.value = false
  if (!res) return
  applyItemContainers(res.data)
}

const sortItemContainer = async() => {
  const playerId = playerIdOrZero()
  if (!playerId) return
  sorting.value = true
  const res = await sortGmItem({ playerId, containerType: query.containerType }).catch(() => null)
  sorting.value = false
  if (!res) return
  msgSuccess(t('admin.gameItem.sortOk'))
  await loadItemContainer()
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
  drag.hoverX = Math.min(Math.max(0, cellX), itemContainer.gridW - itemW(drag.item))
  drag.hoverY = Math.min(Math.max(0, cellY), itemContainer.gridH - itemH(drag.item))
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
  moveGmItem({
    playerId,
    containerType: query.containerType,
    itemId: item.itemId ?? '',
    toX,
    toY,
    rotated: item.rotated ?? false
  }).then(() => {
    msgSuccess(t('admin.gameItem.moveOk'))
    loadItemContainer()
  }).catch(() => loadItemContainer())
}

const onItemPointerDown = (e: PointerEvent, item: GmItemEntry) => {
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

const openAction = (item: GmItemEntry) => {
  actionDialog.item = item
  actionDialog.name = item.name || `#${item.itemCfgId}`
  actionDialog.visible = true
}

const rotateItem = async() => {
  const item = actionDialog.item
  const playerId = playerIdOrZero()
  if (!item || !playerId) return
  rotating.value = true
  const res = await rotateGmItem({
    playerId,
    containerType: query.containerType,
    itemId: item.itemId ?? ''
  }).catch(() => null)
  rotating.value = false
  if (!res) return
  msgSuccess(t('admin.gameItem.rotateOk'))
  actionDialog.visible = false
  await loadItemContainer()
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
  const res = await transferGmItem({
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
  msgSuccess(t('admin.gameItem.transferOk'))
  transferDialog.visible = false
  applyItemContainers(res.data)
}

// ── 添加物品 ─────────────────────────────────────────────────────

/** Bottom catalog row clicked: prefill the add dialog, grid position/count stay editable. */
const addFromCatalog = (entry: GameCatalogEntry) => {
  if (!playerIdOrZero()) {
    msgInfo(t('admin.gameItem.playerIdInvalid'))
    return
  }
  if (!itemContainer.loaded) {
    msgInfo(t('admin.gameItem.emptyHint'))
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
    msgInfo(t('admin.gameItem.playerIdInvalid'))
    return
  }
  if (!addDialog.itemCfgId) {
    msgInfo(t('admin.gameItem.pickItem'))
    return
  }
  addDialog.loading = true
  const res = await addGmItem({
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
  msgSuccess(t('admin.gameItem.addOk'))
  addDialog.visible = false
  applyItemContainers(res.data)
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
