SPECFILE=javadocr.spec
VERSION=$(shell grep '%define version' ${SPECFILE} | awk '{print $$3}')
RELEASE=$(shell grep '^Release:' ${SPECFILE} | awk '{print $$2}' | grep -o '[[:digit:]]\+')
ARCH=x86_64
MOCK=/usr/bin/mock
MOCKENV=fedora-23-${ARCH}
SRPM_SUFFIX=.fc23.src.rpm
RPM_SUFFIX=.fc23.${ARCH}.rpm
RPM_PREFIX=javadocr-${VERSION}-${RELEASE}
DEST_REPO_PATH=Fedora/23

TARGET_SERVER=bookshelf.spongeproject.net
REPO_PATH=/mnt/data/repo

SRPM_NAME=${RPM_PREFIX}${SRPM_SUFFIX}
RPM_NAME=${RPM_PREFIX}${RPM_SUFFIX}

ifdef BUILD_NUMBER
BUILD_PARAM = -D 'build_number ${BUILD_NUMBER}'
RELEASE = ${BUILD_NUMBER}
else
BUILD_PARAM = 
endif

all: ${RPM_NAME}

master.tar.gz:
	curl -L https://github.com/lukegb/javadocr/archive/master.tar.gz > $@

${SRPM_NAME}: master.tar.gz javadocr.spec javadocr.service
	${MOCK} -r ${MOCKENV} ${BUILD_PARAM} --buildsrpm --sources=. --spec=${SPECFILE} && cp /var/lib/mock/${MOCKENV}/result/$@ $@

${RPM_NAME}: ${SRPM_NAME}
	${MOCK} -r ${MOCKENV} ${BUILD_PARAM} --rebuild $< && cp /var/lib/mock/${MOCKENV}/result/$@ $@

install: ${SRPM_NAME} ${RPM_NAME}
	scp ${SRPM_NAME} ${TARGET_SERVER}:${REPO_PATH}/html/${DEST_REPO_PATH}/SRPMS/${SRPM_NAME}
	ssh ${TARGET_SERVER} 'cd ${REPO_PATH}/html/${DEST_REPO_PATH}/SRPMS && createrepo_c --update .'
	scp ${RPM_NAME} ${TARGET_SERVER}:${REPO_PATH}/html/${DEST_REPO_PATH}/${ARCH}/${RPM_NAME}
	ssh ${TARGET_SERVER} 'cd ${REPO_PATH}/html/${DEST_REPO_PATH}/${ARCH} && createrepo_c --update .'

clean:
	rm -f ${RPM_NAME} ${SRPM_NAME}

distclean: clean
	rm -f master.tar.gz
