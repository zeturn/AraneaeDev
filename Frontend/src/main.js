/*
 * Copyright (c)   2024.11  Henry Zhao. All rights reserved.
 * From CA.
 */

// src/main.js
import { createApp } from 'vue'
import {createPinia} from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './style.css'
import App from './App.vue'
import Router from './router'
import {configKeepScroll} from '@/hooks/useKeepScroll'

const app = createApp(App)
app.use(createPinia())
app.use(ElementPlus)
app.use(Router)
app.mount('#app')

// 配置记录滚动位置的滚动容器
configKeepScroll('#app-main-scroller')

