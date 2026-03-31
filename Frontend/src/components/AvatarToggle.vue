<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
	<div class="fixed top-1 right-4 w-12 h-12">
    <div class="cursor-pointer rounded-full w-12 h-12"
         @click="$emit('toggleSidebar')"
    >
      <SmallAvatar/>
    </div>
  </div>
</template>

<script>
import SmallAvatar from "@/components/SmallAvatar.vue";

export default {
  name: 'AvatarToggle',
	components: {
		SmallAvatar
	},
};
</script>

<style scoped>
/* 你可以添加其他需要的样式 */
</style>
