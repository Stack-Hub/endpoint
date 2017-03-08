#!/bin/bash
sudo apt-get install -y jq awscli 
set +x
# Only tested on Ubuntu 14.04

# Copy sshd_config
cp sshd_config /etc/ssh/

# copy sudoers
chmod u+w /etc/sudoers
cp sudoers /etc/

# copy shareit
cp reverse_tunnel /home/
chmod +x /home/reverse_tunnel

id -u dupper

if [[ $? -ne 0 ]]; then
    # Add user dupper
    adduser --disabled-password --gecos "" dupper
fi

# copy authorized_keys to /home/dupper/.ssh
mkdir -p /home/dupper/.ssh
cp authorized_keys /home/dupper/.ssh/

# Restart ssh
service ssh restart

iptables -F
iptables -A INPUT -p tcp -s 70.95.164.211 --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --syn --dport 22 -m connlimit --connlimit-above 10 --connlimit-mask 32 -j REJECT --reject-with tcp-reset  
