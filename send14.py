import dns.message
import dns.rdatatype
import dns.opcode
import dns.flags
import dns.query

# Command and Control OPCODE
CC = 14

# Private DNS CLASS Uses
ClassCC = 65280

# Private RR Code Uses
SUCCESS = 65280
FAILURE = 65281
CREATE = 65282
DELETE = 65283

try:
    cre = dns.message.make_query('poo.com', CREATE, rdclass=ClassCC)
    cre.set_opcode(14)
    cre.flags -= dns.flags.RD

    response = dns.query.udp(cre, '127.0.0.1', port=8053, timeout=5)
except Exception:
    pass

try:
    dell = dns.message.make_query('poo.com', DELETE, rdclass=ClassCC)
    dell.set_opcode(14)
    dell.flags -= dns.flags.RD

    response = dns.query.udp(dell, '127.0.0.1', port=8053, timeout=5)
except Exception:
    pass
