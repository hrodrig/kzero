# kzero — BSD Make (FreeBSD) stub: forwards to GNU Make.
# GNU Make reads GNUmakefile before this file on Linux/macOS/CI.
#
# On FreeBSD: pkg install gmake   then   make <target>   or   gmake <target>

_CHECK := command -v gmake >/dev/null 2>&1 || { echo "This project requires GNU make. On FreeBSD: pkg install gmake"; exit 1; }

all:
	@${_CHECK}
	@gmake help

.DEFAULT:
	@${_CHECK}
	@gmake $@
