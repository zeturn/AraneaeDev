import axios from 'axios';
import { createStore } from 'vuex';
import { getAccessToken } from '@/utils/authStorage';
import { resolveBackendBase } from '@/utils/backendBase';

const backendBase = resolveBackendBase();
axios.defaults.baseURL = `${backendBase}/`;

export default createStore({
    state: {
        token: getAccessToken() || '',
        csrfToken: '',
        user: {}
    },
    mutations: {},
    actions: {},
    getters: {
        isAuthenticated: state => !!state.token,
        csrfToken: state => state.csrfToken
    }
});
