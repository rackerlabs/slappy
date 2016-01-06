from functionaltests.common.config import cfg
from functionaltests.common.rndc.targets.docker import DockerRndcTarget

CONF = cfg.CONF


def load_master_rndc_target():
    if CONF.master.rndc_target_type == 'docker':
        return DockerRndcTarget(
            zone_file_dir=CONF['master:docker'].dir,
            container_id=CONF['master:docker'].id,
        )
    else:
        raise Exception("Unable to load bind target-type '%s'"
                        % CONF.master.rndc_target_type)
