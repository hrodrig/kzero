# OpenBSD port for kzero (Kubernetes pipeline CLI)
# Copy to /usr/ports/sysutils/kzero/ and run: make makesum
# Submit: send diff to ports@openbsd.org

COMMENT =	declarative Kubernetes workload pipelines (CLI)
DISTNAME =	kzero_v0.9.1_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}
PKGNAME =	kzero-0.9.1
CATEGORIES =	sysutils
HOMEPAGE =	https://github.com/hrodrig/kzero
MAINTAINER =	hrodrig@usb.ve

# MIT
PERMIT_PACKAGE =	Yes
PERMIT_DISTFILES =	Yes

# Local test before release: copy tarball to DISTDIR or use
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local tarball".
MASTER_SITES =	https://github.com/hrodrig/kzero/releases/download/v0.9.1/
DISTFILES =	kzero_v0.9.1_openbsd_${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX =	.tar.gz

ONLY_FOR_ARCHS =	amd64 arm64 aarch64
NO_BUILD =	Yes
NO_TEST =	Yes
PKG_ARCH =	*

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/kzero ${PREFIX}/bin/kzero
	${INSTALL_PROGRAM} ${WRKDIR}/kubectl-kzero ${PREFIX}/bin/kubectl-kzero
	${INSTALL_MAN} ${WRKDIR}/share/man/man1/kzero.1 ${PREFIX}/man/man1/kzero.1
	${INSTALL_DATA_DIR} ${PREFIX}/share/doc/kzero
	${INSTALL_DATA} ${WRKDIR}/share/doc/kzero/LICENSE ${PREFIX}/share/doc/kzero/
	${INSTALL_DATA_DIR} ${PREFIX}/share/examples/kzero
	${INSTALL_DATA} ${WRKDIR}/share/examples/kzero/kzero.sample.yml ${PREFIX}/share/examples/kzero/

.include <bsd.port.mk>
