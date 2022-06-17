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
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	. "github.com/recogni/newtmgr/nmxact/bledefs"
	"github.com/recogni/newtmgr/nmxact/nmxutil"
	"github.com/recogni/newtmgr/nmxact/sesn"
)

const WRITE_CMD_BASE_SZ = 3
const NOTIFY_CMD_BASE_SZ = 3

var nextSeq BleSeq = BLE_SEQ_MIN
var seqMtx sync.Mutex

func NextSeq() BleSeq {
	seqMtx.Lock()
	defer seqMtx.Unlock()

	seq := nextSeq
	nextSeq++
	if nextSeq >= BLE_SEQ_EVT_MIN {
		nextSeq = BLE_SEQ_MIN
	}

	return seq
}

func BhdTimeoutError(rspType MsgType, seq BleSeq) error {
	str := fmt.Sprintf(
		"Timeout waiting for blehostd to send %s response (seq=%d)",
		MsgTypeToString(rspType), seq)

	log.Debug(str)

	return nmxutil.NewXportError(str)
}

func StatusError(op MsgOp, msgType MsgType, status int) error {
	str := fmt.Sprintf("%s %s indicates error: %s (%d)",
		MsgOpToString(op),
		MsgTypeToString(msgType),
		ErrCodeToString(status),
		status)

	log.Debug(str)
	return nmxutil.NewBleHostError(status, str)
}

func BleDescFromConnFindRsp(r *BleConnFindRsp) BleConnDesc {
	return BleConnDesc{
		ConnHandle:      r.ConnHandle,
		OwnIdAddrType:   r.OwnIdAddrType,
		OwnIdAddr:       r.OwnIdAddr,
		OwnOtaAddrType:  r.OwnOtaAddrType,
		OwnOtaAddr:      r.OwnOtaAddr,
		PeerIdAddrType:  r.PeerIdAddrType,
		PeerIdAddr:      r.PeerIdAddr,
		PeerOtaAddrType: r.PeerOtaAddrType,
		PeerOtaAddr:     r.PeerOtaAddr,
		Role:            r.Role,
		Encrypted:       r.Encrypted,
		Authenticated:   r.Authenticated,
		Bonded:          r.Bonded,
		KeySize:         r.KeySize,
	}
}

func BleAdvReportFromScanEvt(e *BleScanEvt) BleAdvReport {
	return BleAdvReport{
		EventType: e.EventType,
		Sender: BleDev{
			AddrType: e.AddrType,
			Addr:     e.Addr,
		},
		Rssi: e.Rssi,

		Fields: BleAdvFields{
			Data: e.Data.Bytes,

			Flags:              e.DataFlags,
			Uuids16:            e.DataUuids16,
			Uuids16IsComplete:  e.DataUuids16IsComplete,
			Uuids32:            e.DataUuids32,
			Uuids32IsComplete:  e.DataUuids32IsComplete,
			Uuids128:           e.DataUuids128,
			Uuids128IsComplete: e.DataUuids128IsComplete,
			Name:               e.DataName,
			NameIsComplete:     e.DataNameIsComplete,
			TxPwrLvl:           e.DataTxPwrLvl,
			SlaveItvlMin:       e.DataSlaveItvlMin,
			SlaveItvlMax:       e.DataSlaveItvlMax,
			SvcDataUuid16:      e.DataSvcDataUuid16.Bytes,
			PublicTgtAddrs:     e.DataPublicTgtAddrs,
			Appearance:         e.DataAppearance,
			AdvItvl:            e.DataAdvItvl,
			SvcDataUuid32:      e.DataSvcDataUuid32.Bytes,
			SvcDataUuid128:     e.DataSvcDataUuid128.Bytes,
			Uri:                e.DataUri,
			MfgData:            e.DataMfgData.Bytes,
		},
	}
}

func NewBleConnectReq() *BleConnectReq {
	return &BleConnectReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_CONNECT,
		Seq:  NextSeq(),

		OwnAddrType:  BLE_ADDR_TYPE_PUBLIC,
		PeerAddrType: BLE_ADDR_TYPE_PUBLIC,
		PeerAddr:     BleAddr{},

		DurationMs:         30000,
		ScanItvl:           0x0010,
		ScanWindow:         0x0010,
		ItvlMin:            24,
		ItvlMax:            40,
		Latency:            0,
		SupervisionTimeout: 0x0200,
		MinCeLen:           0x0010,
		MaxCeLen:           0x0300,
	}
}

