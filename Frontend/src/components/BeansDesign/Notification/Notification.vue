<!--
  - Copyright (c)  2025.5.18
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Notification.vue
  - Last Modified: 2025-05-18 23:33:07  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<div
		:class="classes"
		:style="{ top: `${offset}px` }"
		class="flex items-start gap-3 px-4 py-3 rounded-xl shadow-lg transition-all duration-500 relative min-w-[280px] max-w-xs"
		@mouseenter="pauseTimer"
		@mouseleave="resumeTimer"
	>
		<span class="mt-1 text-xl">{{ icon }}</span>
		<div class="flex-1">
			<div class="font-bold text-base">{{ title }}</div>
			<div class="text-sm mt-1">{{ message }}</div>
		</div>
		<button aria-label="关闭" class="btn-muted ml-2 px-2 py-1 text-lg" @click="close">&times;</button>
	</div>
</template>

<script setup>
import {ref, onMounted, onUnmounted} from 'vue';

const props = defineProps({
	id: Number,
	type: String,
	title: String,
	message: String,
	duration: {type: Number, default: 5000},
	offset: Number,
	onClose: Function,
});

const timer = ref(null);

const icons = {
	success: "✅",
	info: "ℹ️",
	error: "❌",
	warning: "⚠️",
	special: "💜"
};
const icon = icons[props.type] || "ℹ️";

const classes = {
	'success': 'border-1 border-green-400 bg-green-50 text-green-800',
	'info': 'border-1 border-blue-400 bg-blue-50 text-blue-800',
	'error': 'border-1 border-red-400 bg-red-50 text-red-800',
	'warning': 'border-1 border-yellow-400 bg-yellow-50 text-yellow-800',
	'special': 'border-1 border-purple-400 bg-purple-50 text-purple-800'
}[props.type] || 'border-1 border-blue-400 bg-blue-50 text-blue-800';

function close() {
	props.onClose && props.onClose(props.id);
}

function startTimer() {
	timer.value = setTimeout(close, props.duration);
}

function pauseTimer() {
	clearTimeout(timer.value);
}

function resumeTimer() {
	startTimer();
}

onMounted(startTimer);
onUnmounted(() => clearTimeout(timer.value));
</script>
