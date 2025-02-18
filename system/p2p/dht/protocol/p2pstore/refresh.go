package p2pstore

import (
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/types"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

func (p *Protocol) refreshLocalChunk() {
	//优先处理同步任务
	if reply, err := p.API.IsSync(); err != nil || !reply.IsOk {
		return
	}
	start := time.Now()
	var saveNum, syncNum, deleteNum int
	for chunkNum := int64(0); ; chunkNum++ {
		records, err := p.getChunkRecordFromBlockchain(&types.ReqChunkRecords{Start: chunkNum, End: chunkNum})
		if err != nil || len(records.Infos) != 1 {
			log.Info("refreshLocalChunk", "break at chunk num", chunkNum, "error", err)
			break
		}
		info := records.Infos[0]
		msg := &types.ChunkInfoMsg{
			ChunkHash: info.ChunkHash,
			Start:     info.Start,
			End:       info.End,
		}
		if p.shouldSave(info.ChunkHash) {
			saveNum++
			// 本地处理
			// 1. p2pStore已保存数据则直接更新p2pStore
			// 2. p2pStore未保存数据则从blockchain获取数据
			if err := p.storeChunk(msg); err == nil {
				continue
			}

			// 通过网络从其他节点获取数据
			p.chunkInfoCacheMutex.Lock()
			p.chunkInfoCache[hex.EncodeToString(info.ChunkHash)] = msg
			p.chunkInfoCacheMutex.Unlock()
			syncNum++
			p.chunkToSync <- msg
			log.Info("refreshLocalChunk sync", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "chunkHash", hex.EncodeToString(info.ChunkHash), "start", info.Start)

		} else {
			deleteNum++
			p.chunkToDelete <- msg
			log.Info("refreshLocalChunk delete", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "chunkHash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
		}
	}
	log.Info("refreshLocalChunk", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "time cost", time.Since(start), "exRT size", p.getExtendRoutingTable().Size())
}

func (p *Protocol) shouldSave(chunkHash []byte) bool {
	if p.SubConfig.IsFullNode {
		return true
	}
	pids := p.getExtendRoutingTable().NearestPeers(genDHTID(chunkHash), 100)
	size := len(pids)
	if size == 0 {
		return true
	}
	if kbt.Closer(p.Host.ID(), pids[size*p.SubConfig.Percentage/100], genChunkNameSpaceKey(chunkHash)) {
		return true
	}
	return false
}

func (p *Protocol) getExtendRoutingTable() *kbt.RoutingTable {
	if p.extendRoutingTable.Size() < p.RoutingTable.Size() {
		return p.RoutingTable
	}
	return p.extendRoutingTable
}

func (p *Protocol) updateExtendRoutingTable() {
	key := []byte("temp")
	count := 200
	start := time.Now()
	for _, pid := range p.extendRoutingTable.ListPeers() {
		if p.RoutingTable.Find(pid) == "" {
			p.extendRoutingTable.RemovePeer(pid)
		}
	}
	peers := p.RoutingTable.ListPeers()
	for _, pid := range peers {
		_, _ = p.extendRoutingTable.TryAddPeer(pid, true, true)
	}
	if key != nil {
		peers = p.RoutingTable.NearestPeers(genDHTID(key), p.RoutingTable.Size())
	}

	for i, pid := range peers {
		// 至少从 3 个节点上获取新节点，
		if i+1 > 3 && p.extendRoutingTable.Size() > p.SubConfig.MaxExtendRoutingTableSize {
			break
		}
		extendPeers, err := p.fetchShardPeers(key, count, pid)
		if err != nil {
			log.Error("updateExtendRoutingTable", "fetchShardPeers error", err, "peer id", pid)
			continue
		}
		for _, cPid := range extendPeers {
			if cPid == p.Host.ID() {
				continue
			}
			_, _ = p.extendRoutingTable.TryAddPeer(cPid, true, true)
		}
	}
	log.Info("updateExtendRoutingTable", "pid", p.Host.ID(), "local peers count", p.RoutingTable.Size(), "extendRoutingTable peer count", p.extendRoutingTable.Size(), "time cost", time.Since(start), "origin count", count)
}