func NewBleTerminateReq() *BleTerminateReq {
	return &BleTerminateReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_TERMINATE,
		Seq:  NextSeq(),

		ConnHandle: 0,
		HciReason:  0,
	}
}

func NewBleConnCancelReq() *BleConnCancelReq {
	return &BleConnCancelReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_CONN_CANCEL,
		Seq:  NextSeq(),
	}
}

func NewBleDiscAllSvcsReq() *BleDiscAllSvcsReq {
	return &BleDiscAllSvcsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_DISC_ALL_SVCS,
		Seq:  NextSeq(),
	}
}

func NewBleDiscSvcUuidReq() *BleDiscSvcUuidReq {
	return &BleDiscSvcUuidReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_DISC_SVC_UUID,
		Seq:  NextSeq(),

		ConnHandle: 0,
		Uuid:       BleUuid{},
	}
}

func NewBleDiscAllChrsReq() *BleDiscAllChrsReq {
	return &BleDiscAllChrsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_DISC_ALL_CHRS,
		Seq:  NextSeq(),
	}
}

func NewBleDiscAllDscsReq() *BleDiscAllDscsReq {
	return &BleDiscAllDscsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_DISC_ALL_DSCS,
		Seq:  NextSeq(),
	}
}

func NewBleExchangeMtuReq() *BleExchangeMtuReq {
	return &BleExchangeMtuReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_EXCHANGE_MTU,
		Seq:  NextSeq(),

		ConnHandle: 0,
	}
}

func NewBleGenRandAddrReq() *BleGenRandAddrReq {
	return &BleGenRandAddrReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_GEN_RAND_ADDR,
		Seq:  NextSeq(),
	}
}

func NewBleSetRandAddrReq() *BleSetRandAddrReq {
	return &BleSetRandAddrReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SET_RAND_ADDR,
		Seq:  NextSeq(),
	}
}

func NewBleWriteCmdReq() *BleWriteCmdReq {
	return &BleWriteCmdReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_WRITE_CMD,
		Seq:  NextSeq(),
	}
}

func NewBleWriteReq() *BleWriteReq {
	return &BleWriteReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_WRITE,
		Seq:  NextSeq(),
	}
}

func NewBleScanReq() *BleScanReq {
	return &BleScanReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SCAN,
		Seq:  NextSeq(),
	}
}

func NewBleScanCancelReq() *BleScanCancelReq {
	return &BleScanCancelReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SCAN_CANCEL,
		Seq:  NextSeq(),
	}
}

func NewBleSetPreferredMtuReq() *BleSetPreferredMtuReq {
	return &BleSetPreferredMtuReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SET_PREFERRED_MTU,
		Seq:  NextSeq(),
	}
}

func NewBleConnFindReq() *BleConnFindReq {
	return &BleConnFindReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_CONN_FIND,
		Seq:  NextSeq(),
	}
}

func NewResetReq() *BleResetReq {
	return &BleResetReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_RESET,
		Seq:  NextSeq(),
	}
}

func NewBleSecurityInitiateReq() *BleSecurityInitiateReq {
	return &BleSecurityInitiateReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SECURITY_INITIATE,
		Seq:  NextSeq(),
	}
}

func NewBleAdvFieldsReq() *BleAdvFieldsReq {
	return &BleAdvFieldsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADV_FIELDS,
		Seq:  NextSeq(),
	}
}

func NewBleAdvSetDataReq() *BleAdvSetDataReq {
	return &BleAdvSetDataReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADV_SET_DATA,
		Seq:  NextSeq(),
	}
}

func NewBleAdvRspSetDataReq() *BleAdvRspSetDataReq {
	return &BleAdvRspSetDataReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADV_RSP_SET_DATA,
		Seq:  NextSeq(),
	}
}

func NewBleAdvStartReq() *BleAdvStartReq {
	return &BleAdvStartReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADV_START,
		Seq:  NextSeq(),
	}
}

