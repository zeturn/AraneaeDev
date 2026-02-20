import hashlib
import secrets
from urllib.parse import urlencode

import jwt
import requests
from django.conf import settings
from django.contrib.auth import get_user_model
from django.http import HttpResponseRedirect, JsonResponse
from rest_framework.permissions import AllowAny
from rest_framework.views import APIView
from rest_framework_simplejwt.tokens import RefreshToken
from drf_spectacular.utils import OpenApiResponse, extend_schema

from Araneae_main.models import OAuthIdentity

User = get_user_model()
SESSION_STATE_KEY = "bp_oauth_state"
SESSION_VERIFIER_KEY = "bp_oauth_verifier"
SESSION_NEXT_KEY = "bp_oauth_next"
PROVIDER = "basaltpass"


def _oauth_config():
    cfg = settings.BASALTPASS_OAUTH
    required = ("DISCOVERY_URL", "CLIENT_ID", "CLIENT_SECRET", "REDIRECT_URI")
    missing = [k for k in required if not cfg.get(k)]
    if missing:
        raise ValueError(f"Missing OAuth settings: {', '.join(missing)}")
    return cfg


def _fetch_discovery(discovery_url: str) -> dict:
    resp = requests.get(discovery_url, timeout=8)
    resp.raise_for_status()
    return resp.json()


def _build_pkce_pair() -> tuple[str, str]:
    # BasaltPass integration docs use hex digest for S256 code challenge.
    verifier = secrets.token_hex(32)
    challenge = hashlib.sha256(verifier.encode("utf-8")).hexdigest()
    return verifier, challenge


def _normalize_next(raw_next: str | None) -> str:
    if not raw_next:
        return "/aprons/workplaces"
    if not raw_next.startswith("/"):
        return "/aprons/workplaces"
    return raw_next


def _frontend_redirect(frontend_base: str, callback_path: str, **params):
    return HttpResponseRedirect(f"{frontend_base}{callback_path}?{urlencode(params)}")


def _safe_username(preferred: str) -> str:
    base = "".join(ch for ch in preferred.lower() if ch.isalnum() or ch in ("_", "-", "."))
    if not base:
        base = "user"
    if len(base) > 100:
        base = base[:100]

    candidate = base
    seq = 1
    while User.objects.filter(username=candidate).exists():
        suffix = f"_{seq}"
        candidate = f"{base[: max(1, 100 - len(suffix))]}{suffix}"
        seq += 1
    return candidate


def _validate_id_token(token: str, discovery: dict, client_id: str) -> dict | None:
    try:
        jwks_uri = discovery.get("jwks_uri")
        issuer = discovery.get("issuer")
        if not jwks_uri or not issuer:
            return None
        jwk_client = jwt.PyJWKClient(jwks_uri)
        signing_key = jwk_client.get_signing_key_from_jwt(token).key
        return jwt.decode(
            token,
            signing_key,
            algorithms=["RS256"],
            audience=client_id,
            issuer=issuer,
        )
    except Exception:
        return None


def _basaltpass_oauth_login(request):
    if not settings.BASALTPASS_OAUTH.get("ENABLED", False):
        return JsonResponse({"error": "BasaltPass OAuth is disabled"}, status=400)

    try:
        cfg = _oauth_config()
        discovery = _fetch_discovery(cfg["DISCOVERY_URL"])
    except Exception as exc:
        return JsonResponse({"error": f"OAuth discovery failed: {exc}"}, status=502)

    state = secrets.token_urlsafe(32)
    verifier, challenge = _build_pkce_pair()
    next_path = _normalize_next(request.GET.get("next"))

    request.session[SESSION_STATE_KEY] = state
    request.session[SESSION_VERIFIER_KEY] = verifier
    request.session[SESSION_NEXT_KEY] = next_path
    request.session.modified = True

    params = {
        "response_type": "code",
        "client_id": cfg["CLIENT_ID"],
        "redirect_uri": cfg["REDIRECT_URI"],
        "scope": cfg.get("SCOPE", "openid profile email offline_access"),
        "state": state,
        "code_challenge": challenge,
        "code_challenge_method": "S256",
    }
    authorize_url = f"{discovery['authorization_endpoint']}?{urlencode(params)}"
    return HttpResponseRedirect(authorize_url)


