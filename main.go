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
package main

import (
    "os"
    
    "./version"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
)
    
func main() {

    app := cli.NewApp()
	app.Name = "trafficrouter"
	app.Version = version.FullVersion()
	app.Author = "@athakwani"
	app.Email = "athakwani@gmail.com"
	app.Usage = "Zero-config push based load balancer"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Enable debug logging",
		},
        cli.StringFlag{
			Name:  "passwd, p",
			Usage: "Password to secure connections",
			Value: "123456789",
		},	
        cli.StringFlag{
			Name:  "require",
            Usage: "Services required for application. Format `app:port@laddr:lport` e.g. db:3306@localhost",
			Value: "",
		},	
        cli.StringFlag{
			Name:  "register",
            Usage: "Services Cmd should register. Format `app-1:port@raddr, app-1:port@raddr` e.g. db:3306@app or db:3360@app-*",
			Value: "",
		},	
        cli.BoolFlag{
			Name:  "forcecmd, f",
            Usage: "Use as force command (Don't set, it is used internally)",
			Hidden: false,
		},	
        cli.StringFlag{
			Name:  "cmd",
            Usage: "Execute Command",
			Value: "",
		},	
        cli.IntFlag{
			Name:  "poll-interval",
            Usage: "Interval to detect new hosts, used with wildcard in --register option",
			Value: 1,
		},	
        cli.StringFlag{
			Name:  "usr, u",
            Usage: "Usernames used for local service discovery (Don't set, it is used internally)",
			Value: "",
		},	
        cli.BoolFlag{
			Name:  "swmp",
            Usage: "Flag to indicate that process was swamped for local service discovery (Don't set, it is used internally)",
			Value: false,
		},	
    }
    
    app.Action = TrafficRouter

    app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

    if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}