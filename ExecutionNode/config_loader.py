# -*- coding: utf-8 -*-
"""
Lightweight JSON config loader.
Search order:
1) ARANEAE_CONFIG env var (absolute path)
2) repository root ../.. /config.json relative to this file
3) project root ../ /config.json relative to this file
If none found, returns empty dict.
"""
from __future__ import annotations
from pathlib import Path
import json
import os
from typing import Any, Dict


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
    for p in _candidate_paths():
        try:
            if p and p.exists():
                with p.open("r", encoding="utf-8") as f:
                    return json.load(f)
        except Exception:
            return {}
    return {}
