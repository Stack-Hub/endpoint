// +build darwin

/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package forcecmd

import (
    "os"
    "os/signal"
    "syscall"

    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"    
)


/*
 * Wait for parent to exit.
 */
func wait(fd int) {

    sigs := make(chan os.Signal, 1)    
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        sig := <-sigs
        log.Debug(sig)
        utils.UnlockFile(fd)
        os.Exit(1)
    }()
    
    utils.BlockForever()
}