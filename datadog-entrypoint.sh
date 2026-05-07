#!/bin/bash
rm -rf /etc/datadog-agent/datadog.yaml

cat > /etc/datadog-agent/datadog.yaml << EOF
network_path:
  collector:
    workers: 0
  connections_monitoring:
    enabled: false
system_probe_config:
  enabled: false
service_monitoring_config:
  enabled: false
runtime_security_config:
  enabled: false
compliance_config:
  enabled: false
process_config:
  enabled: false
cloud_provider_metadata: []
workloadmeta:
  enabled: false
EOF

# Config específico do system-probe para garantir que não sobe
cat > /etc/datadog-agent/system-probe.yaml << EOF
system_probe_config:
  enabled: false
network_config:
  enabled: false
service_monitoring_config:
  enabled: false
runtime_security_config:
  enabled: false
event_monitoring_config:
  enabled: false
EOF

exec /bin/entrypoint.sh