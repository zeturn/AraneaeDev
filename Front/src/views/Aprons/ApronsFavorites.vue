<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsFavorites.vue
  - Last Modified: 2025-03-07 18:25:27  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Aprons>
		<div class="flex flex-col p-2">
			<h1 class="text-3xl font-bold text-gray-700">组件展示</h1>

			<!-- 按钮组件 -->
			<div>
				<h2 class="text-2xl font-semibold mb-2">按钮组件</h2>
				<p class="text-gray-500 text-sm mb-4">展示各种按钮样式</p>
				<div class="grid grid-cols-2 md:grid-cols-4 gap-2">
					<ButtonLargeSolid/>
					<ButtonLargeOutline/>
					<ButtonMediumSolid/>
					<ButtonMediumOutline/>
					<ButtonSmallSolid/>
					<ButtonSmallOutline/>
					<ButtonText/>
					<Link/>
				</div>
			</div>

			<!-- 输入组件 -->
			<div>
				<h2 class="text-2xl font-semibold mb-2">输入组件</h2>
				<p class="text-gray-500 text-sm mb-4">展示输入框、文本区、文件上传等组件</p>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<InputField/>
					<TextareaField/>
					<FileInputField acceptTypes=".zip"/>
				</div>
			</div>

			<!-- 开关和复选框 -->
			<div>
				<h2 class="text-2xl font-semibold mb-2">开关和复选框</h2>
				<p class="text-gray-500 text-sm mb-4">展示开关按钮和不同样式的复选框</p>
				<div class="grid grid-cols-3 gap-6">
					<ToggleSwitch/>
					<CheckboxRoundField/>
					<CheckboxSquareField/>
				</div>
			</div>

			<!-- 标签组件 -->
			<div>
				<h2 class="text-2xl font-semibold mb-2">标签组件</h2>
				<p class="text-gray-500 text-sm mb-4">展示标签样式</p>
				<div class="flex space-x-4">
					<BadgeSolid>大标签</BadgeSolid>
					<BadgeOutline>大标签</BadgeOutline>
				</div>
			</div>

			<!-- 折叠面板 -->
			<div>
				<h2 class="text-2xl font-semibold mb-2">折叠面板</h2>
				<p class="text-gray-500 text-sm mb-4">展示折叠面板组件</p>
				<CollapsePanel title="折叠面板标题">
					<p class="text-gray-600">这是折叠面板的内容。点击标题可以展开或收起内容。</p>
				</CollapsePanel>
			</div>

			<div>
				<div>
					<h2 class="text-2xl font-semibold mb-2">基础卡片</h2>
					<p class="text-gray-500 text-sm mb-4">使用于显示各类内容</p>
					<BasicCard description="这是一个简单的描述" title="示例标题">
						<button class="bg-green-500 text-white px-4 py-2 rounded-lg">点击</button>
					</BasicCard>
				</div>
				<div class="mt-4">
					<BasicCard
						borderColor="red"
						description="这是一个示例描述"
						title="示例标题"
						titleColor="red"
					>
						<button class="bg-red-500 text-white px-4 py-2 rounded-lg">点击</button>
					</BasicCard>
				</div>
			</div>
		</div>
	</Aprons>
</template>

<script>
import Aprons from "@/views/Aprons/Aprons.vue";
import ButtonLargeSolid from '@/components/BeansDesign/Button/ButtonLargeSolid.vue';
import ButtonLargeOutline from '@/components/BeansDesign/Button/ButtonLargeOutline.vue';
import ButtonMediumSolid from '@/components/BeansDesign/Button/ButtonMediumSolid.vue';
import ButtonMediumOutline from '@/components/BeansDesign/Button/ButtonMediumOutline.vue';
import ButtonSmallSolid from '@/components/BeansDesign/Button/ButtonSmallSolid.vue';
import ButtonSmallOutline from '@/components/BeansDesign/Button/ButtonSmallOutline.vue';
import ButtonText from '@/components/BeansDesign/Button/ButtonText.vue';
import InputField from '@/components/BeansDesign/Input/InputField.vue';
import TextareaField from '@/components/BeansDesign/Textarea/TextareaField.vue';
import FileInputField from '@/components/BeansDesign/Fileinput/FileInputField.vue';
import ToggleSwitch from '@/components/BeansDesign/Toggle/ToggleSwitch.vue';
import CheckboxRoundField from '@/components/BeansDesign/Checkbox/CheckboxRoundField.vue';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
import BadgeSolid from '@/components/BeansDesign/Badge/BadgeSolid.vue';
import BadgeOutline from '@/components/BeansDesign/Badge/BadgeOutline.vue';
import CollapsePanel from '@/components/BeansDesign/Collapse/CollapsePanel.vue';
import Link from '@/components/BeansDesign/Link/Link.vue';
import BasicCard from '@/components/BeansDesign/Card/BasicCard.vue';

export default {
	components: {
		Aprons,
		ButtonLargeSolid,
		ButtonLargeOutline,
		ButtonMediumSolid,
		ButtonMediumOutline,
		ButtonSmallSolid,
		ButtonSmallOutline,
		ButtonText,
		InputField,
		TextareaField,
		FileInputField,
		ToggleSwitch,
		CheckboxRoundField,
		CheckboxSquareField,
		BadgeSolid,
		BadgeOutline,
		CollapsePanel,
		Link,
		BasicCard,
	},
};
</script>

<style scoped>
.grid {
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}
</style>