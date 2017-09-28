#!/bin/sh
set -e
# Owner: Ashish Thakwani (athakwani at gmail dot com)
# This script is meant for quick & easy install of trafficrouter via:
#   'curl -sSL https://get.dupper.co/trafficrouter | sudo sh'
# or:
#   'wget -qO- https://get.dupper.co/trafficrouter | sudo sh'
#

check_perm() {
    if [ "$(whoami)" != 'root' ]; then
        echo "This installation require sudo access. Please run with sudo access:
        curl -sSL https://get.dupper.co | sudo sh"
        exit
    fi
}

check_perm

ROOT_URL=https://get.dupper.co/trafficrouter
VERSION=0.0.4
BINARY=trafficrouter
INSTALL_DIR=/usr/local

command_exists() {
	command -v "$@" > /dev/null 2>&1
}

unsupported_distro() {
    cat >&2 <<-'EOF'
    Error: Unsupported Distribution. 
EOF
    exit 1
}

unsupported_os() {
    cat >&2 <<-'EOF'
    Error: Unsupported OS. Trafficrouter currently only supports Linux 64-bit OS.
EOF
    exit 1
}

install() {
    OS=""
    MACHINE=""
    
    # Check machine arch
    case "$(uname -m)" in
		*64)
            MACHINE=x86_64
            ;;
		*)
            unsupported_os
            ;;
	esac

    # Check OS
    OS="$(uname -s)"
    case $OS in
		Linux)
            ;;
		*)
            unsupported_os
            ;;
	esac

	CURL=''
	if command_exists curl; then
		CURL='curl -sSL'
	elif command_exists wget; then
		CURL='wget -qO-'
	elif command_exists busybox && busybox --list-modules | grep -q wget; then
		CURL='busybox wget -qO-'
	fi

    # Detect platform
    lsb_dist=''
    if [ -r /etc/debian_version ]; then
        lsb_dist='debian'
    fi
    if [ -r /etc/alpine-release ]; then
        lsb_dist='alpine'
    fi

    case "$lsb_dist" in

        debian)
            apt-get install -y ssh sshpass
        ;;

        alpine)
            apk add openssh sshpass
        ;;

        *)
            unsupported_distro
        ;;

    esac

    # install dupper binaries at /usr/local/bin 
    URL="${ROOT_URL}/release/${OS}/${MACHINE}/${BINARY}-${VERSION}.tgz"
    $CURL ${URL} > /tmp/${BINARY}-${VERSION}.tgz
    tar -xvzf /tmp/${BINARY}-${VERSION}.tgz -C ${INSTALL_DIR} 
    chown root:root ${INSTALL_DIR}/bin/${BINARY}
    chmod u+s ${INSTALL_DIR}/bin/${BINARY}
    rm /tmp/${BINARY}-${VERSION}.tgz
    
    exit 0
}

# install binary and dependencies
install

