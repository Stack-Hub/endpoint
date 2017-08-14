// +build linux

/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package config

import (
    "os"
    "os/signal"
    "syscall"

    "github.com/duppercloud/trafficrouter/utils"
    "golang.org/x/sys/unix"
    log "github.com/Sirupsen/logrus"    
)


/*
 * Wait for parent to exit.
 */
func wait(fptr *os.File) {

    sigs := make(chan os.Signal, 1)    
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        sig := <-sigs
        log.Debug(sig)
        utils.UnlockFile(fptr)
        os.Exit(0)
    }()

    flag := unix.SIGHUP
    if err := unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(flag), 0, 0, 0); err != nil {
        return
    }
    
    utils.BlockForever()
}