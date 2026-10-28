<template>
  <span class="game-item-icon" :style="style" :title="name">{{ glyph }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

/**
 * Generated tile standing in for a game item icon.
 *
 * The icon fields in the game configs are atlas coordinates ("1,1") or empty
 * strings, and the admin UI ships no art assets, so there is nothing to put in
 * an <img>. This tile renders the name's first character on a rarity-colored
 * chip instead, keeping the GM pages (container/hero/weapon/skill/gear)
 * visually scannable.
 */
defineOptions({ name: 'GameItemIcon' })

const RARITY_COLORS: Record<number, string> = {
  1: '#9ca3af',
  2: '#22c55e',
  3: '#3b82f6',
  4: '#a855f7',
  5: '#f59e0b'
}

const props = withDefaults(defineProps<{
  /** Display name; the first character becomes the tile glyph. */
  name?: string
  /** Item rarity (1-5) picks the color; 0/unknown falls back to tone. */
  rarity?: number
  /** Fallback color for configs without a rarity concept (per-module accent). */
  tone?: string
  /** Tile edge length in px. */
  size?: number
}>(), {
  name: '',
  rarity: 0,
  tone: '#6b7280',
  size: 32
})

const color = computed(() => RARITY_COLORS[props.rarity] ?? props.tone)

const glyph = computed(() => [...(props.name ?? '').trim()][0] ?? '#')

const style = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  fontSize: `${Math.round(props.size * 0.5)}px`,
  color: color.value,
  borderColor: color.value,
  background: `color-mix(in srgb, ${color.value} 16%, transparent)`
}))
</script>

<style scoped>
.game-item-icon {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  font-weight: 600;
  line-height: 1;
  border: 1px solid;
  border-radius: 6px;
  user-select: none;
}
</style>
