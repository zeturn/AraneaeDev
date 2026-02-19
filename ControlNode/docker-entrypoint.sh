#!/usr/bin/env sh
set -e

cd /app

# Optional: ensure DB schema exists
if [ "${DJANGO_MIGRATE:-1}" = "1" ]; then
  python manage.py migrate --noinput
fi

# Ensure an initial superuser exists (when running under gunicorn, manage.py __main__ won't run)
if [ "${DJANGO_CREATE_INITIAL_SUPERUSER:-1}" = "1" ]; then
  python - <<'PY'
import os
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'Araneae.settings')
import django
django.setup()

from django.contrib.auth import get_user_model
from Araneae_main.initial.user import create_initial_superuser

User = get_user_model()
if not User.objects.filter(is_superuser=True).exists():
    create_initial_superuser()
PY
fi

# Optional: collect static (only if you configured STATIC_ROOT)
if [ "${DJANGO_COLLECTSTATIC:-0}" = "1" ]; then
  python manage.py collectstatic --noinput
fi

exec "$@"
