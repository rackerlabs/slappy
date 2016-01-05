import logging

import dns
import dns.opcode
import dns.query
import dns.rdataclass

LOG = logging.getLogger(__name__)


def _prepare_query(zone_name, rdatatype, rdataclass, opcode):
    dns_message = dns.message.make_query(
        qname=zone_name,
        rdtype=rdatatype,
        rdclass=rdataclass,
    )
    dns_message.set_opcode(opcode)
    return dns_message


def _dig(func, *args, **kwargs):
    result = func(*args, **kwargs)
    LOG.info("Digging[func=%s, args=%s, kwargs=%s]\n%s", func.__name__, args,
             kwargs, result)
    return result


def tcp(zone_name, nameserver, rdatatype, rdataclass=dns.rdataclass.IN,
        opcode=dns.opcode.QUERY, port=53):
    query = _prepare_query(zone_name, rdatatype, rdataclass, opcode)
    return _dig(dns.query.tcp, query, nameserver, port=port)


def udp(zone_name, nameserver, rdatatype, rdataclass=dns.rdataclass.IN,
        opcode=dns.opcode.QUERY, port=53):
    query = _prepare_query(zone_name, rdatatype, rdataclass, opcode)
    return _dig(dns.query.udp, query, nameserver, port=port)
