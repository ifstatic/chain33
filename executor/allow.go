package executor

import (
	"bytes"

	"gitlab.33.cn/chain33/chain33/types"
)

func isAllowExec(key, realExecer []byte, tx *types.Transaction, height int64) bool {
	keyExecer, err := types.FindExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err)
		return false
	}
	//平行链中 user.p.guodun.xxxx -> 实际上是 xxxx
	//TODO: 后面 GetDriverName(), 中驱动的名字可以被配置
	if types.IsPara() && bytes.Equal(keyExecer, realExecer) {
		return true
	}
	//其他合约可以修改自己合约内部(执行器只能修改执行器自己内部的数据)
	if bytes.Equal(keyExecer, tx.Execer) {
		return true
	}
	//每个合约中，都会开辟一个区域，这个区域是另外一个合约可以修改的区域
	//我们把数据限制在这个位置，防止合约的其他位置被另外一个合约修改
	//  execaddr 是加了前缀生成的地址， 而参数 realExecer 是没有前缀的执行器名字
	keyExecAddr, ok := types.GetExecKey(key)
	if ok && keyExecAddr == drivers.ExecAddress(string(tx.Execer)) {
		return true
	}
	// 历史原因做只针对对bityuan的fork特殊化处理一下
	// manage 的key 是 config
	// token 的部分key 是 mavl-create-token-
	if !types.IsMatchFork(height, types.ForkV13ExecKey) {
		if bytes.Equal(realExecer, types.ExecerManage) && bytes.Equal(keyExecer, types.ExecerConfig) {
			return true
		}
		if bytes.Equal(realExecer, types.ExecerToken) {
			if bytes.HasPrefix(key, []byte("mavl-create-token-")) {
				return true
			}
		}
	}
	//分成两种情况:
	//是执行器余额，判断 friend
	execdriver := keyExecer
	if ok && keyExecAddr == drivers.ExecAddress(string(realExecer)) {
		//判断user.p.xxx.token 是否可以写 token 合约的内容之类的
		execdriver = realExecer
	}
	d, err := drivers.LoadDriver(string(execdriver), height)
	if err != nil {
		elog.Error("load drivers error", "err", err)
		return false
	}
	//交给 -> friend 来判定
	return d.IsFriend(execdriver, key, tx)
}
