import os
from ipaddress import ip_network
from pathlib import Path

DEFAULTS = {
    "DEFAULT_ACTION": os.getenv("DEFAULT_ACTION", "BLOCK"),  # BLOCK or FORWARD
    "DEST_HOST": os.getenv("DEST_HOST", "127.0.0.1"),
    "DEST_PORT": os.getenv("DEST_PORT", "5515"),
    "DEST_PROTOCOL": os.getenv("DEST_PROTOCOL", "udp"),
    "MAX_TEMP_STORAGE_MB": os.getenv("MAX_TEMP_STORAGE_MB", "200"),
    "MAX_DROPPED_STORAGE_MB": os.getenv("MAX_DROPPED_STORAGE_MB", "100"),
    "MAX_THRESHOLD_REACHED_STORAGE_MB": os.getenv("MAX_THRESHOLD_REACHED_STORAGE_MB", "100"),
    "THRESHOLD_ENABLED": os.getenv("THRESHOLD_ENABLED", "false"),
    "THRESHOLD_BYTES": os.getenv("THRESHOLD_BYTES", "0"),
    "THRESHOLD_PERIOD": os.getenv("THRESHOLD_PERIOD", "1d"),
    "LISTEN_UDP_PORT": os.getenv("LISTEN_UDP_PORT", "514"),
    "LISTEN_TCP_PORT": os.getenv("LISTEN_TCP_PORT", "514"),
}

def parse_bool(v: str) -> bool:
    return str(v).lower() in ("1","true","yes","on")

def load_settings_from_env():
    return DEFAULTS.copy()
