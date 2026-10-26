// Package idgen 紧凑 snowflake ID 生成器（预留）：
// 41bit 毫秒时间戳 | 10bit 节点 | 12bit 序列，单节点每毫秒 4096 个。
package util

import (
	"fmt"
	"sync"
	"time"
)

const (
	nodeBits   = 10
	seqBits    = 12
	maxNodeID  = -1 ^ (-1 << nodeBits) // 1023
	maxSeq     = -1 ^ (-1 << seqBits)  // 4095
	epochMilli = 1735689600000         // 2025-01-01 00:00:00 UTC，可沿用约 69 年
)

// Node snowflake 节点。
type Node struct {
	mu   sync.Mutex
	node int64
	last int64 // 上次生成毫秒
	seq  int64
}

var SnowflakeNode *Node

// NewNode 构造节点；nodeID ∈ [0, 1023]。
func NewNode(nodeID int64) error {
	if nodeID < 0 || nodeID > maxNodeID {
		return fmt.Errorf("idgen: node id %d out of range [0, %d]", nodeID, maxNodeID)
	}
	SnowflakeNode = &Node{node: nodeID}
	return nil
}

// Next 生成下一个 ID。
func (n *Node) Next() int64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now().UnixMilli() - epochMilli
	if now == n.last {
		n.seq = (n.seq + 1) & maxSeq
		if n.seq == 0 {
			// 序列耗尽，自旋到下一毫秒。
			for now <= n.last {
				now = time.Now().UnixMilli() - epochMilli
			}
		}
	} else {
		n.seq = 0
	}
	n.last = now
	return now<<(nodeBits+seqBits) | n.node<<seqBits | n.seq
}
