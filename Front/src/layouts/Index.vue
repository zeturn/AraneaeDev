<template>
  <div class="w-screen" style="height: 100vh; display: flex;">
    <!-- 侧边栏 -->
    <aside style="width: 200px; overflow-y: auto; border-right: 1px solid #eaeaea;">
      <el-menu router>
        <el-menu-item
            v-for="(menu, index) in menus"
            :key="index"
            :index="String(index)"
            :route="{ path: menu.link }">
          {{ menu.title }}
        </el-menu-item>
      </el-menu>
    </aside>

    <!-- 内容区 -->
    <div style="flex: 1; display: flex; flex-direction: column;">
      <header style="text-align: right; font-size: 12px; padding: 10px 20px;">
        <layout-tabs></layout-tabs>
      </header>

      <p style="color: #999; padding: 0 20px 5px;">缓存组件：{{ caches }}</p>

      <main id="app-main-scroller" style="flex: 1; overflow-y: auto;">
        <div style="padding: 20px;">
          <router-view v-slot="{ Component }">
            <keep-alive :include="caches">
              <component :is="Component"/>
            </keep-alive>
            <!-- 通过isRenderTab刷新tab页方案，Vue3会报错了 -->
            <!-- <component :is="Component" v-if="layoutStore.isRenderTab" /> -->
          </router-view>
        </div>
      </main>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {ref} from 'vue'
import LayoutTabs from './LayoutTabs.vue'
import useRouteCache from '@/hooks/useRouteCache'
import useLayoutStore from '@/store/layout'

const {caches} = useRouteCache()
const layoutStore = useLayoutStore()
const menus = ref([
  {
    link: '/',
    title: '首页'
  },
  {
    link: '/article',
    title: '文章列表'
  },
  {
    link: '/child',
    title: '多级缓存'
  },
  {
    link: '/KeepScroll',
    title: '记录滚动位置'
  }
])
</script>

<style scoped>
:deep(.el-menu-item.is-active) {
  color: #303133;
}

.layout-container-demo .el-header {
  position: relative;
}

.layout-container-demo .el-aside {
  border-right: 1px solid #eee;
}

.layout-container-demo .el-menu {
  border-right: none;
}

.layout-container-demo .el-main {
  padding: 0;
}

.layout-container-demo .toolbar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  right: 20px;
}
</style>