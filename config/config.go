package config

const (
    RUNPATH     = "/var/run/trafficrouter/"
    SSHD_CONFIG = "/etc/ssh/sshd_config" 
    MATCHBLK    = `
Match User %s
    AllowTCPForwarding yes
    X11Forwarding no
    AllowAgentForwarding no
    PermitTTY yes
    ForceCommand mkdir -p `+ RUNPATH +` && flock `+ RUNPATH +`$$ -c "/usr/sbin/trafficrouter -t $SSH_ORIGINAL_COMMAND"
`
    SERVER_HOST = "0.0.0.0"
    SERVER_PORT = "80"
    SERVER_TYPE = "tcp"
)
