export function getApiFlavor() {
    return (import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase();
}

export function resolveBackendBase() {
    const apiFlavor = getApiFlavor();

    if (typeof window !== 'undefined') {
        const { protocol, hostname, origin } = window.location;
        if (hostname === 'localhost' || hostname === '127.0.0.1') {
            return apiFlavor === 'go' ? 'http://localhost:8180' : 'http://localhost:8107';
        }
        if (hostname.startsWith('araneae-front.')) {
            return `${protocol}//${hostname.replace(/^araneae-front\./, 'araneae-control.')}`;
        }
        // A hosted SPA and its API share the public origin. Prefer it even if
        // a cached/legacy build argument still contains an old domain.
        if (/^https?:$/i.test(protocol) && origin) {
            return origin;
        }
    }

    return apiFlavor === 'go' ? 'http://localhost:8180' : 'http://localhost:8107';
}
