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
        // A production SPA is served from the same public origin as the API.
        // Keep localhost as the development fallback, but never send a
        // browser request for a deployed app to the user's own machine.
        if (typeof window !== 'undefined') {
            const { protocol, hostname, origin } = window.location;
            if (/^https?:$/i.test(protocol)
                && hostname !== 'localhost'
                && hostname !== '127.0.0.1'
                && origin) {
                return origin;
            }
        }
        return 'http://localhost:8107';
    }

    if (typeof window !== 'undefined') {
        const { protocol, hostname, origin } = window.location;
        if (hostname === 'localhost' || hostname === '127.0.0.1') {
            return 'http://localhost:8180';
        }
        if (hostname.startsWith('araneae-front.')) {
            return `${protocol}//${hostname.replace(/^araneae-front\./, 'araneae-control.')}`;
        }
        return origin;
    }

    return 'http://localhost:8180';
}
