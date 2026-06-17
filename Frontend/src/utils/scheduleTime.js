const RFC3339_REGEX = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/;
const LOCAL_DATE_TIME_REGEX = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/;

const pad2 = value => String(value).padStart(2, '0');

export const currentTimezoneOffset = () => {
    const minutes = -new Date().getTimezoneOffset();
    const sign = minutes >= 0 ? '+' : '-';
    const abs = Math.abs(minutes);
    const hours = Math.floor(abs / 60);
    const mins = abs % 60;
    return `${sign}${pad2(hours)}:${pad2(mins)}`;
};

export const buildTimezoneOptions = () => {
    const options = [];
    for (let minutes = -12 * 60; minutes <= 14 * 60; minutes += 15) {
        const sign = minutes >= 0 ? '+' : '-';
        const abs = Math.abs(minutes);
        const hours = Math.floor(abs / 60);
        const mins = abs % 60;
        const offset = `${sign}${pad2(hours)}:${pad2(mins)}`;
        options.push({
            value: offset,
            label: `UTC${offset}`,
        });
    }
    return options;
};

export const normalizeLocalDateTime = value => {
    const raw = String(value || '').trim();
    if (!raw) {
        return '';
    }
    if (!LOCAL_DATE_TIME_REGEX.test(raw)) {
        return '';
    }
    return raw.length === 16 ? `${raw}:00` : raw;
};

export const toRunAtRFC3339 = (localDateTime, timezoneOffset) => {
    const normalizedLocal = normalizeLocalDateTime(localDateTime);
    const tz = String(timezoneOffset || '').trim() || '+00:00';
    if (!normalizedLocal) {
        return '';
    }
    if (tz !== 'Z' && !/^[+-]\d{2}:\d{2}$/.test(tz)) {
        return '';
    }
    return `${normalizedLocal}${tz}`;
};

export const fromRunAtRFC3339 = runAt => {
    const raw = String(runAt || '').trim();
    if (!raw) {
        return {localDateTime: '', timezoneOffset: currentTimezoneOffset()};
    }
    const match = raw.match(RFC3339_REGEX);
    if (!match) {
        return {localDateTime: '', timezoneOffset: currentTimezoneOffset()};
    }
    const datePart = match[1];
    const hour = match[2];
    const minute = match[3];
    const second = match[4] || '00';
    const timezoneOffset = match[5] === 'Z' ? '+00:00' : match[5];
    return {
        localDateTime: `${datePart}T${hour}:${minute}:${second}`,
        timezoneOffset,
    };
};
