#!/usr/bin/env bash

mv /usr/local/bin/hiveos-prometheus /usr/local/bin/hiveos-prometheus.bak
wget "$1"/hiveos_prometheus/hiveos-prometheus -qO /usr/local/bin/hiveos-prometheus
chmod +x /usr/local/bin/hiveos-prometheus
wget "$1"/hiveos_prometheus/hiveos-prometheus.service -qO /etc/systemd/system/hiveos-prometheus.service
systemctl daemon-reload
systemctl enable --now hiveos-prometheus.service
