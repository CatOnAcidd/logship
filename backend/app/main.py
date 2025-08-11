import asyncio, logging, os
from fastapi import FastAPI, Depends, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy.orm import Session
from sqlalchemy import select, func, desc
from datetime import timedelta
from ipaddress import ip_address
import orjson

from .database import Base, engine, get_db
from .models import Rule, Setting, LogEvent
from .schemas import RuleIn, RuleOut, SettingKV, LogOut, DashboardStats
from .settings import load_settings_from_env, parse_bool
from .rules import matches
from .syslog_listener import UDPServer, TCPServer, parse_syslog
from .forwarder import Forwarder
from .utils import now_utc, iso_period_to_timedelta

logging.basicConfig(level=os.getenv("BACKEND_LOG_LEVEL","INFO"))
log = logging.getLogger("app")

app = FastAPI(title="Logship Syslog Filter API")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"], allow_methods=["*"], allow_headers=["*"]
)

Base.metadata.create_all(bind=engine)

# In-memory forwarding meter (reset by task)
forwarder = Forwarder()
settings_cache = load_settings_from_env()
threshold_window_end = now_utc() + iso_period_to_timedelta(settings_cache.get("THRESHOLD_PERIOD","1d"))

async def handle_syslog(source_ip: str, data: bytes):
    db = next(get_db())
    try:
        raw = data.decode(errors="replace").strip("\r\n")
        parsed = parse_syslog(raw)
        # Record as received
        ev = LogEvent(ts=now_utc(), source_ip=source_ip, raw=raw, size_bytes=len(data), action="received")
        db.add(ev); db.commit(); db.refresh(ev)

        # Rule evaluation
        action = settings_cache.get("DEFAULT_ACTION","BLOCK").upper()
        matched_rule_id = None
        if rules := db.scalars(select(Rule).where(Rule.enabled==True)).all():
            for r in rules:
                if matches(r, source_ip, parsed, raw):
                    matched_rule_id = r.id
                    action = "FORWARD"
                    break
        else:
            # No rules present -> unmatched list
            pass

        # Threshold check
        threshold_enabled = parse_bool(settings_cache.get("THRESHOLD_ENABLED","false"))
        if threshold_enabled and now_utc() <= threshold_window_end:
            if forwarder.total_forwarded_bytes + len(data) >= int(settings_cache.get("THRESHOLD_BYTES","0")):
                # Threshold reached -> drop and tag
                action = "drop"

        if action == "FORWARD":
            await forwarder.forward(settings_cache["DEST_HOST"],
                                    int(settings_cache["DEST_PORT"]),
                                    settings_cache["DEST_PROTOCOL"],
                                    data)
            ev.action = "forward"
        elif matched_rule_id is None and action != "FORWARD":
            ev.action = "unmatched"
        else:
            ev.action = "drop"
        ev.rule_id = matched_rule_id
        db.add(ev); db.commit()
    finally:
        db.close()

@app.on_event("startup")
async def startup():
    # Start UDP
    loop = asyncio.get_running_loop()
    transport, _ = await loop.create_datagram_endpoint(
        lambda: UDPServer(handle_syslog),
        local_addr=("0.0.0.0", int(settings_cache.get("LISTEN_UDP_PORT","514")))
    )
    # Start TCP
    server = await asyncio.start_server(
        TCPServer(handle_syslog).handle,
        host="0.0.0.0",
        port=int(settings_cache.get("LISTEN_TCP_PORT","514"))
    )
    app.state.udp_transport = transport
    app.state.tcp_server = server
    log.info("Syslog listeners started on UDP/TCP %s", settings_cache.get("LISTEN_UDP_PORT","514"))

    async def reset_threshold_task():
        global threshold_window_end
        while True:
            await asyncio.sleep(5)
            if now_utc() > threshold_window_end:
                forwarder.total_forwarded_bytes = 0
                threshold_window_end = now_utc() + iso_period_to_timedelta(settings_cache.get("THRESHOLD_PERIOD","1d"))
    asyncio.create_task(reset_threshold_task())

# ---------------- API ----------------

@app.get("/api/health")
def health():
    return {"status":"ok"}

