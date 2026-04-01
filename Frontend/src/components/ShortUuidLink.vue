<template>
	<RouterLink
		v-if="isLink"
		:to="to"
		:class="baseClass"
		:title="titleText"
	>
		{{ displayText }}
	</RouterLink>
	<span v-else :class="baseClass" :title="titleText">
		{{ displayText }}
	</span>
</template>

<script setup>
import {computed} from 'vue';

const props = defineProps({
	value: {
		type: [String, Number],
		default: '-',
	},
	to: {
		type: [String, Object],
		default: '',
	},
	className: {
		type: String,
		default: '',
	},
});

const normalizedValue = computed(() => {
	if (props.value === null || props.value === undefined) {
		return '-';
	}
	const text = String(props.value).trim();
	return text || '-';
});

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const compactUuidPattern = /^[0-9a-f]{32}$/i;

const isUuid = computed(() => {
	const text = normalizedValue.value;
	return uuidPattern.test(text) || compactUuidPattern.test(text);
});

const displayText = computed(() => {
	if (normalizedValue.value === '-') {
		return '-';
	}
	return isUuid.value ? normalizedValue.value.slice(0, 8) : normalizedValue.value;
});

const isLink = computed(() => {
	if (normalizedValue.value === '-') {
		return false;
	}
	if (!props.to) {
		return false;
	}
	if (typeof props.to === 'string') {
		return props.to.trim().length > 0;
	}
	return true;
});

const titleText = computed(() => {
	return isUuid.value ? normalizedValue.value : '';
});

const baseClass = computed(() => {
	const classList = ['font-mono text-sm'];
	if (isLink.value) {
		classList.push('text-teal-700', 'hover:text-teal-900', 'hover:underline');
	} else {
		classList.push('text-gray-700');
	}
	if (props.className) {
		classList.push(props.className);
	}
	return classList.join(' ');
});
</script>
