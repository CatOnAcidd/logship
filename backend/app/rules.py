import ipaddress, regex, logging
from typing import Optional
from sqlalchemy.orm import Session
from .models import Rule

log = logging.getLogger("rules")

def matches(rule: Rule, source_ip: str, parsed: dict, raw: str) -> bool:
    if not rule.enabled:
        return False
    if rule.source_cidr:
        try:
            net = ipaddress.ip_network(rule.source_cidr, strict=False)
            if ipaddress.ip_address(source_ip) not in net:
                return False
        except Exception:
            return False
    if rule.hostname and parsed.get("hostname") != rule.hostname:
        return False
    if rule.app_name and parsed.get("app_name") != rule.app_name:
        return False
    if rule.facility and parsed.get("facility") != rule.facility:
        return False
    if rule.severity and parsed.get("severity") != rule.severity:
        return False
    if rule.message_regex:
        try:
            if not regex.search(rule.message_regex, raw):
                return False
        except Exception as e:
            log.warning("Invalid regex in rule %s: %s", rule.id, e)
            return False
    return True
