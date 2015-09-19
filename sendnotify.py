import dns.message
import dns.rdatatype
import dns.opcode
import dns.flags
import dns.query

notify = dns.message.make_query('poo.com', dns.rdatatype.SOA)
notify.set_opcode(dns.opcode.NOTIFY)
notify.flags -= dns.flags.RD

try:
    response = dns.query.tcp(notify, '127.0.0.1', port=8053, timeout=5)
except Exception:
    pass
