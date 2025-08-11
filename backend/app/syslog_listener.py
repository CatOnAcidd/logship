import asyncio, logging, re
from datetime import datetime, timezone

log = logging.getLogger("syslog")

PRI_RE = re.compile(r"^<(\d{1,3})>")
HEADER_RE = re.compile(r"^(?:<\d{1,3}>)(\w{3}\s+\d{1,2}\s\d{2}:\d{2}:\d{2})\s(\S+)\s(\S+):?\s?(.*)")
# Very light RFC3164; for 5424 you'd extend this parser

def parse_syslog(raw: str):
    m = HEADER_RE.search(raw)
    facility = severity = None
    try:
        pri = int(PRI_RE.match(raw).group(1))
        facility = str(pri // 8)
        severity = str(pri % 8)
    except Exception:
        pass
    if m:
        ts_str, hostname, app, msg = m.groups()
        # We won't fully parse date (no year); store as now for simplicity
        return {
            "hostname": hostname,
            "app_name": app.strip(":"),
            "message": msg,
            "facility": facility,
            "severity": severity,
        }
    return {
        "hostname": None,
        "app_name": None,
        "message": raw,
        "facility": facility,
        "severity": severity,
    }

class UDPServer(asyncio.DatagramProtocol):
    def __init__(self, handler):
        self.handler = handler

    def datagram_received(self, data, addr):
        ip, _ = addr
        asyncio.create_task(self.handler(ip, data))

class TCPServer:
    def __init__(self, handler):
        self.handler = handler

    async def handle(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
        addr = writer.get_extra_info("peername")
        ip = addr[0] if addr else "unknown"
        try:
            data = await reader.read(64 * 1024)
            if data:
                await self.handler(ip, data)
        finally:
            writer.close()
            await writer.wait_closed()
