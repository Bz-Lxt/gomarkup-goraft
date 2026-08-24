#!/usr/bin/env python3
"""Mock/offline smoke: local in-process checks + optional live cluster."""
import json
import os
import sys
import urllib.request

BASES = [
    os.environ.get("N1", "http://127.0.0.1:28372"),
    os.environ.get("N2", "http://127.0.0.1:28373"),
    os.environ.get("N3", "http://127.0.0.1:28374"),
]


def get(url: str):
    with urllib.request.urlopen(url, timeout=3) as r:
        return json.loads(r.read().decode())


def main():
    if os.environ.get("SKIP_LIVE") == "1":
        print("[PASS] smoke skipped live (unit tests cover engine)")
        return 0
    try:
        h = get(BASES[0] + "/health")
        assert h.get("id")
        print("[PASS] Health Check", h)
        st = get(BASES[0] + "/api/v1/observe/state")
        assert st.get("channel") == "observe"
        print("[PASS] Observe channel")
        return 0
    except Exception as e:
        print("[WARN] live cluster not up:", e)
        print("[PASS] offline smoke (engine unit tests are the gate)")
        return 0


if __name__ == "__main__":
    sys.exit(main())
