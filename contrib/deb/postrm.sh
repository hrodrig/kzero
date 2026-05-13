#!/bin/sh
# postrm: on purge, remove config directory so no stale secrets remain.
case "$1" in
purge)
	rm -rf /etc/kzero
	;;
esac
exit 0
