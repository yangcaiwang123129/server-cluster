<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <div class="search-bar">
          <el-input
            v-model="keyword"
            :placeholder="t('components.gameCatalog.searchPlaceholder')"
            clearable
            style="width: 240px"
            @keyup.enter="search"
            @clear="search"
          />
          <el-button type="primary" @click="search">{{ t('common.search') }}</el-button>
        </div>
        <span class="hint">{{ t('components.gameCatalog.hint') }}</span>
      </div>
    </template>

    <el-table v-loading="loading" :data="rows" border :empty-text="t('common.empty')">
      <el-table-column prop="name" :label="t('components.gameCatalog.name')" min-width="200">
        <template #default="{ row }">
          <div class="item-cell">
            <GameItemIcon :name="row.name" :rarity="row.rarity" :tone="tone" :size="28" />
            <span class="item-name">{{ row.name || '-' }}</span>
            <span class="item-id">#{{ row.id }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column v-if="showRarity" :label="t('components.gameCatalog.rarity')" width="90">
        <template #default="{ row }">{{ row.rarity || '-' }}</template>
      </el-table-column>
      <el-table-column
        v-if="showDesc"
        :label="t('components.gameCatalog.desc')"
        min-width="220"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ row.desc || '-' }}</template>
      </el-table-column>
      <el-table-column v-if="showSize" :label="t('components.gameCatalog.size')" width="100">
        <template #default="{ row }">{{ (row.width ?? 1) }} × {{ (row.height ?? 1) }}</template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="90" fixed="right">
        <template #default="{ row }">
          <el-button
            v-permisaction="[permission]"
            link
            type="primary"
            @click="emit('add', row)"
          >{{ t('components.gameCatalog.add') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <Pagination
      v-model:page="page"
      v-model:limit="limit"
      :total="total"
      @pagination="load"
    />
  </el-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import GameItemIcon from '@/components/GameItemIcon/index.vue'
import Pagination from '@/components/Pagination/index.vue'
import { listGameCatalog } from '@/api/admin/gm'
import type { GameCatalogEntry } from '@/types/admin'

defineOptions({ name: 'GameCatalogList' })

const props = withDefaults(defineProps<{
  /** Luban table to read: hero/weapon/skill/equip/item. */
  type: 'hero' | 'weapon' | 'skill' | 'equip' | 'item'
  /** Icon accent for tables without a rarity concept. */
  tone?: string
  /** Permission required to add a row to the player. */
  permission: string
}>(), {
  tone: '#6b7280'
})

const emit = defineEmits<{ add: [entry: GameCatalogEntry] }>()

const { t } = useI18n()

const keyword = ref('')
const loading = ref(false)
const rows = ref<GameCatalogEntry[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)

/** Rarity exists for heroes and items; skills/equips carry descriptions; items carry grid size. */
const showRarity = computed(() => props.type === 'hero' || props.type === 'item')
const showDesc = computed(() => props.type === 'skill' || props.type === 'equip')
const showSize = computed(() => props.type === 'item')

const load = async() => {
  loading.value = true
  const res = await listGameCatalog({
    type: props.type,
    keyword: keyword.value || undefined,
    pageIndex: page.value,
    pageSize: limit.value
  }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data.list ?? []
  total.value = res.data.count ?? 0
}

const search = () => {
  page.value = 1
  load()
}

onMounted(load)
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
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
