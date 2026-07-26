<!--
  - Copyright (c)  2025.5.18
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - App.vue
  - Last Modified: 2025-05-18 23:31:52  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
  <ThemeProvider :default-mode="'system'" storage-key="theme">
    <LocaleProvider>
      <div id="app" class="h-screen w-full overflow-x-hidden">
        <router-view></router-view>
        <NotificationContainer ref="notificationContainer" position="bottom-right" />

      </div>
    </LocaleProvider>
  </ThemeProvider>
</template>

<script lang="ts" setup>
import { ThemeProvider, LocaleProvider } from '@zeturn/watercolor-vue'
import useRouteCache from '@/hooks/useRouteCache'
import NotificationContainer from '@/components/BeansDesign/Notification/NotificationContainer.vue';

// 收集路由配置meta为keepAlive: ture的缓存
const {collectCaches} = useRouteCache()
collectCaches()

</script>
