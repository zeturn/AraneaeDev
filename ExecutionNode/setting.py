# -*- coding: utf-8 -*-

import os
from config_loader import load_config

FLASK_PROJECT_ROOT = os.path.dirname(os.path.abspath(__file__))
LOCAL_REPO = os.path.join(FLASK_PROJECT_ROOT, "repo")

_all_cfg = load_config()
_rabbit = _all_cfg.get('rabbitmq', {})
_security = _all_cfg.get('security', {})
_controlnode = _all_cfg.get('controlnode', {})

RABBITMQ = {
    'HOST': os.environ.get('RABBITMQ_HOST', _rabbit.get('host', 'localhost')),
    'PORT': int(os.environ.get('RABBITMQ_PORT', _rabbit.get('port', 5672))),
    'USERNAME': os.environ.get('RABBITMQ_USERNAME', _rabbit.get('username', 'guest')),
    'PASSWORD': os.environ.get('RABBITMQ_PASSWORD', _rabbit.get('password', '')),
    'VHOST': os.environ.get('RABBITMQ_VHOST', _rabbit.get('vhost', '/')),
}

CONTROLNODE = {
    'BASE_URL': os.environ.get('CONTROLNODE_BASE_URL', _controlnode.get('base_url', 'http://127.0.0.1:8000')).rstrip('/'),
}

SECURITY = {
    'CALLBACK_SHARED_SECRET': os.environ.get('ARANEAE_CALLBACK_SHARED_SECRET', _security.get('callback_shared_secret', '')),
    'NODE_API_TOKEN': os.environ.get('ARANEAE_NODE_API_TOKEN', _security.get('node_api_token', '')),
}
