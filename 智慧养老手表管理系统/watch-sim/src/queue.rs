//! 离线队列。
//!
//! 语义对齐后端与 HarmonyOS 端 `Protocol.ets::enqueue`：
//! 队列**不跨绑定版本混传**——绑定一变，旧记录不能再补传，只能留待核查。
//! 容量对齐后端公布的 `maxBatchSize`（200）。

use std::collections::VecDeque;

use crate::protocol::{HealthSample, MAX_BATCH_SIZE};

#[derive(Debug)]
pub struct OfflineQueue {
    capacity: usize,
    items: VecDeque<HealthSample>,
    /// 入队时锁定的绑定版本；与当前版本不一致即视为陈旧队列。
    epoch: Option<i64>,
    dropped: u64,
}

impl Default for OfflineQueue {
    fn default() -> Self {
        Self::new(MAX_BATCH_SIZE)
    }
}

impl OfflineQueue {
    pub fn new(capacity: usize) -> Self {
        OfflineQueue {
            capacity: capacity.max(1),
            items: VecDeque::with_capacity(capacity.max(1)),
            epoch: None,
            dropped: 0,
        }
    }

    pub fn len(&self) -> usize {
        self.items.len()
    }

    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }

    pub fn epoch(&self) -> Option<i64> {
        self.epoch
    }

    /// 累计因容量溢出被丢弃的条数——用于在面板上显示"真的丢了"。
    pub fn dropped(&self) -> u64 {
        self.dropped
    }

    /// 入队。队列满时丢最旧的一条并计数，而不是静默拒绝。
    pub fn push(&mut self, sample: HealthSample, epoch: i64) {
        if self.epoch.is_none() {
            self.epoch = Some(epoch);
        }
        if self.items.len() >= self.capacity {
            self.items.pop_front();
            self.dropped += 1;
        }
        self.items.push_back(sample);
    }

    /// 取最多 `max` 条待上传样本（不出队，失败时无需回填）。
    pub fn peek(&self, max: usize) -> Vec<HealthSample> {
        self.items
            .iter()
            .take(max.min(self.capacity))
            .cloned()
            .collect()
    }

    /// 上传成功后按条数出队。后端对重复 `recordId` 返回 duplicates，
    /// 因此"确认成功"只需按已发送条数推进，不必比对内容。
    pub fn commit(&mut self, n: usize) {
        for _ in 0..n.min(self.items.len()) {
            self.items.pop_front();
        }
        if self.items.is_empty() {
            self.epoch = None;
        }
    }

    pub fn clear(&mut self) {
        self.items.clear();
        self.epoch = None;
    }

    /// 绑定版本变化时清空队列，返回作废条数。
    pub fn invalidate_if_epoch_changed(&mut self, current: i64) -> usize {
        match self.epoch {
            Some(e) if e != current => {
                let n = self.items.len();
                self.clear();
                n
            }
            _ => 0,
        }
    }

    /// 队列内容是否属于旧绑定（决定重连后能不能补传）。
    pub fn is_stale_for(&self, current: i64) -> bool {
        matches!(self.epoch, Some(e) if e != current)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample(id: &str) -> HealthSample {
        HealthSample::heart_rate(id.into(), "W1", 72, "2026-11-01T00:00:00Z")
    }

    #[test]
    fn overflow_drops_oldest_and_counts() {
        let mut q = OfflineQueue::new(2);
        q.push(sample("a"), 1);
        q.push(sample("b"), 1);
        q.push(sample("c"), 1);
        assert_eq!(q.len(), 2);
        assert_eq!(q.dropped(), 1);
        assert_eq!(q.peek(10)[0].record_id, "b");
    }

    #[test]
    fn peek_then_commit_preserves_order() {
        let mut q = OfflineQueue::default();
        for i in 0..5 {
            q.push(sample(&format!("r{i}")), 1);
        }
        let batch = q.peek(3);
        assert_eq!(
            batch
                .iter()
                .map(|s| s.record_id.as_str())
                .collect::<Vec<_>>(),
            ["r0", "r1", "r2"]
        );
        // 失败：不出队，内容不变。
        assert_eq!(q.len(), 5);
        q.commit(3);
        assert_eq!(q.len(), 2);
        assert_eq!(q.peek(1)[0].record_id, "r3");
    }

    #[test]
    fn epoch_change_invalidates_queue() {
        let mut q = OfflineQueue::default();
        q.push(sample("a"), 7);
        assert!(q.is_stale_for(8));
        assert_eq!(q.invalidate_if_epoch_changed(8), 1);
        assert!(q.is_empty());
        assert_eq!(q.epoch(), None);
    }

    #[test]
    fn same_epoch_survives_reconnect() {
        let mut q = OfflineQueue::default();
        q.push(sample("a"), 7);
        assert!(!q.is_stale_for(7));
        assert_eq!(q.invalidate_if_epoch_changed(7), 0);
        assert_eq!(q.len(), 1);
    }

    #[test]
    fn commit_more_than_present_is_clamped() {
        let mut q = OfflineQueue::default();
        q.push(sample("a"), 1);
        q.commit(99);
        assert!(q.is_empty());
    }
}
