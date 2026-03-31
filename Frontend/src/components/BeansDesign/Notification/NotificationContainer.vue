<!--
  - Copyright (c)  2025.5.18
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - NotificationContainer.vue
  - Last Modified: 2025-05-18 23:36:05  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<div
		:class="containerClass"
		class="fixed z-[9999] flex flex-col gap-3 pointer-events-none"
	>
		<Notification
			v-for="(notif, idx) in notifications"
			:key="notif.id"
			:offset="getOffset(idx)"
			:onClose="removeNotification"
			class="pointer-events-auto"
			v-bind="notif"
		/>
	</div>
</template>

<script lang="ts" setup>
import {ref, onMounted, onUnmounted, computed} from 'vue';
import Notification from './Notification.vue';
import EventBus, {type NotifyEvent} from '@/utils/event-bus';

const props = defineProps({
	// 可选值: top-right, top-left, top-center, bottom-right, bottom-left, bottom-center
	position: {
		type: String,
		default: 'top-right'
	}
});

type NotificationItem = NotifyEvent & { id: number };
const notifications = ref<NotificationItem[]>([]);
let id = 0;

// 监听通知事件
function addNotification(payload: NotifyEvent) {
	notifications.value.push({
		id: ++id,
		...payload,
		duration: payload.duration ?? 5000,
	});
}

function removeNotification(rid: number) {
	notifications.value = notifications.value.filter(notif => notif.id !== rid);
}

// 位置容器class
const containerClass = computed(() => {
	switch (props.position) {
		case 'top-left':
			return 'top-5 left-5 items-start';
		case 'top-center':
			return 'top-5 left-1/2 -translate-x-1/2 items-center';
		case 'bottom-right':
			return 'bottom-5 right-5 items-end';
		case 'bottom-left':
			return 'bottom-5 left-5 items-start';
		case 'bottom-center':
			return 'bottom-5 left-1/2 -translate-x-1/2 items-center';
		default:
			return 'top-5 right-5 items-end'; // top-right
	}
});

// 垂直堆叠间隔
function getOffset(idx: number) {
	return idx * 88;
}

onMounted(() => {
	EventBus.on('notify', addNotification);
});
onUnmounted(() => {
	EventBus.off('notify', addNotification);
});

// 如果你有外部需求，也可以暴露方法
defineExpose({addNotification});
</script>
