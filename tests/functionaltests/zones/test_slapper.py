import time
import unittest

import dns.name
from dns.rdatatype import ANY, SOA
from dns.rdataclass import IN
from dns.rcode import NOERROR, REFUSED

from functionaltests.common import dig
from functionaltests.common.config import cfg
from functionaltests.common.datagen import random_zone
from functionaltests.common.rndc.loader import load_master_rndc_target
from functionaltests.common.client import SlappyClient
from functionaltests.common.utils import parameterized_class
from functionaltests.common.utils import parameterized

DIG_FUNCS = {
    'udp': dict(dig=dig.udp),
    'tcp': dict(dig=dig.tcp),
}


@parameterized_class
class TestSlapper(unittest.TestCase):

    def setUp(self):
        super(TestSlapper, self).setUp()
        self.master = load_master_rndc_target()
        self.zone = random_zone(name='slapper')

    @parameterized(DIG_FUNCS)
    def test_create_zone(self, dig):
        slappy = SlappyClient(dig)
        self._add_zone_to_master(dig)
        self._add_zone_to_target_via_slappy(dig, slappy)

    @parameterized(DIG_FUNCS)
    def test_delete_zone(self, dig):
        slappy = SlappyClient(dig)
        self._add_zone_to_master(dig)
        self._add_zone_to_target_via_slappy(dig, slappy)

        query = slappy.delete_zone(name=self.zone.name)
        self.assertEqual(query.rcode(), NOERROR)

        query = dig(
            zone_name=self.zone.name,
            nameserver=cfg.CONF.slappy.host,
            port=53,
            rdatatype=ANY,
        )
        self.assertEqual(query.rcode(), REFUSED)

    @parameterized(DIG_FUNCS)
    def test_update_zone(self, dig):
        slappy = SlappyClient(dig)
        self._add_zone_to_master(dig)
        self._add_zone_to_target_via_slappy(dig, slappy)

        # bump the serial of the zone on the master
        # note: rndc reload needs to see the zone file timestamp has changed
        time.sleep(1)
        self.zone.serial += 1
        self._update_zone_on_master(dig)

        # notify slappy to pull down the new zone
        query = slappy.notify(name=self.zone.name)
        self.assertEqual(query.rcode(), NOERROR)

        # if slappy hasn't added the zone within a second, this test will fail
        time.sleep(1)

        query = dig(
            zone_name=self.zone.name,
            nameserver=cfg.CONF.slappy.host,
            rdatatype=ANY,
            port=53,
        )
        self._check_serial(query, expected=self.zone.serial)

    def _add_zone_to_master(self, dig):
        _, _, ret = self.master.write_zone_file(self.zone)
        self.assertEqual(ret, 0)
        _, _, ret = self.master.addzone(self.zone)
        self.assertEqual(ret, 0)

        query = dig(
            zone_name=self.zone.name,
            nameserver=cfg.CONF.master.host,
            port=cfg.CONF.master.port,
            rdatatype=ANY,
        )
        self.assertEqual(query.rcode(), NOERROR)
        self._check_serial(query, expected=self.zone.serial)

    def _update_zone_on_master(self, dig):
        _, _, ret = self.master.write_zone_file(self.zone)
        self.assertEqual(ret, 0)
        _, _, ret = self.master.reload(self.zone)
        self.assertEqual(ret, 0)

        query = dig(
            zone_name=self.zone.name,
            nameserver=cfg.CONF.master.host,
            port=cfg.CONF.master.port,
            rdatatype=ANY,
        )
        self.assertEqual(query.rcode(), NOERROR)
        self._check_serial(query, expected=self.zone.serial)

    def _add_zone_to_target_via_slappy(self, dig, slappy):
        query = slappy.create_zone(name=self.zone.name)
        self.assertEqual(query.rcode(), NOERROR)

        query = dig(
            zone_name=self.zone.name,
            nameserver=cfg.CONF.slappy.host,
            rdatatype=ANY,
            port=53,
        )
        self.assertEqual(query.rcode(), NOERROR)
        self._check_serial(query, expected=self.zone.serial)

    def _check_serial(self, query, expected):
        # find_rrset raises a key error on not found
        soa = query.find_rrset(
            query.answer, dns.name.from_text(self.zone.name), IN, SOA
        )
        self.assertEqual(soa[0].serial, expected)
