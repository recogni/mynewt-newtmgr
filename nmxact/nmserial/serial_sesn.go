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

package nmserial

import (
	"fmt"
	"sync"
	"time"

	"github.com/runtimeco/go-coap"

	"github.com/recogni/newtmgr/nmxact/mgmt"
	"github.com/recogni/newtmgr/nmxact/nmcoap"
	"github.com/recogni/newtmgr/nmxact/nmp"
	"github.com/recogni/newtmgr/nmxact/nmxutil"
	"github.com/recogni/newtmgr/nmxact/omp"
	"github.com/recogni/newtmgr/nmxact/sesn"
)

type SerialSesn struct {
	cfg    sesn.SesnCfg
	sx     *SerialXport
	txvr   *mgmt.Transceiver
	isOpen bool

	// This mutex ensures:
	//     * accesses to isOpen are protected.
	m  sync.Mutex
	wg sync.WaitGroup

	errChan  chan error
	msgChan  chan []byte
	connChan chan *SerialSesn
	stopChan chan struct{}
}

func NewSerialSesn(sx *SerialXport, cfg sesn.SesnCfg) (*SerialSesn, error) {
	s := &SerialSesn{
		cfg: cfg,
		sx:  sx,
	}

	txvr, err := mgmt.NewTransceiver(cfg.TxFilter, cfg.RxFilter, false,
		cfg.MgmtProto, 3)
	if err != nil {
		return nil, err
	}
	s.txvr = txvr

	return s, nil
}

func (s *SerialSesn) Open() error {
	s.m.Lock()

	if s.isOpen {
		s.m.Unlock()
		return nmxutil.NewSesnAlreadyOpenError(
			"Attempt to open an already-open serial session")
	}

	txvr, err := mgmt.NewTransceiver(s.cfg.TxFilter, s.cfg.RxFilter, false,
		s.cfg.MgmtProto, 3)
	if err != nil {
		s.m.Unlock()
		return err
	}
	s.txvr = txvr
	s.errChan = make(chan error)
	s.msgChan = make(chan []byte, 16)
	s.connChan = make(chan *SerialSesn, 4)
	s.stopChan = make(chan struct{})

	s.isOpen = true
	s.m.Unlock()
	if s.cfg.MgmtProto == sesn.MGMT_PROTO_COAP_SERVER {
		return nil
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			select {
			case msg, ok := <-s.msgChan:
				if !ok {
					continue
				}
				if s.cfg.MgmtProto == sesn.MGMT_PROTO_OMP {
					s.txvr.DispatchCoap(msg)
				} else if s.cfg.MgmtProto == sesn.MGMT_PROTO_NMP {
					s.txvr.DispatchNmpRsp(msg)
				}
			case <-s.errChan:
				// XXX pass it on
			case <-s.stopChan:
				return
			}
		}
	}()
	return nil
}

func (s *SerialSesn) Close() error {
	s.m.Lock()

	if !s.isOpen {
		s.m.Unlock()
		return nmxutil.NewSesnClosedError(
			"Attempt to close an unopened serial session")
	}

	s.isOpen = false
	s.txvr.ErrorAll(fmt.Errorf("closed"))
	s.txvr.Stop()
	close(s.stopChan)
	close(s.connChan)
	s.sx.Lock()
	if s == s.sx.acceptSesn {
		s.sx.acceptSesn = nil

	}
	if s == s.sx.reqSesn {
		s.sx.reqSesn = nil
	}
	s.sx.Unlock()
	s.m.Unlock()

	s.wg.Wait()
	s.stopChan = nil
	s.txvr = nil

	for {
		s, ok := <-s.connChan
		if !ok {
			break
		}
		s.Close()
	}
	close(s.msgChan)
	for {
		if _, ok := <-s.msgChan; !ok {
			break
		}
	}
	close(s.errChan)
	for {
		if _, ok := <-s.errChan; !ok {
			break
		}
	}

	return nil
}

func (s *SerialSesn) IsOpen() bool {
	s.m.Lock()
	defer s.m.Unlock()

	return s.isOpen
}

func (s *SerialSesn) MtuIn() int {
	return 1024 - omp.OMP_MSG_OVERHEAD
}

func (s *SerialSesn) MtuOut() int {
	// Mynewt commands have a default chunk buffer size of 512.  Account for
	// base64 encoding.
	return s.sx.cfg.Mtu*3/4 - omp.OMP_MSG_OVERHEAD
}

func (s *SerialSesn) AbortRx(seq uint8) error {
	s.txvr.ErrorAll(fmt.Errorf("Rx aborted"))
	return nil
}

