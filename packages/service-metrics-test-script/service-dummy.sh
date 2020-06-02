set -x
# This metric will be renamed to `service_dummy` in loggregator envelope
# format.
/bin/echo -n '[{"key":"service-dummy","value":99,"unit":"metric"}]'
