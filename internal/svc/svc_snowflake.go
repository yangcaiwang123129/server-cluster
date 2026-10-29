package svc

import (
	"fmt"
	"sync"
	"time"
)

// 雪花 ID 生成器：41bit 毫秒时间戳 | 10bit 节点 | 12bit 序列，单节点每毫秒 4096 个。
const (
	nodeBits   = 10
	seqBits    = 12
	maxNodeID  = -1 ^ (-1 << nodeBits) // 1023
	maxSeq     = -1 ^ (-1 << seqBits)  // 4095
	epochMilli = 1735689600000         // 2025-01-01 00:00:00 UTC，可沿用约 69 年
)

// Snowflake 雪花 ID 生成节点。
type Snowflake struct {
	mu   sync.Mutex
	node int64
	last int64 // 上次生成毫秒
	seq  int64
}

// newSnowflake 构造节点；nodeID ∈ [0, 1023]。
func newSnowflake(nodeID int64) (*Snowflake, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, fmt.Errorf("svc: snowflake node id %d out of range [0, %d]", nodeID, maxNodeID)
	}
	return &Snowflake{node: nodeID}, nil
}

// Next 生成下一个 ID。
func (this *Snowflake) Next() int64 {
	this.mu.Lock()
	defer this.mu.Unlock()
	now := time.Now().UnixMilli() - epochMilli
	if now == this.last {
		this.seq = (this.seq + 1) & maxSeq
		if this.seq == 0 {
			// 序列耗尽，自旋到下一毫秒。
			for now <= this.last {
				now = time.Now().UnixMilli() - epochMilli
			}
		}
	} else {
		this.seq = 0
	}
	this.last = now
	return now<<(nodeBits+seqBits) | this.node<<seqBits | this.seq
}
