const trimTrailingSlash = value => String(value || '').replace(/\/+$/, '');

export function getApiFlavor() {
    return (import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase();
}

export function resolveBackendBase() {
    const configured = trimTrailingSlash(import.meta.env.VITE_BACKEND_BASE_URL);
    if (configured) {
        return configured;
    }

    const apiFlavor = getApiFlavor();
    if (apiFlavor !== 'go') {
        return 'http://localhost:8107';
    }

    if (typeof window !== 'undefined') {
        const { protocol, hostname } = window.location;
        if (hostname === 'localhost' || hostname === '127.0.0.1') {
            return 'http://localhost:8180';
        }
        if (hostname.startsWith('araneae-front.')) {
            return `${protocol}//${hostname.replace(/^araneae-front\./, 'araneae-control.')}`;
        }
    }

    return 'http://localhost:8180';
}
