PORTNAME=	kzero
PORTVERSION=	1.1.0
DISTVERSIONPREFIX=	v
CATEGORIES=	sysutils
# Local test before release: copy the tarball into DISTDIR (see `make -V DISTDIR`
# from this port directory) using the same name as DISTFILES, or run e.g.
#   make fetch MASTER_SITES=file:///absolute/path/to/dir/
# See README.md "Test with a local distfile".
MASTER_SITES=	https://github.com/hrodrig/kzero/releases/download/v${PORTVERSION}/
DISTFILES=	kzero_v${PORTVERSION}_freebsd_${ARCH:S/aarch64/arm64/}.tar.gz
EXTRACT_SUFX=	.tar.gz

MAINTAINER=	hrodrig@usb.ve
COMMENT=	Declarative Kubernetes workload shutdown and startup pipelines (CLI)
WWW=		https://github.com/hrodrig/kzero
LICENSE=	MIT

NO_BUILD=	yes
NO_ARCH=	yes

do-install:
	${INSTALL_PROGRAM} ${WRKDIR}/kzero ${STAGEDIR}${PREFIX}/bin/kzero
	${INSTALL_PROGRAM} ${WRKDIR}/kubectl-kzero ${STAGEDIR}${PREFIX}/bin/kubectl-kzero
	${MKDIR} ${STAGEDIR}${PREFIX}/share/man/man1
	${INSTALL_MAN} ${WRKDIR}/share/man/man1/kzero.1 ${STAGEDIR}${PREFIX}/share/man/man1/
	${MKDIR} ${STAGEDIR}${PREFIX}/share/doc/kzero
	${INSTALL_DATA} ${WRKDIR}/share/doc/kzero/LICENSE ${STAGEDIR}${PREFIX}/share/doc/kzero/
	${MKDIR} ${STAGEDIR}${PREFIX}/share/examples/kzero
	${INSTALL_DATA} ${WRKDIR}/share/examples/kzero/kzero.sample.yml ${STAGEDIR}${PREFIX}/share/examples/kzero/

.include <bsd.port.mk>
