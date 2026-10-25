<template>
  <img v-if="avatar" :src="avatar" :title="$t('profile.avatar.hint')" class="img-circle img-lg">
  <el-avatar v-else :size="100" class="img-circle">
    <el-icon :size="48"><user-filled /></el-icon>
  </el-avatar>
</template>

<script>
import { useUserStore } from '@/stores/user'

export default {
  props: {
    // eslint-disable-next-line vue/require-default-prop
    user: { type: Object }
  },
  computed: {
    avatar() {
      return this.user.avatar || useUserStore().avatar
    }
  }
}
</script>
