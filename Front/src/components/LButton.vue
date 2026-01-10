<!--
  - Copyright (c)   2024.11  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <button
      :class="buttonClasses"
      @mouseenter="isHovered = true"
      @mouseleave="isHovered = false"
  >
    {{ text }}
  </button>
</template>

<script>
export default {
  props: {
    color: {
      type: String,
      default: 'green',
    },
    text: {
      type: String,
      default: 'Button',
    },
    type: {
      type: String,
      default: 'outline', // 'outline' or 'solid'
    },
  },
  data() {
    return {
      isHovered: false,
    };
  },
  computed: {
    buttonClasses() {
      const baseClasses = 'px-4 py-2 rounded-md transition-all duration-300';

      if (this.type === 'outline') {
        return `${baseClasses} border-2 border-${this.color}-500 text-${this.color}-500 bg-white hover:bg-${this.color}-500 hover:text-white`;
      } else if (this.type === 'solid') {
        return `${baseClasses} bg-${this.color}-500 text-white hover:bg-${this.color}-600`;
      }
    },
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