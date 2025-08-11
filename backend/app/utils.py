from datetime import datetime, timedelta, timezone
import re

def now_utc():
    return datetime.now(timezone.utc)

def iso_period_to_timedelta(s: str):
    # Supports simple shorthands and ISO-ish strings like "1d", "7d", "30d" or "PT12H"
    s = s.lower().strip()
    if s.endswith("d") and s[:-1].isdigit():
        return timedelta(days=int(s[:-1]))
    if s.endswith("h") and s[:-1].isdigit():
        return timedelta(hours=int(s[:-1]))
    # naive ISO8601 partial support: PnD and PTnH
    m = re.match(r"p(\d+)d", s)
    if m:
        return timedelta(days=int(m.group(1)))
    m = re.match(r"pt(\d+)h", s)
    if m:
        return timedelta(hours=int(m.group(1)))
    return timedelta(days=1)  # default 1 day
