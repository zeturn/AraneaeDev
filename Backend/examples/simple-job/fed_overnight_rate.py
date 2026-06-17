#!/usr/bin/env python3
"""
示例：采集美联储隔夜利率并通过 Araneae Sink 上报到 HashSlip。

运行环境（在 Araneae Executor 中）会自动注入 `araneae_sink.py`，
脚本只需 import 并 emit_timeseries。
"""

from __future__ import annotations

import json
import os
import urllib.request
from datetime import datetime, timezone

import araneae_sink


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def fetch_fed_overnight_rate() -> float:
    """
    默认请求 FRED SOFR 序列。
    可通过环境变量注入数据，便于本地调试：
      FED_OVERNIGHT_RATE_MOCK=5.33
    """
    mock = os.getenv("FED_OVERNIGHT_RATE_MOCK", "").strip()
    if mock:
        return float(mock)

    # 无 key 的公开接口可能受限；失败时让任务失败，便于重试/告警。
    url = "https://api.stlouisfed.org/fred/series/observations?series_id=SOFR&api_key=demo&file_type=json&sort_order=desc&limit=1"
    with urllib.request.urlopen(url, timeout=15) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
    observations = payload.get("observations") or []
    if not observations:
        raise RuntimeError("no observations returned from FRED")
    value = observations[0].get("value")
    if value in (None, ".", ""):
        raise RuntimeError("invalid SOFR value")
    return float(value)


def main() -> None:
    now = utc_now()
    value = fetch_fed_overnight_rate()

    # 关键：每天固定 hash_key + bucket_date，实现“同一天幂等更新、跨天追加”
    araneae_sink.emit_timeseries(
        source="araneae.fed",
        metric="fed_overnight_rate",
        timestamp=now.isoformat(),
        value=value,
        tags={"market": "US", "series": "SOFR"},
        payload={"collector": "fed_overnight_rate.py"},
        hash_key="fed_overnight_rate_daily",
        bucket_date=now.strftime("%Y-%m-%d"),
    )
    print(f"FED_OVERNIGHT_RATE={value}")


if __name__ == "__main__":
    main()

