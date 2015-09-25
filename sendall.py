import dns.message
import dns.rdatatype
import dns.opcode
import dns.flags
import dns.query
import sys


# Command and Control OPCODE
CC = 14

# Private DNS CLASS Uses
ClassCC = 65280

# Private RR Code Uses
SUCCESS = 65280
FAILURE = 65281
CREATE = 65282
DELETE = 65283

zone = sys.argv[1]
cre = dns.message.make_query(zone, CREATE, rdclass=ClassCC)
cre.set_opcode(14)
cre.flags -= dns.flags.RD

response = dns.query.tcp(cre, '127.0.0.1', port=5358, timeout=5)

# notify = dns.message.make_query(zone, dns.rdatatype.SOA)
# notify.set_opcode(dns.opcode.NOTIFY)
# notify.flags -= dns.flags.RD

# response = dns.query.tcp(notify, '127.0.0.1', port=5358, timeout=5)


dell = dns.message.make_query(zone, DELETE, rdclass=ClassCC)
dell.set_opcode(14)
dell.flags -= dns.flags.RD

response = dns.query.tcp(dell, '127.0.0.1', port=5358, timeout=5)

zone = sys.argv[1]
cre = dns.message.make_query(zone, CREATE, rdclass=ClassCC)
cre.set_opcode(14)
cre.flags -= dns.flags.RD

response = dns.query.tcp(cre, '127.0.0.1', port=5358, timeout=5)