func NewBleAdvStopReq() *BleAdvStopReq {
	return &BleAdvStopReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADV_STOP,
		Seq:  NextSeq(),
	}
}

func NewBleClearSvcsReq() *BleClearSvcsReq {
	return &BleClearSvcsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_CLEAR_SVCS,
		Seq:  NextSeq(),
	}
}

func NewBleAddSvcsReq() *BleAddSvcsReq {
	return &BleAddSvcsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ADD_SVCS,
		Seq:  NextSeq(),
	}
}

func NewBleCommitSvcsReq() *BleCommitSvcsReq {
	return &BleCommitSvcsReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_COMMIT_SVCS,
		Seq:  NextSeq(),
	}
}

func NewAccessStatusReq() *BleAccessStatusReq {
	return &BleAccessStatusReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_ACCESS_STATUS,
		Seq:  NextSeq(),
	}
}

func NewNotifyReq() *BleNotifyReq {
	return &BleNotifyReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_NOTIFY,
		Seq:  NextSeq(),
	}
}

func NewFindChrReq() *BleFindChrReq {
	return &BleFindChrReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_FIND_CHR,
		Seq:  NextSeq(),
	}
}

func NewSyncReq() *BleSyncReq {
	return &BleSyncReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SYNC,
		Seq:  NextSeq(),
	}
}

func NewBleSmInjectIoReq() *BleSmInjectIoReq {
	return &BleSmInjectIoReq{
		Op:   MSG_OP_REQ,
		Type: MSG_TYPE_SM_INJECT_IO,
		Seq:  NextSeq(),
	}
}

func ConnFindXact(x *BleXport, connHandle uint16) (BleConnDesc, error) {
	r := NewBleConnFindReq()
	r.ConnHandle = connHandle

	key := SeqKey(r.Seq)
	bl, err := x.AddListener(key)
	if err != nil {
		return BleConnDesc{}, err
	}
	defer x.RemoveListener(bl)

	return connFind(x, bl, r)
}

func GenRandAddrXact(x *BleXport) (BleAddr, error) {
	r := NewBleGenRandAddrReq()

	key := SeqKey(r.Seq)
	bl, err := x.AddListener(key)
	if err != nil {
		return BleAddr{}, err
	}
	defer x.RemoveListener(bl)

	return genRandAddr(x, bl, r)
}

func SetRandAddrXact(x *BleXport, addr BleAddr) error {
	r := NewBleSetRandAddrReq()
	r.Addr = addr

	key := SeqKey(r.Seq)
	bl, err := x.AddListener(key)
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return setRandAddr(x, bl, r)
}

func SetPreferredMtuXact(x *BleXport, mtu uint16) error {
	r := NewBleSetPreferredMtuReq()
	r.Mtu = mtu

	key := SeqKey(r.Seq)
	bl, err := x.AddListener(key)
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return setPreferredMtu(x, bl, r)
}

func ResetXact(x *BleXport) error {
	r := NewResetReq()

	key := SeqKey(r.Seq)
	bl, err := x.AddListener(key)
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return reset(x, bl, r)
}

func ClearSvcsXact(x *BleXport) error {
	r := NewBleClearSvcsReq()

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return clearSvcs(x, bl, r)
}

func AddSvcsXact(x *BleXport, svcs []BleAddSvc) error {
	r := NewBleAddSvcsReq()
	r.Svcs = svcs

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return addSvcs(x, bl, r)
}

func CommitSvcsXact(x *BleXport) ([]BleRegSvc, error) {
	r := NewBleCommitSvcsReq()

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return nil, err
	}
	defer x.RemoveListener(bl)

	return commitSvcs(x, bl, r)
}

func AccessStatusXact(x *BleXport, attStatus uint8, data []byte) error {
	r := NewAccessStatusReq()
	r.AttStatus = attStatus
	r.Data.Bytes = data

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return accessStatus(x, bl, r)
}

func NotifyXact(x *BleXport, connHandle uint16, attrHandle uint16,
	data []byte) error {

	r := NewNotifyReq()
	r.ConnHandle = connHandle
	r.AttrHandle = attrHandle
	r.Data.Bytes = data

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return err
	}
	defer x.RemoveListener(bl)

	return notify(x, bl, r)
}

