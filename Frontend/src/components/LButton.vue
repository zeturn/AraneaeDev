<!--
  - Copyright (c)   2024.11  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <button
      :class="buttonClasses"
      :style="buttonStyle"
  >
    {{ text }}
  </button>
</template>

<script>
export default {
  props: {
    color: {
      type: String,
      default: 'primary',
    },
    text: {
      type: String,
      default: 'Button',
    },
    type: {
      type: String,
      default: 'ghost', // 'ghost' or 'solid'
    },
  },
  computed: {
    resolvedType() {
      return this.type === 'outline' ? 'ghost' : this.type;
    },
    resolvedColor() {
      const colorMap = {
        primary: 'var(--accent)',
        green: '#0f766e',
        red: 'var(--danger)',
        danger: 'var(--danger)',
        blue: '#1d4ed8',
        slate: '#475569',
        gray: '#475569',
      };
      return colorMap[this.color] || colorMap.primary;
    },
    buttonClasses() {
      const baseClasses = 'px-4 py-2.5 rounded-xl transition';
      return this.resolvedType === 'solid'
        ? `${baseClasses} btn-solid`
        : `${baseClasses} btn-ghost`;
    },
    buttonStyle() {
      if (this.resolvedType === 'solid') {
        return {'--btn-solid-bg': this.resolvedColor};
      }
      return {color: this.resolvedColor};
    }
  },
};
</script>

<style scoped>
/* Add any custom styles if necessary */
</style>


<!--

Props:

color: 用户可以通过属性定义颜色，如绿色 'green'。
text: 按钮的文字内容。
type: 定义按钮的样式类型（outline 或 solid）。
样式逻辑：

如果 type 为 outline，按钮会显示白底、绿边、绿字，悬停时会变成绿色背景、白字。
如果 type 为 solid，按钮默认是绿色背景、白字，悬停时颜色会变深。
悬停效果：

通过 @mouseenter 和 @mouseleave 事件实现悬停状态的动态样式。

-->