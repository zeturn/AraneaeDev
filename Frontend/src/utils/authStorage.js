const ACCESS_TOKEN_KEY = 'token';
const REFRESH_TOKEN_KEY = 'refresh_token';
const LEGACY_REFRESH_KEY = 'refresh';
const CSRF_TOKEN_KEY = 'csrf_token';

function getStorage() {
    if (typeof window === 'undefined') {
        return null;
    }
    return window.sessionStorage;
}

function migrateLegacyValue(key) {
    if (typeof window === 'undefined') {
        return '';
    }
    const sessionStorage = getStorage();
    const legacy = window.localStorage.getItem(key) || '';
    if (legacy && sessionStorage) {
        sessionStorage.setItem(key, legacy);
        window.localStorage.removeItem(key);
    }
    return legacy;
}

export function getAccessToken() {
    const sessionStorage = getStorage();
    if (sessionStorage) {
        const token = sessionStorage.getItem(ACCESS_TOKEN_KEY) || '';
        if (token) {
            return token;
        }
    }
    return migrateLegacyValue(ACCESS_TOKEN_KEY);
}

export function setAccessToken(token) {
    const sessionStorage = getStorage();
    const clean = typeof token === 'string' ? token : '';
    if (!sessionStorage) {
        return;
    }
    if (!clean) {
        sessionStorage.removeItem(ACCESS_TOKEN_KEY);
        return;
    }
    sessionStorage.setItem(ACCESS_TOKEN_KEY, clean);
}

export function getRefreshToken() {
    const sessionStorage = getStorage();
    if (sessionStorage) {
        const refresh = sessionStorage.getItem(REFRESH_TOKEN_KEY) || '';
        if (refresh) {
            return refresh;
        }
    }
    const migrated = migrateLegacyValue(REFRESH_TOKEN_KEY);
    if (migrated) {
        return migrated;
    }
    const legacyRefresh = migrateLegacyValue(LEGACY_REFRESH_KEY);
    if (legacyRefresh && sessionStorage) {
        sessionStorage.setItem(REFRESH_TOKEN_KEY, legacyRefresh);
        window.localStorage.removeItem(LEGACY_REFRESH_KEY);
    }
    return legacyRefresh;
}

export function setRefreshToken(token) {
    const sessionStorage = getStorage();
    const clean = typeof token === 'string' ? token : '';
    if (!sessionStorage) {
        return;
    }
    if (!clean) {
        sessionStorage.removeItem(REFRESH_TOKEN_KEY);
        return;
    }
    sessionStorage.setItem(REFRESH_TOKEN_KEY, clean);
}

export function getCsrfToken() {
    const sessionStorage = getStorage();
    if (sessionStorage) {
        const csrf = sessionStorage.getItem(CSRF_TOKEN_KEY) || '';
        if (csrf) {
            return csrf;
        }
    }
    return migrateLegacyValue(CSRF_TOKEN_KEY);
}

export function setCsrfTokenValue(token) {
    const sessionStorage = getStorage();
    const clean = typeof token === 'string' ? token : '';
    if (!sessionStorage) {
        return;
    }
    if (!clean) {
        sessionStorage.removeItem(CSRF_TOKEN_KEY);
        return;
    }
    sessionStorage.setItem(CSRF_TOKEN_KEY, clean);
}

export function hasStoredAuth() {
    return !!getAccessToken() || !!getRefreshToken();
}

export function clearStoredAuth() {
    if (typeof window !== 'undefined') {
        window.localStorage.removeItem(ACCESS_TOKEN_KEY);
        window.localStorage.removeItem(REFRESH_TOKEN_KEY);
        window.localStorage.removeItem(LEGACY_REFRESH_KEY);
        window.localStorage.removeItem(CSRF_TOKEN_KEY);
    }
    const sessionStorage = getStorage();
    if (!sessionStorage) {
        return;
    }
    sessionStorage.removeItem(ACCESS_TOKEN_KEY);
    sessionStorage.removeItem(REFRESH_TOKEN_KEY);
    sessionStorage.removeItem(CSRF_TOKEN_KEY);
}