func FindChrXact(x *BleXport, svcUuid BleUuid, chrUuid BleUuid) (
	uint16, uint16, error) {

	r := NewFindChrReq()
	r.SvcUuid = svcUuid
	r.ChrUuid = chrUuid

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return 0, 0, err
	}
	defer x.RemoveListener(bl)

	return findChr(x, bl, r)
}

func SyncXact(x *BleXport) (bool, error) {
	r := NewSyncReq()

	bl, err := x.AddListener(SeqKey(r.Seq))
	if err != nil {
		return false, err
	}

	synced, err := checkSync(x, bl, r)
	if err != nil {
		return false, err
	}

	return synced, nil
}

func DiscoverDeviceWithName(
	bx *BleXport,
	ownAddrType BleAddrType,
	timeout time.Duration,
	name string) (*BleDev, error) {

	advPred := func(r BleAdvReport) bool {
		return r.Fields.Name != nil && *r.Fields.Name == name
	}

	return DiscoverDevice(bx, ownAddrType, timeout, advPred)
}

func BleAdvFieldsToReq(f BleAdvFields) *BleAdvFieldsReq {
	r := NewBleAdvFieldsReq()

	r.Flags = f.Flags
	r.Uuids16 = f.Uuids16
	r.Uuids16IsComplete = f.Uuids16IsComplete
	r.Uuids32 = f.Uuids32
	r.Uuids32IsComplete = f.Uuids32IsComplete
	r.Uuids128 = f.Uuids128
	r.Uuids128IsComplete = f.Uuids128IsComplete
	r.Name = f.Name
	r.NameIsComplete = f.NameIsComplete
	r.TxPwrLvl = f.TxPwrLvl
	r.SlaveItvlMin = f.SlaveItvlMin
	r.SlaveItvlMax = f.SlaveItvlMax
	r.SvcDataUuid16 = BleBytes{f.SvcDataUuid16}
	r.PublicTgtAddrs = f.PublicTgtAddrs
	r.Appearance = f.Appearance
	r.AdvItvl = f.AdvItvl
	r.SvcDataUuid32 = BleBytes{f.SvcDataUuid32}
	r.SvcDataUuid128 = BleBytes{f.SvcDataUuid128}
	r.Uri = f.Uri
	r.MfgData = BleBytes{f.MfgData}

	return r
}

func BleSvcToAddSvc(svc BleSvc) BleAddSvc {
	as := BleAddSvc{
		Uuid:    svc.Uuid,
		SvcType: svc.SvcType,
	}

	for _, chr := range svc.Chrs {
		ac := BleAddChr{
			Uuid:       chr.Uuid,
			Flags:      chr.Flags,
			MinKeySize: chr.MinKeySize,
		}

		for _, dsc := range chr.Dscs {
			ad := BleAddDsc{
				Uuid:       dsc.Uuid,
				AttFlags:   dsc.AttFlags,
				MinKeySize: dsc.MinKeySize,
			}

			ac.Dscs = append(ac.Dscs, ad)
		}

		as.Chrs = append(as.Chrs, ac)
	}

	return as
}

func GapService(devName string) BleSvc {
	return BleSvc{
		Uuid:    BleUuid{0x1800, [16]byte{}},
		SvcType: BLE_SVC_TYPE_PRIMARY,
		Chrs: []BleChr{
			// Device name.
			BleChr{
				Uuid:       BleUuid{0x2a00, [16]byte{}},
				Flags:      BLE_GATT_F_READ,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte(devName)
				},
			},

			// Appearance.
			BleChr{
				Uuid:       BleUuid{0x2a01, [16]byte{}},
				Flags:      BLE_GATT_F_READ,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte{0, 0}
				},
			},

			// Peripheral privacy flag.
			BleChr{
				Uuid:       BleUuid{0x2a02, [16]byte{}},
				Flags:      BLE_GATT_F_READ,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte{0}
				},
			},

			// Reconnection address.
			BleChr{
				Uuid:       BleUuid{0x2a03, [16]byte{}},
				Flags:      BLE_GATT_F_READ,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte{0, 0, 0, 0, 0, 0}
				},
			},

			// Peripheral preferred connection parameters.
			BleChr{
				Uuid:       BleUuid{0x2a04, [16]byte{}},
				Flags:      BLE_GATT_F_READ,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte{0, 0, 0, 0, 0, 0, 0, 0}
				},
			},
		},
	}
}

