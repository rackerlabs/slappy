import dns.rdatatype
import dns.opcode

from functionaltests.common.config import cfg


class SlappyClient(object):

    OPCODE = 14
    RDCLASS = 65280
    CREATE_RDTYPE = 65282
    DELETE_RDTYPE = 65283

    def __init__(self, dig):
        """
        :param dig: the dig function to use, so you can swap out tcp and udp
        """
        self.host = cfg.CONF.slappy.host
        self.port = cfg.CONF.slappy.port
        self.dig = dig

    def create_zone(self, name):
        return self.dig(
            zone_name=name,
            nameserver=self.host,
            port=self.port,
            rdatatype=self.CREATE_RDTYPE,
            rdataclass=self.RDCLASS,
            opcode=self.OPCODE,
        )

    def delete_zone(self, name):
        return self.dig(
            zone_name=name,
            nameserver=self.host,
            port=self.port,
            rdatatype=self.DELETE_RDTYPE,
            rdataclass=self.RDCLASS,
            opcode=self.OPCODE,
        )

    def notify(self, name):
        return self.dig(
            zone_name=name,
            nameserver=self.host,
            port=self.port,
            rdatatype=dns.rdatatype.SOA,
            rdataclass=self.RDCLASS,
            opcode=dns.opcode.NOTIFY,
        )
