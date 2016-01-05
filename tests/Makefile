MASTER_MAKEFILE=$(shell find . -name "master.makefile")
SLAPPY_MAKEFILE=$(shell find . -name "slappy.makefile")

MASTER_IP=$(shell make -s -f $(MASTER_MAKEFILE) ip)
MASTER_TAG=$(shell make -s -f $(MASTER_MAKEFILE) tag)
SLAPPY_IP=$(shell make -s -f $(SLAPPY_MAKEFILE) ip)

SLAPPY_CONF=docker/slappy.conf
TEST_CONF=test.conf

help:
	@echo "start-containers             - start all containers"
	@echo "stop-containers              - stop all containers"
	@echo "check-containers             - check the containers are running the things"
	@echo "ips                          - print the ips of our containers"
	@echo "write-config                 - write the slappy config used in the docker container"
	@echo "write-test-config            - write the config for the tests"


# the slappy config needs the ip of the master
start-containers: build-and-start-master write-config build-and-start-slappy

build-and-start-master:
	make -f $(MASTER_MAKEFILE) build
	make -f $(MASTER_MAKEFILE) start

build-and-start-slappy:
	make -f $(SLAPPY_MAKEFILE) build
	make -f $(SLAPPY_MAKEFILE) start

stop-containers:
	make -f $(MASTER_MAKEFILE) stop
	make -f $(SLAPPY_MAKEFILE) stop

check-containers:
	make -f $(MASTER_MAKEFILE) check
	make -f $(SLAPPY_MAKEFILE) check

ips:
	@echo "MASTER: `make -s -f $(MASTER_MAKEFILE) ip`"
	@echo "TARGET: `make -s -f $(SLAPPY_MAKEFILE) ip`"

write-config:
	@rm -f $(SLAPPY_CONF) && touch $(SLAPPY_CONF)
	@echo "[slappy]"            >> $(SLAPPY_CONF)
	@echo "#################"   >> $(SLAPPY_CONF)
	@echo "# General"           >> $(SLAPPY_CONF)
	@echo "#################"   >> $(SLAPPY_CONF)
	@echo "debug = true"        >> $(SLAPPY_CONF)
	@echo "log = slappy.log"    >> $(SLAPPY_CONF)
	@echo "use_syslog = false"  >> $(SLAPPY_CONF)
	@echo ""                    >> $(SLAPPY_CONF)

	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "# Network Options"       >> $(SLAPPY_CONF)
	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "bind_address = 0.0.0.0"  >> $(SLAPPY_CONF)
	@echo "bind_port = 5358"        >> $(SLAPPY_CONF)
	@echo "all_tcp = true"          >> $(SLAPPY_CONF)
	@echo ""                        >> $(SLAPPY_CONF)

	@echo "#################"               >> $(SLAPPY_CONF)
	@echo "# DNS Options"                   >> $(SLAPPY_CONF)
	@echo "#################"               >> $(SLAPPY_CONF)
	@echo "master = $(MASTER_IP):53"        >> $(SLAPPY_CONF)
	@echo "queries = 127.0.0.1:53"          >> $(SLAPPY_CONF)
	@echo "zone_path = /var/cache/bind/"    >> $(SLAPPY_CONF)
	@echo "allow_notify = \"\""             >> $(SLAPPY_CONF)
	@echo "query_timeout = 10"              >> $(SLAPPY_CONF)
	@echo "transfer_source = \"\""          >> $(SLAPPY_CONF)
	@echo ""                                >> $(SLAPPY_CONF)

	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "# Limit RNDC"            >> $(SLAPPY_CONF)
	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "limit_rndc = false"      >> $(SLAPPY_CONF)
	@echo "rndc_limit = 50"         >> $(SLAPPY_CONF)
	@echo "rndc_timeout = 25"       >> $(SLAPPY_CONF)
	@echo ""                        >> $(SLAPPY_CONF)

	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "# Stats/Monitoring"      >> $(SLAPPY_CONF)
	@echo "#################"       >> $(SLAPPY_CONF)
	@echo "status_file = """        >> $(SLAPPY_CONF)
	@echo "status_interval = 60"    >> $(SLAPPY_CONF)
	@echo "stats_uri = /stats."     >> $(SLAPPY_CONF)
	@echo ""                        >> $(SLAPPY_CONF)

write-test-config:
	@rm -f $(TEST_CONF) && touch $(TEST_CONF)
	@echo "[slappy]"            >> $(TEST_CONF)
	@echo "host = $(SLAPPY_IP)" >> $(TEST_CONF)
	@echo "port = 5358"         >> $(TEST_CONF)
	@echo ""                    >> $(TEST_CONF)

	@echo "[master]"                    >> $(TEST_CONF)
	@echo "host = $(MASTER_IP)"         >> $(TEST_CONF)
	@echo "port = 53"                   >> $(TEST_CONF)
	@echo "rndc_target_type = docker"   >> $(TEST_CONF)
	@echo ""                            >> $(TEST_CONF)

	@echo "[master:docker]"         >> $(TEST_CONF)
	@echo "id = $(MASTER_TAG)"      >> $(TEST_CONF)
	@echo "dir = /var/cache/bind/"  >> $(TEST_CONF)
	@echo ""                        >> $(TEST_CONF)
