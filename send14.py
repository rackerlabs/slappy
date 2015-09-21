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
    cre = dns.message.make_query('example1.com', CREATE, rdclass=ClassCC)
    cre.set_opcode(14)
    cre.flags -= dns.flags.RD

    response = dns.query.udp(cre, '127.0.0.1', port=5358, timeout=5)
except Exception:
    pass

# try:
#     dell = dns.message.make_query('example1.com', DELETE, rdclass=ClassCC)
#     dell.set_opcode(14)
#     dell.flags -= dns.flags.RD

#     response = dns.query.udp(dell, '127.0.0.1', port=5358, timeout=5)
# except Exception:
#     pass

# import dns.resolver

# resolver = dns.resolver.Resolver()
# resolver.nameservers = ['127.0.0.1:5358']
# answers = dns.resolver.query('example1.com', 'A')
# for rdata in answers:
#     print rdata