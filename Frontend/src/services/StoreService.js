import axios from 'axios';
import { createStore } from 'vuex';
import { getAccessToken } from '@/utils/authStorage';

const apiFlavor = (import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase();
const backendBase = import.meta.env.VITE_BACKEND_BASE_URL || (apiFlavor === 'go' ? 'http://localhost:8180' : 'http://localhost:8107');
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
