/*
 * Copyright (c)   2024.11  Henry Zhao. All rights reserved.
 * From CA.
 */

// src/main.js
import { createApp } from 'vue'
import {createPinia} from 'pinia'
import { ElIcon, ElMenu, ElMenuItem, ElOption, ElSelect, ElTabPane, ElTabs } from 'element-plus'
import 'element-plus/es/components/icon/style/css'
import 'element-plus/es/components/menu/style/css'
import 'element-plus/es/components/menu-item/style/css'
import 'element-plus/es/components/option/style/css'
import 'element-plus/es/components/select/style/css'
import 'element-plus/es/components/tab-pane/style/css'
import 'element-plus/es/components/tabs/style/css'
import './style.css'
import App from './App.vue'
import Router from './router'
import {configKeepScroll} from '@/hooks/useKeepScroll'

const app = createApp(App)
app.use(createPinia())
app.use(ElIcon)
app.use(ElMenu)
app.use(ElMenuItem)
app.use(ElOption)
app.use(ElSelect)
app.use(ElTabPane)
app.use(ElTabs)
app.use(Router)
app.mount('#app')

// 配置记录滚动位置的滚动容器
configKeepScroll('#app-main-scroller')

