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
	"fmt"

	log "github.com/sirupsen/logrus"

	. "github.com/recogni/newtmgr/nmxact/bledefs"
	"github.com/recogni/newtmgr/nmxact/nmxutil"
	"github.com/recogni/newtmgr/nmxact/sesn"
)

type AdvertiseCfg struct {
	// Mandatory
	OwnAddrType   BleAddrType
	ConnMode      BleAdvConnMode
	DiscMode      BleAdvDiscMode
	ItvlMin       uint16
	ItvlMax       uint16
	ChannelMap    uint8
	FilterPolicy  BleAdvFilterPolicy
	HighDutyCycle bool
	AdvFields     BleAdvFields
	RspFields     BleAdvFields
	SesnCfg       sesn.SesnCfg

	// Only required for direct advertisements
	PeerAddr *BleAddr
}

func NewAdvertiseCfg() AdvertiseCfg {
	return AdvertiseCfg{
		OwnAddrType: BLE_ADDR_TYPE_RANDOM,
		ConnMode:    BLE_ADV_CONN_MODE_UND,
		DiscMode:    BLE_ADV_DISC_MODE_GEN,
	}
}

type Advertiser struct {
	bx          *BleXport
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func NewAdvertiser(bx *BleXport) *Advertiser {
	return &Advertiser{
		bx: bx,
	}
}

func (a *Advertiser) fields(f BleAdvFields) ([]byte, error) {
	r := BleAdvFieldsToReq(f)

	bl, err := a.bx.AddListener(SeqKey(r.Seq))
	if err != nil {
		return nil, err
	}
	defer a.bx.RemoveListener(bl)

	return advFields(a.bx, bl, r)
}

func (a *Advertiser) setAdvData(data []byte) error {
	r := NewBleAdvSetDataReq()
	r.Data = BleBytes{data}

	bl, err := a.bx.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer a.bx.RemoveListener(bl)

	if err := advSetData(a.bx, bl, r); err != nil {
		return err
	}

	return nil
}

func (a *Advertiser) setRspData(data []byte) error {
	r := NewBleAdvRspSetDataReq()
	r.Data = BleBytes{data}

	bl, err := a.bx.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer a.bx.RemoveListener(bl)

	if err := advRspSetData(a.bx, bl, r); err != nil {
		return err
	}

	return nil
}

func (a *Advertiser) advertise(cfg AdvertiseCfg) (uint16, *Listener, error) {
	r := NewBleAdvStartReq()

	r.OwnAddrType = cfg.OwnAddrType
	r.DurationMs = 0x7fffffff
	r.ConnMode = cfg.ConnMode
	r.DiscMode = cfg.DiscMode
	r.ItvlMin = cfg.ItvlMin
	r.ItvlMax = cfg.ItvlMax
	r.ChannelMap = cfg.ChannelMap
	r.FilterPolicy = cfg.FilterPolicy
	r.HighDutyCycle = cfg.HighDutyCycle
	r.PeerAddr = cfg.PeerAddr

	bl, err := a.bx.AddListener(SeqKey(r.Seq))
	if err != nil {
		return 0, nil, err
	}

	connHandle, err := advStart(a.bx, bl, a.stopChan, r)
	if err != nil {
		a.bx.RemoveListener(bl)
		if !nmxutil.IsXport(err) {
			// The transport did not restart; always attempt to cancel the
			// advertise operation.  In some cases, the host has already stopped
			// advertising and will respond with an "ealready" error that can be
			// ignored.
			if err := a.stopAdvertising(); err != nil {
				log.Debugf("Failed to cancel advertise in progress: %s",
					err.Error())
			}
		}
		return 0, nil, err
	}

	return connHandle, bl, nil
}

func (a *Advertiser) stopAdvertising() error {
	r := NewBleAdvStopReq()

	bl, err := a.bx.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer a.bx.RemoveListener(bl)

	return advStop(a.bx, bl, r)
}

func (a *Advertiser) buildSesn(cfg AdvertiseCfg, connHandle uint16,
	bl *Listener) (sesn.Sesn, error) {

	s, err := NewBleSesn(a.bx, cfg.SesnCfg)
	if err != nil {
		return nil, err
	}

	if err := s.OpenConnected(connHandle, bl); err != nil {
		return nil, err
	}

	return s, nil
}

func (a *Advertiser) Start(cfg AdvertiseCfg) (sesn.Sesn, error) {
	var advData []byte
	var rspData []byte
	var connHandle uint16
	var bl *Listener
	var err error

	fns := []func() error{
		// Convert advertising fields to data.
		func() error {
			advData, err = a.fields(cfg.AdvFields)
			return err
		},

		// Set advertising data.
		func() error {
			return a.setAdvData(advData)
		},

		// Convert response fields to data.
		func() error {
			rspData, err = a.fields(cfg.RspFields)
			return err
		},

		// Set response data.
		func() error {
			return a.setRspData(rspData)
		},

		// Advertise
		func() error {
			connHandle, bl, err = a.advertise(cfg)
			return err
		},
	}

	a.stopChan = make(chan struct{})
	a.stoppedChan = make(chan struct{})

	defer func() {
		a.stopChan = nil
		close(a.stoppedChan)
	}()

	if err := a.bx.AcquireSlave(a); err != nil {
		return nil, err
	}
	defer a.bx.ReleaseSlave()

	for _, fn := range fns {
		// Check for abort before each step.
		select {
		case <-a.stopChan:
			return nil, fmt.Errorf("advertise aborted")
		default:
		}

		if err := fn(); err != nil {
			return nil, err
		}
	}

	return a.buildSesn(cfg, connHandle, bl)
}

func (a *Advertiser) Stop() error {
	stopChan := a.stopChan
	if stopChan == nil {
		return fmt.Errorf("advertiser already stopped")
	}
	close(stopChan)

	a.bx.StopWaitingForSlave(a, fmt.Errorf("advertise aborted"))
	a.stopAdvertising()

	// Block until abort is complete.
	<-a.stoppedChan

	return nil
}
