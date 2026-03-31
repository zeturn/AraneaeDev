<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - CheckboxSquareField.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<label :for="resolvedId" :style="labelStyles">
		<input
			:id="resolvedId"
			:name="name"
			:disabled="disabled"
			:checked="currentChecked"
			:style="inputStyles"
			type="checkbox"
			@change="toggleCheck"
		>
		<div :style="checkboxStyles" aria-hidden="true">
			<svg :style="svgStyles" class="size-6" fill="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
				<path clip-rule="evenodd"
					  d="M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353 8.493-12.74a.75.75 0 0 1 1.04-.207Z"
					  fill-rule="evenodd"/>
			</svg>
		</div>
		<span v-if="!hideLabel && (label || $slots.default)" :style="labelTextStyles">
			<slot>{{ label }}</slot>
		</span>
	</label>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'CheckboxSquareField',
	props: {
		modelValue: {
			type: null,
			default: undefined,
		},
		checked: {
			type: null,
			default: undefined,
		},
		label: {
			type: String,
			default: 'Checkbox',
		},
		id: {
			type: String,
			default: '',
		},
		name: {
			type: String,
			default: '',
		},
		disabled: {
			type: Boolean,
			default: false,
		},
		hideLabel: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['update:modelValue', 'change'],
	data() {
		return {
			colors,
			localChecked: false,
			generatedId: `cb-square-${Math.random().toString(36).slice(2, 10)}`,
		};
	},
	computed: {
		hasModelValue() {
			return this.modelValue !== undefined;
		},
		hasCheckedProp() {
			return this.checked !== undefined;
		},
		currentChecked() {
			if (this.hasModelValue) {
				return !!this.modelValue;
			}
			if (this.hasCheckedProp) {
				return !!this.checked;
			}
			return this.localChecked;
		},
		resolvedId() {
			return this.id || this.generatedId;
		},
		labelStyles() {
			return {
				display: 'inline-flex',
				alignItems: 'center',
				cursor: this.disabled ? 'not-allowed' : 'pointer',
				opacity: this.disabled ? '0.65' : '1',
			};
		},
		inputStyles() {
			return {
				position: 'absolute',
				opacity: '0',
				width: '0',
				height: '0',
				pointerEvents: 'none',
			};
		},
		checkboxStyles() {
			return {
				width: '24px',
				height: '24px',
				border: `2px solid ${this.colors.yellowGreen}`,
				backgroundColor: this.currentChecked ? this.colors.yellowGreen : 'transparent',
				borderRadius: '4px',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				transition: 'all 0.2s ease',
				boxSizing: 'border-box',
			};
		},
		svgStyles() {
			return {
				width: '16px',
				height: '16px',
				color: this.colors.white,
				display: this.currentChecked ? 'block' : 'none',
				transform: this.currentChecked ? 'scale(1)' : 'scale(0)',
				transition: 'transform 0.2s ease',
			};
		},
		labelTextStyles() {
			return {
				marginLeft: '8px',
				fontSize: '14px',
				fontWeight: '500',
			};
		},
	},
	methods: {
		toggleCheck(event) {
			if (this.disabled) {
				return;
			}
			const nextValue = !!event.target.checked;
			this.localChecked = nextValue;
			this.$emit('update:modelValue', nextValue);
			this.$emit('change', nextValue);
		},
	},
};
</script>