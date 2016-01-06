from functionaltests.common.rndc.utils import run_cmd
from functionaltests.common.rndc.utils import parse_rndc_version
from functionaltests.common.rndc.targets.base import BaseRndcTarget


class DockerRndcTarget(BaseRndcTarget):

    def __init__(self, zone_file_dir, container_id):
        """
        :param container_id: the container name or id
        """
        self.zone_file_dir = zone_file_dir.rstrip('/')
        self.container_id = container_id

    def _docker_exec(self, *cmd):
        cmd = ["docker", "exec", self.container_id] + list(cmd)
        return run_cmd(*cmd)

    def _get_zone_file_path(self, zone):
        return "{0}/{1}.zone".format(self.zone_file_dir, zone.name.rstrip('.'))

    def _get_rndc_version(self):
        """Return a tuple like (9, 9, 5) that specifies the version of rndc and
        Bind running in the docker container
        """
        out, _, _, = self._docker_exec('rndc', 'status')
        return parse_rndc_version(out)

    def write_zone_file(self, zone):
        super(DockerRndcTarget, self).write_zone_file(zone)
        text = zone.zone_file_string().replace('\n', '\\n')
        path = self._get_zone_file_path(zone)
        return self._docker_exec(
            'bash', '-c', "echo -e '{0}' > {1}".format(text, path)
        )

    def delete_zone_file(self, zone):
        super(DockerRndcTarget, self).delete_zone_file(zone)
        path = self._get_zone_file_path(zone)
        return self._docker_exec('rm', '-f', path)

    def addzone(self, zone):
        super(DockerRndcTarget, self).addzone(zone)
        # ensure we have a trailing dot
        zone_name = zone.name.rstrip('.') + '.'
        # this file must already exist
        path = self._get_zone_file_path(zone)
        return self._docker_exec(
            'rndc', 'addzone', zone_name, '{ type master; file "%s"; };' % path
        )

    def reload(self, zone):
        super(DockerRndcTarget, self).reload(zone)
        zone_name = zone.name.rstrip('.') + '.'
        return self._docker_exec('rndc', 'reload', zone_name)

    def delzone(self, zone):
        super(DockerRndcTarget, self).delzone(zone)

        # on Ubuntu 14.04 (rndc 9.9.5-3ubuntu0.2-Ubuntu), rndc delzone cannot
        # have the trailing '.' but on Ubuntu 12.04 (rndc 9.8.1-P1), rndc
        # delzone MUST have the trailing '.'
        #
        # I'm not sure the specific version where this changed, but this makes
        # it work on these two versions of Ubuntu
        if self._get_rndc_version() >= (9, 9, 0):
            zone_name = zone.name.rstrip('.')
        else:
            zone_name = zone.name.rstrip('.') + '.'
        return self._docker_exec('rndc', 'delzone', zone_name)
