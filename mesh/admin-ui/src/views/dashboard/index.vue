<template>
  <div class="dashboard-container">
    <el-card shadow="never">
      <h2 class="dashboard-title">{{ $t('route.dashboard') }}</h2>
      <p class="dashboard-user">
        <span>{{ $t('profile.username') }}</span>
        {{ user.name }}
      </p>
    </el-card>
  </div>
</template>

<script setup>
import { useUserStore } from '@/stores/user'

const user = useUserStore()
</script>

<style lang="scss" scoped>
.dashboard-container {
  padding: 16px;
}

.dashboard-title {
  margin: 0 0 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--ga-text-1);
}

.dashboard-user {
  margin: 0;
  font-size: 14px;
  color: var(--ga-text-2);
}
</style>
