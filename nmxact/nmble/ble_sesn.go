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

package nmble

import (
	"time"

	"github.com/runtimeco/go-coap"

	. "github.com/recogni/newtmgr/nmxact/bledefs"
	"github.com/recogni/newtmgr/nmxact/nmcoap"
	"github.com/recogni/newtmgr/nmxact/nmp"
	"github.com/recogni/newtmgr/nmxact/sesn"
)

type BleSesn struct {
	cfg sesn.SesnCfg
	bx  *BleXport
	Ns  *NakedSesn
}

func NewBleSesn(bx *BleXport, cfg sesn.SesnCfg) (
	*BleSesn, error) {

	Ns, err := NewNakedSesn(bx, cfg)
	if err != nil {
		return nil, err
	}

	s := &BleSesn{
		cfg: cfg,
		bx:  bx,
		Ns:  Ns,
	}

	return s, nil
}

func (s *BleSesn) AbortRx(seq uint8) error {
	return s.Ns.AbortRx(seq)
}

func (s *BleSesn) Open() error {
	if err := s.bx.AcquireMasterPrimary(s); err != nil {
		return err
	}
	defer s.bx.ReleaseMaster()

	return s.Ns.Open()
}

func (s *BleSesn) OpenConnected(
	connHandle uint16, eventListener *Listener) error {

	return s.Ns.OpenConnected(connHandle, eventListener)
}

func (s *BleSesn) Close() error {
	return s.Ns.Close()
}

func (s *BleSesn) IsOpen() bool {
	return s.Ns.IsOpen()
}

func (s *BleSesn) MtuIn() int {
	return s.Ns.MtuIn()
}

func (s *BleSesn) MtuOut() int {
	return s.Ns.MtuOut()
}

func (s *BleSesn) CoapIsTcp() bool {
	return s.Ns.CoapIsTcp()
}

func (s *BleSesn) MgmtProto() sesn.MgmtProto {
	return s.Ns.MgmtProto()
}

func (s *BleSesn) ConnInfo() (BleConnDesc, error) {
	return s.Ns.ConnInfo()
}

func (s *BleSesn) SetOobKey(key []byte) {
	s.Ns.SetOobKey(key)
}

func (s *BleSesn) TxRxMgmt(m *nmp.NmpMsg,
	timeout time.Duration) (nmp.NmpRsp, error) {

	return s.Ns.TxRxMgmt(m, timeout)
}

func (s *BleSesn) TxRxMgmtAsync(m *nmp.NmpMsg,
	timeout time.Duration, ch chan nmp.NmpRsp, errc chan error) error {

	return s.Ns.TxRxMgmtAsync(m, timeout, ch, errc)
}

func (s *BleSesn) TxCoap(m coap.Message) error {
	return s.Ns.TxCoap(m)
}

func (s *BleSesn) ListenCoap(mc nmcoap.MsgCriteria) (*nmcoap.Listener, error) {
	return s.Ns.ListenCoap(mc)
}

func (s *BleSesn) StopListenCoap(mc nmcoap.MsgCriteria) {
	s.Ns.StopListenCoap(mc)
}

func (s *BleSesn) RxAccept() (sesn.Sesn, *sesn.SesnCfg, error) {
	return s.Ns.RxAccept()
}

func (s *BleSesn) RxCoap(opt sesn.TxOptions) (coap.Message, error) {
	return s.Ns.RxCoap(opt)
}

func (s *BleSesn) Filters() (nmcoap.TxMsgFilter, nmcoap.RxMsgFilter) {
	return s.Ns.Filters()
}

func (s *BleSesn) SetFilters(txFilter nmcoap.TxMsgFilter,
	rxFilter nmcoap.RxMsgFilter) {

	s.Ns.SetFilters(txFilter, rxFilter)
}
