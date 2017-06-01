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
VERSION=0.1.0
BINARY=trafficrouter
INSTALL_DIR=/usr/local

command_exists() {
	command -v "$@" > /dev/null 2>&1
}

install_ssh() {
    cmd="$1"
    
    cat >&2 <<EOF
Unable to install ssh server, sshpass & flock, please install required dependencies and rerun trafficrouter install scrit.

  $cmd
    
EOF
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

missing_dependencies() {
    local _dependencies="${1}"
    cat >&2 <<EOF
    Missing Dependency: "$_dependencies"
    Please install ssh server for Mac and run this script again.
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
    dist_version=''
    if command_exists lsb_release; then
        lsb_dist="$(lsb_release -si)"
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/lsb-release ]; then
        lsb_dist="$(. /etc/lsb-release && echo "$DISTRIB_ID")"
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/debian_version ]; then
        lsb_dist='debian'
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/fedora-release ]; then
        lsb_dist='fedora'
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/oracle-release ]; then
        lsb_dist='oracleserver'
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/centos-release ]; then
        lsb_dist='centos'
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/redhat-release ]; then
        lsb_dist='redhat'
    fi
    if [ -z "$lsb_dist" ] && [ -r /etc/os-release ]; then
        lsb_dist="$(. /etc/os-release && echo "$ID")"
    fi

    lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"

    # Special case redhatenterpriseserver
    if [ "${lsb_dist}" = "redhatenterpriseserver" ]; then
            # Set it to redhat, it will be changed to centos below anyways
            lsb_dist='redhat'
    fi

	if ! command_exists ssh; then
        dependencies="ssh "
    fi

    [ "$dependencies" != "" ] && [ "$OS" = "Darwin" ] && missing_dependencies "$dependencies"

	if ! command_exists ssh; then
        #install docker

        case "$lsb_dist" in

            ubuntu|debian)
                apt-get install -y ssh sshpass flock
            ;;

            'opensuse project'|opensuse|'suse linux'|sle[sd]|fedora|centos|redhat|gentoo)
                yum install -y ssh sshpass flock
            ;;
            
            *)
                install_ssh
            ;;

        esac
    fi

    # install dupper binaries at /usr/local/bin 
    URL="${ROOT_URL}/release/${OS}/${MACHINE}/${BINARY}-${VERSION}.tgz"
    $CURL ${URL} > /tmp/${BINARY}-${VERSION}.tgz
    tar -xvzf /tmp/${BINARY}-${VERSION}.tgz -C ${INSTALL_DIR} 
    rm /tmp/${BINARY}-${VERSION}.tgz
    
    exit 0
}

# install binary and dependencies
install

