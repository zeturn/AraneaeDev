FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# System deps (Pillow/grpc wheels usually available; keep minimal but safe)
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        gcc \
        libjpeg62-turbo-dev \
        zlib1g-dev \
        libpq-dev \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt \
    && pip install --no-cache-dir psycopg2-binary==2.9.10 \
    && pip install --no-cache-dir gunicorn==22.0.0

COPY . .

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["gunicorn", "Araneae.wsgi:application", "--bind", "0.0.0.0:8000", "--workers", "3", "--timeout", "120"]
