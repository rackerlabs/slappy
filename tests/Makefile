MASTER_MAKEFILE=$(shell find . -name "master.makefile")
SLAPPY_MAKEFILE=$(shell find . -name "slappy.makefile")

BIND_MASTER_IP=$(shell make -s -f $(MASTER_MAKEFILE) ip)

SLAPPY_CONF=docker/slappy.conf

help:
	@echo "build-docker-images          - build the docker bind image"
	@echo "start-containers             - start all containers"
	@echo "stop-containers              - stop all containers"

build-docker-images: write-config
	make -f $(MASTER_MAKEFILE) build
	make -f $(SLAPPY_MAKEFILE) build

start-containers:
	make -f $(MASTER_MAKEFILE) start
	make -f $(SLAPPY_MAKEFILE) start

stop-containers:
	make -f $(MASTER_MAKEFILE) stop
	make -f $(SLAPPY_MAKEFILE) stop

write-config:
	@rm -f $(SLAPPY_CONF) && touch $(SLAPPY_CONF)
	@echo "[slappy]"            >> $(SLAPPY_CONF)
	@echo "#################"   >> $(SLAPPY_CONF)
	@echo "# General"           >> $(SLAPPY_CONF)
	@echo "#################"   >> $(SLAPPY_CONF)
	@echo "debug = false"       >> $(SLAPPY_CONF)
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

	@echo "#################"           >> $(SLAPPY_CONF)
	@echo "# DNS Options"               >> $(SLAPPY_CONF)
	@echo "#################"           >> $(SLAPPY_CONF)
	@echo "master = $(BIND_MASTER_IP)"  >> $(SLAPPY_CONF)
	@echo "queries = 127.0.0.1:53"      >> $(SLAPPY_CONF)
	@echo "zone_path = ./"              >> $(SLAPPY_CONF)
	@echo "allow_notify = \"\""         >> $(SLAPPY_CONF)
	@echo "query_timeout = 10"          >> $(SLAPPY_CONF)
	@echo "transfer_source = \"\""      >> $(SLAPPY_CONF)
	@echo ""                            >> $(SLAPPY_CONF)

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