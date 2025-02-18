// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"encoding/hex"

	ntypes "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

// Query_GetEndDelayTime query delay tx end delay time
func (n *None) Query_GetEndDelayTime(req *types.ReqBytes) (types.Message, error) {

	if len(req.GetData()) == 0 {
		return nil, types.ErrInvalidParam
	}

	val, err := n.GetStateDB().Get(formatDelayTxKey(req.GetData()))
	if err != nil {
		eLog.Error("Query_GetEndDelayTime", "txHash", hex.EncodeToString(req.GetData()), "get db err", err)
		return nil, types.ErrGetStateDB
	}
	info := &ntypes.CommitDelayTxLog{}

	err = types.Decode(val, info)
	if err != nil {
		eLog.Error("Query_GetEndDelayTime", "txHash", hex.EncodeToString(req.GetData()), "get db err", err)
		return nil, types.ErrDecode
	}
	return &types.Int64{Data: info.EndDelayTime}, nil
}