func (s *SerialSesn) TxRxMgmt(m *nmp.NmpMsg,
	timeout time.Duration) (nmp.NmpRsp, error) {

	if !s.isOpen {
		return nil, nmxutil.NewSesnClosedError(
			"Attempt to transmit over closed serial session")
	}

	txFn := func(b []byte) error {
		return s.sx.Tx(b)
	}

	err := s.sx.setRspSesn(s)
	if err != nil {
		return nil, err
	}
	defer s.sx.setRspSesn(nil)

	return s.txvr.TxRxMgmt(txFn, m, s.MtuOut(), timeout)
}

func (s *SerialSesn) TxRxMgmtAsync(m *nmp.NmpMsg,
	timeout time.Duration, ch chan nmp.NmpRsp, errc chan error) error {

	rsp, err := s.TxRxMgmt(m, timeout)
	if err != nil {
		errc <- err
	} else {
		ch <- rsp
	}
	return nil
}

func (s *SerialSesn) TxCoap(m coap.Message) error {
	if !s.isOpen {
		return nmxutil.NewSesnClosedError(
			"attempt to transmit over closed serial session")
	}

	err := s.sx.setRspSesn(s)
	if err != nil {
		return err
	}

	return s.txvr.TxCoap(s.sx.Tx, m, s.MtuOut())
}

func (s *SerialSesn) ListenCoap(
	mc nmcoap.MsgCriteria) (*nmcoap.Listener, error) {

	return s.txvr.ListenCoap(mc)
}

func (s *SerialSesn) StopListenCoap(mc nmcoap.MsgCriteria) {
	s.txvr.StopListenCoap(mc)
}

func (s *SerialSesn) MgmtProto() sesn.MgmtProto {
	return s.cfg.MgmtProto
}

func (s *SerialSesn) CoapIsTcp() bool {
	return false
}

func (s *SerialSesn) RxAccept() (sesn.Sesn, *sesn.SesnCfg, error) {
	if !s.isOpen {
		return nil, nil, nmxutil.NewSesnClosedError(
			"Attempt to listen for data from closed connection")
	}
	if s.cfg.MgmtProto != sesn.MGMT_PROTO_COAP_SERVER {
		return nil, nil, fmt.Errorf("Invalid operation for %s", s.cfg.MgmtProto)
	}

	err := s.sx.setAcceptSesn(s)
	if err != nil {
		return nil, nil, err
	}

	s.wg.Add(1)
	defer s.wg.Done()
	for {
		select {
		case cl_s, ok := <-s.connChan:
			if !ok {
				continue
			}
			return cl_s, &cl_s.cfg, nil
		case <-s.stopChan:
			return nil, nil, fmt.Errorf("Session closed")
		}
	}
}

func (s *SerialSesn) RxCoap(opt sesn.TxOptions) (coap.Message, error) {
	if !s.isOpen {
		return nil, nmxutil.NewSesnClosedError(
			"Attempt to listen for data from closed connection")
	}
	if s.cfg.MgmtProto != sesn.MGMT_PROTO_COAP_SERVER {
		return nil, fmt.Errorf("Invalid operation for %s", s.cfg.MgmtProto)
	}
	if s.sx.reqSesn != s {
		return nil, fmt.Errorf("Invalid operation")
	}
	waitTmoChan := time.After(opt.Timeout)
	s.wg.Add(1)
	defer s.wg.Done()
	for {
		select {
		case data, ok := <-s.msgChan:
			if !ok {
				continue
			}
			msg, err := s.txvr.ProcessCoapReq(data)
			if err != nil {
				return nil, err
			}
			if msg != nil {
				return msg, nil
			}
		case _, ok := <-waitTmoChan:
			waitTmoChan = nil
			if ok {
				return nil, nmxutil.NewRspTimeoutError(
					"RxCoap() timed out")
			}
		case err, ok := <-s.errChan:
			if !ok {
				continue
			}
			if err == errTimeout {
				continue
			}
			return nil, err
		case <-s.stopChan:
			return nil, fmt.Errorf("Session closed")
		}
	}
}

func (s *SerialSesn) Filters() (nmcoap.TxMsgFilter, nmcoap.RxMsgFilter) {
	return s.txvr.Filters()
}

func (s *SerialSesn) SetFilters(txFilter nmcoap.TxMsgFilter,
	rxFilter nmcoap.RxMsgFilter) {

	s.txvr.SetFilters(txFilter, rxFilter)
}
