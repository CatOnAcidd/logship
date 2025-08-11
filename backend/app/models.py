from sqlalchemy import Column, Integer, String, DateTime, Text, Enum, Boolean, ForeignKey, BigInteger
from sqlalchemy.orm import relationship
from datetime import datetime, timezone
from .database import Base

class Rule(Base):
    __tablename__ = "rules"
    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, nullable=False)
    enabled = Column(Boolean, default=True)
    source_cidr = Column(String, nullable=True)      # e.g., "10.0.0.0/24" or single IP
    hostname = Column(String, nullable=True)
    app_name = Column(String, nullable=True)
    facility = Column(String, nullable=True)
    severity = Column(String, nullable=True)
    message_regex = Column(String, nullable=True)
    created_at = Column(DateTime, default=lambda: datetime.now(timezone.utc))

class Setting(Base):
    __tablename__ = "settings"
    key = Column(String, primary_key=True)
    value = Column(Text, nullable=False)

class LogEvent(Base):
    __tablename__ = "logs"
    id = Column(Integer, primary_key=True, index=True)
    ts = Column(DateTime, default=lambda: datetime.now(timezone.utc), index=True)
    source_ip = Column(String, index=True)
    raw = Column(Text)
    size_bytes = Column(BigInteger, default=0)
    action = Column(String, index=True)  # 'forward','drop','unmatched','received'
    rule_id = Column(Integer, ForeignKey("rules.id"), nullable=True)
    rule = relationship("Rule")
