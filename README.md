# Araneae

Distributed task scheduling and execution platform.

## Port Convention (Project 07)
- ControlNode API: `8107`
- Frontend: `5109`
- ExecutionNode (host mapping): `4107`
- RabbitMQ AMQP: `5672`
- RabbitMQ Management: `15672`

## Configuration Source
Araneae now uses repository-root `.env` as the primary config source.

- Copy `./.env.example` to `./.env`
- Fill real secrets and URLs
- Do not commit `.env`

Backward compatibility with `config.json` is still kept, but `.env` is preferred.

## Quick Start (Docker)
```powershell
cd Araneae
docker compose up --build
```

Access:
- Front: `http://localhost:5109`
- ControlNode API: `http://localhost:8107`
- ExecutionNode: `http://localhost:4107`
- RabbitMQ Console: `http://localhost:15672`

## Quick Start (Local)
1. ControlNode
```powershell
cd ControlNode
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
python manage.py migrate
python manage.py runserver 0.0.0.0:8107
```

2. ExecutionNode
```powershell
cd ..\ExecutionNode
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
python app.py
```

3. Frontend
```powershell
cd ..\Front
npm install
npm run dev -- --port 5109
```

## BasaltPass OAuth Notes
Ensure these `.env` values are configured:
- `BASALTPASS_OAUTH_CLIENT_ID`
- `BASALTPASS_OAUTH_CLIENT_SECRET`
- `BASALTPASS_OAUTH_DISCOVERY_URL`
- `BASALTPASS_OAUTH_REDIRECT_URI=http://localhost:8107/api/auth/basaltpass/callback/`

## Security Notes
- Set `ARANEAE_CALLBACK_SHARED_SECRET` in both ControlNode and ExecutionNode
- Set `ARANEAE_NODE_API_TOKEN` to protect ExecutionNode control endpoints

## Release Process
- Release gates and branch protection baseline: `docs/RELEASE_GATES.md`
