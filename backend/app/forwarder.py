import asyncio, logging, socket
from typing import Tuple

log = logging.getLogger("forwarder")

class Forwarder:
    def __init__(self):
        self.total_forwarded_bytes = 0

    async def send_udp(self, host: str, port: int, data: bytes):
        loop = asyncio.get_running_loop()
        def _send():
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            try:
                sock.sendto(data, (host, port))
            finally:
                sock.close()
        await loop.run_in_executor(None, _send)

    async def send_tcp(self, host: str, port: int, data: bytes):
        reader, writer = await asyncio.open_connection(host, port)
        writer.write(data)
        await writer.drain()
        writer.close()
        await writer.wait_closed()

    async def forward(self, host: str, port: int, proto: str, data: bytes):
        if proto.lower() == "udp":
            await self.send_udp(host, port, data)
        else:
            await self.send_tcp(host, port, data)
        self.total_forwarded_bytes += len(data)