@app.get("/api/settings", response_model=list[SettingKV])
def get_settings(db: Session = Depends(get_db)):
    rows = db.execute(select(Setting)).scalars().all()
    if not rows:
        return [SettingKV(key=k, value=v) for k,v in settings_cache.items()]
    return [SettingKV(key=r.key, value=r.value) for r in rows]

@app.post("/api/settings", response_model=SettingKV)
def upsert_setting(item: SettingKV, db: Session = Depends(get_db)):
    global settings_cache
    row = db.get(Setting, item.key)
    if row:
        row.value = item.value
    else:
        row = Setting(key=item.key, value=item.value)
        db.add(row)
    db.commit()
    settings_cache[item.key] = item.value
    return item

@app.post("/api/test-destination")
async def test_destination(host: str, port: int, protocol: str = "udp"):
    data = b"<14>Aug 11 00:00:00 logship test: destination test"
    if protocol.lower() == "udp":
        import socket
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.sendto(data, (host, port)); s.close()
    else:
        import asyncio
        r, w = await asyncio.open_connection(host, port)
        w.write(data); await w.drain(); w.close(); await w.wait_closed()
    return {"ok": True}

@app.get("/api/rules", response_model=list[RuleOut])
def list_rules(db: Session = Depends(get_db)):
    return db.execute(select(Rule).order_by(Rule.id)).scalars().all()

@app.post("/api/rules", response_model=RuleOut)
def create_rule(payload: RuleIn, db: Session = Depends(get_db)):
    r = Rule(**payload.model_dump())
    db.add(r); db.commit(); db.refresh(r)
    return r

@app.put("/api/rules/{rule_id}", response_model=RuleOut)
def update_rule(rule_id: int, payload: RuleIn, db: Session = Depends(get_db)):
    r = db.get(Rule, rule_id)
    if not r: raise HTTPException(404, "Rule not found")
    for k, v in payload.model_dump().items():
        setattr(r, k, v)
    db.commit(); db.refresh(r)
    return r

@app.delete("/api/rules/{rule_id}")
def delete_rule(rule_id: int, db: Session = Depends(get_db)):
    r = db.get(Rule, rule_id)
    if not r: raise HTTPException(404, "Rule not found")
    db.delete(r); db.commit()
    return {"ok": True}

@app.get("/api/logs/recent", response_model=dict)
def recent(db: Session = Depends(get_db)):
    def fetch(kind):
        q = select(LogEvent).where(LogEvent.action==kind).order_by(desc(LogEvent.ts)).limit(25)
        return [{
            "id": x.id, "ts": x.ts.isoformat(), "source_ip": x.source_ip,
            "action": x.action, "rule_id": x.rule_id, "raw": x.raw, "size_bytes": x.size_bytes
        } for x in db.execute(q).scalars().all()]
    received = db.execute(select(LogEvent).order_by(desc(LogEvent.ts)).limit(25)).scalars().all()
    return {
        "forwarded": fetch("forward"),
        "dropped": fetch("drop"),
        "unmatched": fetch("unmatched"),
        "received": [{
            "id": x.id, "ts": x.ts.isoformat(), "source_ip": x.source_ip,
            "action": x.action, "rule_id": x.rule_id, "raw": x.raw, "size_bytes": x.size_bytes
        } for x in received]
    }

@app.get("/api/dashboard", response_model=DashboardStats)
def dashboard(db: Session = Depends(get_db)):
    since = now_utc() - timedelta(hours=24)
    def count(kind):
        return db.scalar(select(func.count()).select_from(LogEvent).where(LogEvent.action==kind, LogEvent.ts>=since)) or 0
    # Simple time bucket series (hourly)
    series = []
    for h in range(24):
        t0 = since + timedelta(hours=h)
        t1 = t0 + timedelta(hours=1)
        def cnt(k):
            return db.scalar(select(func.count()).select_from(LogEvent).where(LogEvent.action==k, LogEvent.ts>=t0, LogEvent.ts<t1)) or 0
        series.append([int(t0.timestamp()*1000), cnt("received"), cnt("forward"), cnt("drop"), cnt("unmatched")])
    return {
        "last24h_received": count("received"),
        "last24h_forwarded": count("forward"),
        "last24h_dropped": count("drop"),
        "last24h_unmatched": count("unmatched"),
        "throughput_series": series
    }
