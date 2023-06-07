.PHONY: ldap
ldap: build
	cloudquery sync ldap.yaml --log-level=debug --telemetry-level=none

.PHONY: build
build:
	cd .. && make && cd _test

.PHONY: reset
reset:
	rm -rf *.log *.sqlite

.PHONY: internal
internal: build
	cloudquery sync internal.yaml --log-level=debug --telemetry-level=none