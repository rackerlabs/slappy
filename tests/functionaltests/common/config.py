import os
from oslo_config import cfg

# look for the file in these locations, in order. use the first file found.
_LOCATIONS = (
    os.path.realpath('test.conf'),
)

cfg.CONF.register_group(cfg.OptGroup('slappy'))
cfg.CONF.register_group(cfg.OptGroup('master'))
cfg.CONF.register_group(cfg.OptGroup('tests'))

cfg.CONF.register_opts([
    cfg.StrOpt('host'),
    cfg.IntOpt('port'),
], group='slappy')

cfg.CONF.register_opts([
    cfg.StrOpt('host'),
    cfg.IntOpt('port'),
], group='master')


def _find_config_file():
    for location in _LOCATIONS:
        if os.path.exists(location):
            return location
    raise Exception("Failed to find digaas.conf at any of these paths: {0}"
                    .format(_LOCATIONS))

cfg.CONF(args=[], default_config_files=[_find_config_file()])
