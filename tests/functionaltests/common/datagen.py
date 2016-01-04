import random

from functionaltests.common.model import Record
from functionaltests.common.model import Zone


def random_ip():
    return ".".join(str(random.randrange(0, 256)) for _ in range(4))


def random_digits(n=8):
    return "".join([str(random.randint(0, 9)) for _ in range(n)])


def random_domain(name='testdomain', tld='com'):
    return '{0}{1}.{2}.'.format(name, random_digits(), tld)


def random_zone(name='testdomain', ttl=2400, serial=123456):
    return Zone(
        name=random_domain(name),
        ttl=ttl,
        serial=serial,
    )


def random_record(zone, name='testrecord'):
    return Record(
        name='{0}{1}.{2}'.format(name, random_digits(), zone.name),
        type='A',
        data=random_ip(),
    )