def _basaltpass_oauth_callback(request):
    error = request.GET.get("error")
    cfg = settings.BASALTPASS_OAUTH
    callback_path = cfg.get("FRONTEND_CALLBACK_PATH", "/oauth/callback")
    frontend_base = settings.FRONTEND_BASE_URL.rstrip("/")
    next_path = _normalize_next(request.session.get(SESSION_NEXT_KEY))

    if error:
        return _frontend_redirect(frontend_base, callback_path, error=error)

    code = request.GET.get("code")
    state = request.GET.get("state")
    expected_state = request.session.get(SESSION_STATE_KEY)
    code_verifier = request.session.get(SESSION_VERIFIER_KEY)

    request.session.pop(SESSION_STATE_KEY, None)
    request.session.pop(SESSION_VERIFIER_KEY, None)
    request.session.pop(SESSION_NEXT_KEY, None)

    if not code or not state or not expected_state or state != expected_state:
        return _frontend_redirect(frontend_base, callback_path, error="invalid_state")

    try:
        oauth_cfg = _oauth_config()
        discovery = _fetch_discovery(oauth_cfg["DISCOVERY_URL"])
        token_resp = requests.post(
            discovery["token_endpoint"],
            data={
                "grant_type": "authorization_code",
                "code": code,
                "redirect_uri": oauth_cfg["REDIRECT_URI"],
                "code_verifier": code_verifier,
            },
            auth=(oauth_cfg["CLIENT_ID"], oauth_cfg["CLIENT_SECRET"]),
            timeout=10,
        )
        token_resp.raise_for_status()
        token_data = token_resp.json()
    except Exception as exc:
        return _frontend_redirect(frontend_base, callback_path, error="token_exchange_failed", detail=str(exc))

    access_token = token_data.get("access_token")
    id_token = token_data.get("id_token")
    claims = _validate_id_token(id_token, discovery, oauth_cfg["CLIENT_ID"]) if id_token else None

    if not access_token:
        return _frontend_redirect(frontend_base, callback_path, error="missing_access_token")

    try:
        userinfo_resp = requests.get(
            discovery["userinfo_endpoint"],
            headers={"Authorization": f"Bearer {access_token}"},
            timeout=8,
        )
        userinfo_resp.raise_for_status()
        userinfo = userinfo_resp.json()
    except Exception as exc:
        return _frontend_redirect(frontend_base, callback_path, error="userinfo_failed", detail=str(exc))

    subject = userinfo.get("sub") or (claims or {}).get("sub")
    email = userinfo.get("email") or (claims or {}).get("email")
    preferred_username = userinfo.get("preferred_username") or userinfo.get("name") or (email.split("@")[0] if email else "")
    first_name = userinfo.get("given_name") or ""
    last_name = userinfo.get("family_name") or ""

    if not subject:
        return _frontend_redirect(frontend_base, callback_path, error="missing_subject")

    identity = OAuthIdentity.objects.filter(provider=PROVIDER, subject=subject).select_related("user").first()
    if identity:
        user = identity.user
        if email and user.email != email:
            user.email = email
            user.save(update_fields=["email"])
    else:
        user = None
        if email:
            user = User.objects.filter(email=email).first()
        if not user:
            username = _safe_username(preferred_username or f"bp_{subject[:12]}")
            user = User.objects.create(username=username, email=email or "", first_name=first_name, last_name=last_name)
            user.set_unusable_password()
            user.save()
        OAuthIdentity.objects.create(provider=PROVIDER, subject=subject, user=user, email=email or "")

    refresh = RefreshToken.for_user(user)
    araneae_access = str(refresh.access_token)
    araneae_refresh = str(refresh)
    return _frontend_redirect(
        frontend_base,
        callback_path,
        access=araneae_access,
        refresh=araneae_refresh,
        next=next_path,
    )


class BasaltPassOAuthLoginView(APIView):
    permission_classes = [AllowAny]

    @extend_schema(
        responses={
            302: OpenApiResponse(description="Redirect to BasaltPass authorize endpoint"),
            400: OpenApiResponse(description="OAuth disabled"),
            502: OpenApiResponse(description="Discovery fetch failed"),
        }
    )
    def get(self, request):
        return _basaltpass_oauth_login(request)


class BasaltPassOAuthCallbackView(APIView):
    permission_classes = [AllowAny]

    @extend_schema(
        responses={
            302: OpenApiResponse(description="Redirect back to frontend callback with OAuth result"),
        }
    )
    def get(self, request):
        return _basaltpass_oauth_callback(request)


basaltpass_oauth_login = BasaltPassOAuthLoginView.as_view()
basaltpass_oauth_callback = BasaltPassOAuthCallbackView.as_view()
