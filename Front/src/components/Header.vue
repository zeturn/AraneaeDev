<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <header class="text-white py-2 px-4 flex items-center justify-between">
    <div class="flex items-center">
      <button class="mr-4 bg-transparent text-emerald-200" @click="$emit('toggleSidebar')">
        ☰
      </button>
	    <h1 class="text-lg font-semibold text-blue-600">Araneae<h2 class="text-sm font-semibold text-green-400">demo
		    V0.0.1</h2></h1>

    </div>

    <div>
		<AvatarToggle class="w-12 h-12"/>
    </div>
  </header>
</template>

<script lang="ts" setup>
import SmallAvatar from "@/components/SmallAvatar.vue";
import AvatarToggle from "@/components/AvatarToggle.vue";
</script>

<style scoped>
/* Add any additional styles here */
</style>
