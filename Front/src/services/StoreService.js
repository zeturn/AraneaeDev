/*
 * Copyright (c)   2024.8  Henry Zhao. All rights reserved.
 */

import axios from 'axios';
import {createStore} from 'vuex';

// 设置 Axios 基本URL
axios.defaults.baseURL = 'http://localhost:8000/';  // 确保后端Django服务器运行在8000端口

export default createStore({
    state: {
        token: localStorage.getItem('token') || '', // 用户Token
        csrfToken: '', // CSRF令牌
        user: {} // 用户信息
    },
    mutations: {},
    actions: {},
    getters: {
        isAuthenticated: state => !!state.token,
        csrfToken: state => state.csrfToken
    }
});
