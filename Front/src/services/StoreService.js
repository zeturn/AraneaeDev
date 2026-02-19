import axios from 'axios';
import { createStore } from 'vuex';

const backendBase = import.meta.env.VITE_BACKEND_BASE_URL || 'http://localhost:8107';
axios.defaults.baseURL = `${backendBase}/`;

export default createStore({
    state: {
        token: localStorage.getItem('token') || '',
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
