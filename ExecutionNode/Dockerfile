# Playwright base image includes Chromium + required system deps
# Matches requirements.txt: playwright==1.48.0
FROM mcr.microsoft.com/playwright/python:v1.48.0-jammy

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt \
    && pip install --no-cache-dir psycopg2-binary==2.9.10

COPY . .

EXPOSE 5001

CMD ["python", "app.py"]
