import logging

import dns
import dns.query

LOG = logging.getLogger(__name__)


def _prepare_query(zone_name, rdatatype):
    dns_message = dns.message.make_query(zone_name, rdatatype)
    dns_message.set_opcode(dns.opcode.QUERY)
    return dns_message


def _dig(func, *args, **kwargs):
    result = func(*args, **kwargs)
    LOG.info(result)
    return result


def tcp(zone_name, nameserver, rdatatype, port=53):
    query = _prepare_query(zone_name, rdatatype)
    return _dig(dns.query.tcp, query, nameserver, port=port)


def udp(zone_name, nameserver, rdatatype, port=53):
    query = _prepare_query(zone_name, rdatatype)
    return _dig(dns.query.udp, query, nameserver, port=port)
