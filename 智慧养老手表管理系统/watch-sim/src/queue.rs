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
