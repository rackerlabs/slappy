import time
import unittest

from dns.rdatatype import ANY
from dns.rcode import NOERROR, REFUSED

from functionaltests.common.config import cfg
from functionaltests.common.dig import udp as dig
from functionaltests.common.datagen import random_zone
from functionaltests.common.datagen import random_record
from functionaltests.common.rndc.loader import load_master_rndc_target


class TestDockerRndcTarget(unittest.TestCase):

    def setUp(self):
        super(TestDockerRndcTarget, self).setUp()
        self.client = load_master_rndc_target()
        self.zone = random_zone(name='metadockertest')
        self.nameserver = cfg.CONF.master.host
        self.port = cfg.CONF.master.port

        self._addzone()

    def test_addzone(self):
        # TODO: the zone is added in setUp. check the zone contents here.
        pass

    def test_reload(self):
        record = random_record(self.zone, name='testreload')
        self.zone.records.add(record)
        self.zone.serial += 1

        _, _, ret = self.client.write_zone_file(self.zone)
        self.assertEqual(ret, 0)
        _, _, ret = self.client.reload(self.zone)
        self.assertEqual(ret, 0)

        query = self._dig(name=record.name)
        self.assertEqual(query.rcode(), NOERROR)

    def test_delzone(self):
        _, _, ret = self.client.delzone(self.zone)

        query = self._dig()
        self.assertEqual(query.rcode(), REFUSED)

    def _addzone(self):
        # check that the zone is not on the nameserver
        #
        # note: some nameservers (like 8.8.8.8, at the time of this writing)
        # return NXDOMAIN on zones they don't know anything about. others, like
        # bind in the docker container, return REFUSED on zones they don't know
        # about, and return NXDOMAIN for records not found on zones that exist.
        query = self._dig()
        self.assertEqual(query.rcode(), REFUSED)

        # add the zone
        _, _, ret = self.client.write_zone_file(self.zone)
        self.assertEqual(ret, 0)
        _, _, ret = self.client.addzone(self.zone)
        self.assertEqual(ret, 0)

        # check that the zone was added and is queryable
        query = self._dig()
        self.assertEqual(query.rcode(), NOERROR)

        # `rndc reload` does nothing if the zone file timestamp has not changed
        time.sleep(1)

    def _dig(self, name=None):
        return dig(
            zone_name=name or self.zone.name,
            nameserver=self.nameserver,
            rdatatype=ANY,
            port=self.port,
        )
