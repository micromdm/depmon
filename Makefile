PKGNAME=depnag
PKGVERSION=1.0.0
PKGID=io.micromdm.depnag


.PHONY: clean build package

all: clean build package
clean:
	rm -rf build/
	rm -rf pkgroot/usr/local/bin/

build: clean
	mkdir -p build
	mkdir -p pkgroot/usr/local/bin/
	./release.sh

package:
	cp build/depnag-darwin-amd64 pkgroot/usr/local/bin/depnag
	pkgbuild --root pkgroot --scripts scripts --identifier ${PKGID} --version ${PKGVERSION} build/${PKGNAME}-${PKGVERSION}.pkg


	

