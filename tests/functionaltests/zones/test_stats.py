import unittest

import dns.rdatatype
import dns.rcode

from functionaltests.common import dig
from functionaltests.common.config import cfg
from functionaltests.common.utils import parameterized_class
from functionaltests.common.utils import parameterized


@parameterized_class
class TestStats(unittest.TestCase):

    @parameterized({
        'udp': dict(dig=dig.udp),
        'tcp': dict(dig=dig.tcp),
    })
    def test_stats(self, dig):
        result = dig(
            zone_name='/stats.',
            nameserver=cfg.CONF.slappy.host,
            rdatatype=dns.rdatatype.ANY,
            port=cfg.CONF.slappy.port,
        )
        self.assertEqual(result.rcode(), dns.rcode.NOERROR)

        # sanity-checking our own dns request
        self.assertEqual(len(result.question), 1)
        self.assertEqual(str(result.question[0].name), '/stats.')
        self.assertEqual(result.question[0].rdclass, dns.rdataclass.IN)
        self.assertEqual(result.question[0].rdtype, dns.rdatatype.ANY)

        # check the recordset
        self.assertEqual(len(result.answer), 1)
        self.assertEqual(str(result.answer[0].name), '/stats.')

        # the stats data is provided "key: val" strings in txt records.
        # convert the list of txt records to a dictionary.
        stats_data = {}
        for record in result.answer[0]:
            key, val = str(record).strip('"').split(':')
            stats_data[key.strip()] = val.strip()

        expected_keys = set([
            "uptime",
            "goroutines",
            "memory",
            "nextGC",
            "queries",
            "addzones",
            "reloads",
            "delzones",
            "rndc_attempts",
            "rndc_success",
        ])
        self.assertEqual(set(stats_data.keys()), expected_keys)

        # do some basic validation of stats data
        self.assertGreaterEqual(int(stats_data['uptime']), 0)
        self.assertGreaterEqual(int(stats_data['goroutines']), 0)
        self.assertGreaterEqual(int(stats_data['queries']), 0)
        self.assertGreaterEqual(int(stats_data['addzones']), 0)
        self.assertGreaterEqual(int(stats_data['reloads']), 0)
        self.assertGreaterEqual(int(stats_data['delzones']), 0)
        self.assertGreaterEqual(int(stats_data['rndc_attempts']), 0)
        self.assertGreaterEqual(int(stats_data['rndc_success']), 0)
