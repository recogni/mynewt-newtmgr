/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/recogni/newtmgr/nmxact/bledefs"
	"github.com/recogni/newtmgr/nmxact/nmble"
	"github.com/recogni/newtmgr/nmxact/nmxutil"
	"github.com/recogni/newtmgr/nmxact/sesn"
	"github.com/recogni/newtmgr/nmxact/xact"
	"github.com/recogni/newtmgr/nmxact/xport"
	"mynewt.apache.org/newt/util"
)

func configExitHandler(x xport.Xport, s sesn.Sesn) {
	onExit := func() {
		if s.IsOpen() {
			s.Close()
		}

		x.Stop()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)

	go func() {
		for {
			s := <-sigChan
			switch s {
			case os.Interrupt, syscall.SIGTERM:
				go func() {
					onExit()
					os.Exit(0)
				}()

			case syscall.SIGQUIT:
				util.PrintStacks()
			}
		}
	}()
}

func main() {
	nmxutil.SetLogLevel(log.DebugLevel)
	//nmxutil.SetLogLevel(log.InfoLevel)

	// Initialize the BLE transport.
	params := nmble.NewXportCfg()
	params.SockPath = "/tmp/blehostd-uds"
	params.BlehostdPath = "blehostd"
	params.DevPath = "/dev/cu.usbmodem142141"

	x, err := nmble.NewBleXport(params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating BLE transport: %s\n",
			err.Error())
		os.Exit(1)
	}

	// Start the BLE transport.
	if err := x.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error starting BLE transport: %s\n",
			err.Error())
		os.Exit(1)
	}
	defer x.Stop()

	// Find a device to connect to:
	//     * Peer has name "nimble-bleprph"
	//     * We use a random address.
	dev, err := nmble.DiscoverDeviceWithName(
		x, bledefs.BLE_ADDR_TYPE_RANDOM, 10*time.Second, "c4")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error discovering device: %s\n", err.Error())
		os.Exit(1)
	}
	if dev == nil {
		fmt.Fprintf(os.Stderr, "couldn't find device")
		os.Exit(1)
	}

	// Prepare a BLE session:
	//     * Plain NMP (not tunnelled over OIC).
	//     * We use a random address.
	sc := sesn.NewSesnCfg()
	sc.MgmtProto = sesn.MGMT_PROTO_OMP
	sc.Ble.OwnAddrType = bledefs.BLE_ADDR_TYPE_RANDOM
	sc.PeerSpec.Ble = *dev

	s, err := x.BuildSesn(sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating BLE session: %s\n", err.Error())
		os.Exit(1)
	}

	configExitHandler(x, s)

	// Repeatedly:
	//     * Connect to peer if unconnected.
	//     * Send an echo command to peer.
	//
	// If blehostd crashes or the controller is unplugged, nmxact should
	// recover on the next connect attempt.
	for {
		if !s.IsOpen() {
			// Connect to the peer (open the session).
			if err := s.Open(); err != nil {
				fmt.Fprintf(os.Stderr, "error starting BLE session: %s\n",
					err.Error())
				time.Sleep(time.Second)
				continue
			}
		}

		// Send an echo command to the peer.
		c := xact.NewEchoCmd()
		c.Payload = "hello"

		res, err := c.Run(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing echo command: %s\n",
				err.Error())
			continue
		}

		if res.Status() != 0 {
			fmt.Printf("Peer responded negatively to echo command; status=%d\n",
				res.Status())
		}

		eres := res.(*xact.EchoResult)
		fmt.Printf("Peer echoed back: %s\n", eres.Rsp.Payload)
	}
}
