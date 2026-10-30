<template>
  <PageContainer>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ $t('admin.gameHero.title') }}</span>
          <span class="hint">{{ $t('admin.gameHero.hint') }}</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent>
        <el-form-item :label="$t('admin.gameHero.playerId')">
          <el-input
            v-model="playerId"
            :placeholder="$t('admin.gameHero.playerIdPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadList">
            {{ $t('admin.gameHero.list') }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="rows" border>
        <el-table-column prop="name" :label="$t('admin.gameHero.name')" min-width="180">
          <template #default="{ row }">
            <div class="item-cell">
              <GameItemIcon :name="row.name" :rarity="row.rarity" :size="28" />
              <span class="item-name">{{ row.name || '-' }}</span>
              <span class="item-id">#{{ row.heroId }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="level" :label="$t('admin.gameHero.level')" min-width="90" />
        <el-table-column prop="exp" :label="$t('admin.gameHero.exp')" min-width="110" />
        <el-table-column prop="star" :label="$t('admin.gameHero.star')" min-width="90" />
        <el-table-column prop="skinId" :label="$t('admin.gameHero.skinId')" min-width="100" />
        <el-table-column :label="$t('admin.gameHero.equipped')" min-width="90">
          <template #default="{ row }">{{ row.equipped ? '✓' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="170" fixed="right">
          <template #default="{ row }">
            <el-button
              v-permisaction="['game:gm:hero:list']"
              link
              type="primary"
              @click="openEdit(row)"
            >{{ $t('common.edit') }}</el-button>
            <el-button
              v-permisaction="['game:gm:hero:list']"
              link
              type="primary"
              @click="openAttribute(row)"
            >{{ $t('admin.gameHero.attribute') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-divider content-position="left">{{ $t('admin.gameHero.catalogTitle') }}</el-divider>

    <GameCatalogList
      type="hero"
      permission="game:gm:hero:list"
      @add="addFromCatalog"
    />

    <el-dialog v-model="editDialog.visible" :title="$t('admin.gameHero.editTitle')" width="420px">
      <el-form label-width="100px">
        <el-form-item :label="$t('admin.gameHero.heroId')">
          <div class="item-cell">
            <GameItemIcon :name="editDialog.name" :rarity="editDialog.rarity" :size="22" />
            <span class="item-name">{{ editDialog.name || '-' }}</span>
            <span class="item-id">#{{ editDialog.heroId }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.gameHero.level')">
          <el-input-number v-model="editDialog.level" :min="1" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameHero.exp')">
          <el-input-number v-model="editDialog.exp" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameHero.star')">
          <el-input-number v-model="editDialog.star" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameHero.skinId')">
          <el-input-number v-model="editDialog.skinId" :min="0" />
        </el-form-item>
        <el-form-item :label="$t('admin.gameHero.equipped')">
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

    <el-dialog v-model="attrDialog.visible" :title="$t('admin.gameHero.attributeTitle')" width="440px">
      <el-descriptions v-loading="attrDialog.loading" :column="2" border>
        <el-descriptions-item :label="$t('admin.gameHero.name')">
          {{ attrDialog.name || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.level')">
          {{ attrDialog.level }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.star')">
          {{ attrDialog.star }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.power')">
          {{ attrDialog.power }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.hp')">
          {{ attrDialog.hp }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.armor')">
          {{ attrDialog.armor }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('admin.gameHero.damage')">
          {{ attrDialog.damage }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="attrDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
        <el-button type="warning" :loading="attrDialog.upgrading" @click="upgradeFromDialog">
          {{ $t('admin.gameHero.upgrade') }}
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

import { addGmHero, getGmHeroAttribute, listGmHero, updateGmHero, upgradeGmHero } from '@/api/admin/gm'
import type { GameCatalogEntry, GmHero } from '@/types/admin'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameHeroManage' })

const { t } = useI18n()

const playerId = ref('')
const loading = ref(false)
const rows = ref<GmHero[]>([])

const editDialog = reactive({
  visible: false,
  loading: false,
  heroId: 0,
  name: '',
  rarity: 0,
  level: 1,
  exp: 0,
  star: 0,
  skinId: 0,
  equipped: false
})

const attrDialog = reactive({
  visible: false,
  loading: false,
  upgrading: false,
  heroId: 0,
  name: '',
  level: 0,
  star: 0,
  power: 0,
  hp: 0,
  armor: 0,
  damage: 0
})

const playerIdOrZero = () => {
  const id = Number(playerId.value)
  return Number.isInteger(id) && id > 0 ? id : 0
}

const loadList = async() => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameHero.playerIdInvalid'))
    return
  }
  loading.value = true
  // The interceptor has already reported any failure; null means we stop here.
  const res = await listGmHero({ playerId: id }).catch(() => null)
  loading.value = false
  if (!res) return
  rows.value = res.data ?? []
}

const addFromCatalog = async(entry: GameCatalogEntry) => {
  const id = playerIdOrZero()
  if (!id) {
    msgInfo(t('admin.gameHero.playerIdInvalid'))
    return
  }
  const res = await addGmHero({ playerId: id, heroId: entry.id! }).catch(() => null)
  if (!res) return
  msgSuccess(t('admin.gameHero.addOk'))
  await loadList()
}

const openEdit = (row: GmHero) => {
  editDialog.heroId = row.heroId ?? 0
  editDialog.name = row.name ?? ''
  editDialog.rarity = row.rarity ?? 0
  editDialog.level = row.level ?? 1
  editDialog.exp = row.exp ?? 0
  editDialog.star = row.star ?? 0
  editDialog.skinId = row.skinId ?? 0
  editDialog.equipped = row.equipped ?? false
  editDialog.visible = true
}

const submitEdit = async() => {
  const id = playerIdOrZero()
  if (!id) return
  editDialog.loading = true
  const res = await updateGmHero({
    playerId: id,
    heroId: editDialog.heroId,
    level: editDialog.level,
    exp: editDialog.exp,
    star: editDialog.star,
    skinId: editDialog.skinId,
    equipped: editDialog.equipped
  }).catch(() => null)
  editDialog.loading = false
  if (!res) return
  msgSuccess(t('admin.gameHero.updateOk'))
  editDialog.visible = false
  await loadList()
}

const loadAttribute = async(heroId: number) => {
  const id = playerIdOrZero()
  if (!id) return
  attrDialog.loading = true
  const res = await getGmHeroAttribute({ playerId: id, heroId }).catch(() => null)
  attrDialog.loading = false
  if (!res) return
  const attr = res.data
  attrDialog.level = attr?.level ?? 0
  attrDialog.star = attr?.star ?? 0
  attrDialog.power = attr?.power ?? 0
  attrDialog.hp = attr?.hp ?? 0
  attrDialog.armor = attr?.armor ?? 0
  attrDialog.damage = attr?.damage ?? 0
}

const openAttribute = async(row: GmHero) => {
  attrDialog.heroId = row.heroId ?? 0
  attrDialog.name = row.name ?? ''
  attrDialog.visible = true
  await loadAttribute(attrDialog.heroId)
}

const upgradeFromDialog = async() => {
  const id = playerIdOrZero()
  if (!id) return
  attrDialog.upgrading = true
  // Upgrade consumes upgrade materials from the backpack; the interceptor reports failures.
  const res = await upgradeGmHero({ playerId: id, cfgId: attrDialog.heroId }).catch(() => null)
  attrDialog.upgrading = false
  if (!res) return
  msgSuccess(t('admin.gameHero.upgradeOk'))
  await loadAttribute(attrDialog.heroId)
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
