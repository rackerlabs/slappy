"""
We want a uniform, pluggable way to administer a Bind server from the
tests, including creating/deleting zone files and running rndc commands.
For example, we might have:

    1. Bind running in a docker container
    2. Bind running on the same machine as the tests
    3. Bind running on some other cloud server

This gives the tests an interface for managing any of these bind servers.
"""

from abc import abstractmethod
import logging

LOG = logging.getLogger(__name__)


class BaseRndcTarget(object):

    @abstractmethod
    def addzone(self, zone):
        """Run `rndc addzone`. This does not create the zone file. The zone
        file must already exist. The implementing class should know the zone
        file path based on the zone name.

        :param zone: the Zone to add
        :return: A tuple (out, err, ret) containing the stdout, stderr, and
            return code of the rndc command
        """
        LOG.debug("Running addzone(%s)", zone)

    @abstractmethod
    def delzone(self, zone):
        """Run `rndc delzone`. This does not delete the zone file.

        :return: A tuple (out, err, ret) containing the stdout, stderr, and
            return code of the rndc command
        """
        LOG.debug("Running delzone(%s)", zone)

    @abstractmethod
    def reload(self, zone):
        """Run `rndc reload` on the given zone

        :return: A tuple (out, err, ret) containing the stdout, stderr, and
            return code of the rndc command
        """
        LOG.debug("Running reload(%s)", zone)

    @abstractmethod
    def write_zone_file(self, zone):
        """Write out the given zone to a path on the server. The file name of
        the zone file will be chosen by the implementing class based on the
        zone name. That file will be overwritten if it already exists."""
        LOG.debug("Running write_zone_file(%s)", zone)

    @abstractmethod
    def delete_zone_file(self, zone):
        """Remove the zone file from the server"""
        LOG.debug("Running write_zone_file(%s)", zone)
