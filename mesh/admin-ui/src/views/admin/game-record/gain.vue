<template>
  <PageContainer>
    <RecordTable :action="1" :title="$t('admin.gameRecord.gainTitle')" />
  </PageContainer>
</template>

<script setup lang="ts">
import PageContainer from '@/components/PageContainer/index.vue'
import RecordTable from './RecordTable.vue'

// Must match the menu_name the backend serves, or keep-alive silently misses
defineOptions({ name: 'GameRecordGain' })
</script>
