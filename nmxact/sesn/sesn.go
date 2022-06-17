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

package sesn

import (
	"time"

	"github.com/runtimeco/go-coap"

	"github.com/recogni/newtmgr/nmxact/nmcoap"
	"github.com/recogni/newtmgr/nmxact/nmp"
)

var DfltTxOptions = TxOptions{
	Timeout: 10 * time.Second,
	Tries:   1,
}

type NotifyCb func(msg coap.Message, err error)

type TxOptions struct {
	Timeout time.Duration
	Tries   int
}

func NewTxOptions() TxOptions {
	return DfltTxOptions
}

func (opt *TxOptions) AfterTimeout() <-chan time.Time {
	if opt.Timeout == 0 {
		return nil
	} else {
		return time.After(opt.Timeout)
	}
}

// Represents a communication session with a specific peer.  The particulars
// vary according to protocol and transport. Several Sesn instances can use the
// same Xport.
type Sesn interface {
	////// Public interface:

	// Initiates communication with the peer.  For connection-oriented
	// transports, this creates a connection.
	// Returns:
	//     * nil: success.
	//     * nmxutil.SesnAlreadyOpenError: session already open.
	//     * other error
	Open() error

	// Ends communication with the peer.  For connection-oriented transports,
	// this closes the connection.
	//     * nil: success.
	//     * nmxutil.SesnClosedError: session not open.
	//     * other error
	Close() error

	// Indicates whether the session is currently open.
	IsOpen() bool

	// Retrieves the maximum data payload for incoming data packets.
	MtuIn() int

	// Retrieves the maximum data payload for outgoing data packets.
	MtuOut() int

	MgmtProto() MgmtProto

	// Indicates whether the session uses the TCP form of CoAP.
	CoapIsTcp() bool

	// Stops a receive operation in progress.  This must be called from a
	// separate thread, as sesn receive operations are blocking.
	AbortRx(nmpSeq uint8) error

	// XXX AbortResource(seq uint8) error

	RxAccept() (Sesn, *SesnCfg, error)
	RxCoap(opt TxOptions) (coap.Message, error)

	// Performs a blocking transmit a single management request (NMP / OMP) and
	// listens for the response.
	//     * nil: success.
	//     * nmxutil.SesnClosedError: session not open.
	//     * other error
	TxRxMgmt(m *nmp.NmpMsg, timeout time.Duration) (nmp.NmpRsp, error)
	TxRxMgmtAsync(m *nmp.NmpMsg, timeout time.Duration, ch chan nmp.NmpRsp, errc chan error) error

	// Creates a listener for incoming CoAP messages matching the specified
	// criteria.
	ListenCoap(mc nmcoap.MsgCriteria) (*nmcoap.Listener, error)

	// Cancels the CoAP listener with the specified criteria.
	StopListenCoap(mc nmcoap.MsgCriteria)

	// Transmits a CoAP message.
	TxCoap(m coap.Message) error

	// Returns a transmit and a receive callback used to manipulate CoAP
	// messages
	Filters() (nmcoap.TxMsgFilter, nmcoap.RxMsgFilter)

	// Sets the transmit and a receive callback used to manipulate CoAP
	// messages
	SetFilters(txFilter nmcoap.TxMsgFilter, rxFilter nmcoap.RxMsgFilter)
}
