from pydantic import BaseModel
from typing import Optional, List
from datetime import datetime

class RuleIn(BaseModel):
    name: str
    enabled: bool = True
    source_cidr: Optional[str] = None
    hostname: Optional[str] = None
    app_name: Optional[str] = None
    facility: Optional[str] = None
    severity: Optional[str] = None
    message_regex: Optional[str] = None

class RuleOut(RuleIn):
    id: int

class SettingKV(BaseModel):
    key: str
    value: str

class LogOut(BaseModel):
    id: int
    ts: datetime
    source_ip: str
    action: str
    rule_id: Optional[int] = None
    raw: str
    size_bytes: int

class DashboardStats(BaseModel):
    last24h_received: int
    last24h_forwarded: int
    last24h_dropped: int
    last24h_unmatched: int
    throughput_series: list  # [[epoch_ms, received, forwarded, dropped, unmatched], ...]
