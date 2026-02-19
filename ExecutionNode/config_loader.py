# -*- coding: utf-8 -*-
"""
Load .env (preferred) and optional JSON config (backward compatible).
"""
from __future__ import annotations
from pathlib import Path
import json
import os
from typing import Any, Dict


def _dotenv_paths() -> list[Path]:
    here = Path(__file__).resolve()
    return [
        here.parents[1].parent / ".env",  # repo root
        here.parents[1] / ".env",         # ExecutionNode/.env
    ]


def _load_dotenv_to_env() -> None:
    for p in _dotenv_paths():
        if not p.exists():
            continue
        try:
            for raw in p.read_text(encoding="utf-8").splitlines():
                line = raw.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                key, val = line.split("=", 1)
                key = key.strip()
                val = val.strip().strip('"').strip("'")
                os.environ.setdefault(key, val)
            break
        except Exception:
            continue


def _candidate_paths() -> list[Path]:
    here = Path(__file__).resolve()
    candidates = []
    env_path = os.environ.get("ARANEAE_CONFIG")
    if env_path:
        candidates.append(Path(env_path))
    # repo root (two levels up from this file)
    candidates.append(here.parents[1].parent / "config.json")  # ../../config.json
    # project dir level
    candidates.append(here.parents[1] / "config.json")  # ../config.json
    return candidates


def load_config() -> Dict[str, Any]:
    _load_dotenv_to_env()
    for p in _candidate_paths():
        try:
            if p and p.exists():
                with p.open("r", encoding="utf-8") as f:
                    return json.load(f)
        except Exception:
            return {}
    return {}