func GattService() BleSvc {
	return BleSvc{
		Uuid:    BleUuid{0x1801, [16]byte{}},
		SvcType: BLE_SVC_TYPE_PRIMARY,
		Chrs: []BleChr{
			// Device name.
			BleChr{
				Uuid:       BleUuid{0x2a05, [16]byte{}},
				Flags:      BLE_GATT_F_INDICATE,
				MinKeySize: 0,
				AccessCb: func(access BleGattAccess) (uint8, []byte) {
					return 0, []byte{0, 0, 0, 0}
				},
			},
		},
	}
}

func BuildMgmtChrs(mgmtProto sesn.MgmtProto) (BleMgmtChrs, error) {
	mgmtChrs := BleMgmtChrs{}

	nmpSvcUuid, _ := ParseUuid(NmpPlainSvcUuid)
	nmpChrUuid, _ := ParseUuid(NmpPlainChrUuid)

	ompSvcUuid, _ := ParseUuid(OmpUnsecSvcUuid)
	ompReqChrUuid, _ := ParseUuid(OmpUnsecReqChrUuid)
	ompRspChrUuid, _ := ParseUuid(OmpUnsecRspChrUuid)

	resSvcUuid, _ := ParseUuid(IotivitySvcUuid)
	resReqChrUuid, _ := ParseUuid(IotivityReqChrUuid)
	resRspChrUuid, _ := ParseUuid(IotivityRspChrUuid)

	switch mgmtProto {
	case sesn.MGMT_PROTO_NMP:
		mgmtChrs.NmpReqChr = &BleChrId{nmpSvcUuid, nmpChrUuid}
		mgmtChrs.NmpRspChr = &BleChrId{nmpSvcUuid, nmpChrUuid}

	case sesn.MGMT_PROTO_OMP:
		mgmtChrs.NmpReqChr = &BleChrId{ompSvcUuid, ompReqChrUuid}
		mgmtChrs.NmpRspChr = &BleChrId{ompSvcUuid, ompRspChrUuid}

	default:
		return mgmtChrs,
			fmt.Errorf("invalid management protocol: %+v", mgmtProto)
	}

	mgmtChrs.ResReqChr = &BleChrId{resSvcUuid, resReqChrUuid}
	mgmtChrs.ResRspChr = &BleChrId{resSvcUuid, resRspChrUuid}

	return mgmtChrs, nil
}

func IsSecErr(err error) bool {
	bhdErr := nmxutil.ToBleHost(err)
	if bhdErr == nil {
		return false
	}

	switch bhdErr.Status - ERR_CODE_ATT_BASE {
	case ERR_CODE_ATT_INSUFFICIENT_AUTHEN,
		ERR_CODE_ATT_INSUFFICIENT_AUTHOR,
		ERR_CODE_ATT_INSUFFICIENT_KEY_SZ,
		ERR_CODE_ATT_INSUFFICIENT_ENC:

		return true

	default:
		return false
	}
}

// Attempts to convert the given error to a BLE security error.  The conversion
// succeeds if the error represents a pairing failure due to missing or
// mismatched key material.
func ToSecurityErr(err error) error {
	bhe := nmxutil.ToBleHost(err)
	if bhe == nil {
		return nil
	}

	code := ErrCodeToSmUs(bhe.Status)
	if code == -1 {
		code = ErrCodeToSmPeer(bhe.Status)
	}

	switch code {
	case ERR_CODE_SM_ERR_PASSKEY,
		ERR_CODE_SM_ERR_OOB,
		ERR_CODE_SM_ERR_CONFIRM_MISMATCH,
		ERR_CODE_SM_ERR_UNSPECIFIED,
		ERR_CODE_SM_ERR_DHKEY,
		ERR_CODE_SM_ERR_NUMCMP:
		return nmxutil.NewBleSecurityError(err.Error())

	default:
		return nil
	}
}
