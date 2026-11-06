//! 设备会话：把协议、队列与传输编排成"一台手表能做的全部动作"。
//!
//! 覆盖 preview 后端实际存在的六个设备侧接口：
//! `login` / `bind` / `heartbeat` / `report` / `healthBatch` / `sos`。

use std::sync::Arc;

use serde_json::json;

use crate::encoding::BatchEncoder;
use crate::protocol::{
    login_body, BatchRequest, BatchView, Credentials, HealthSample, HealthSampleData, Platform,
    Reply, ReportPayload, SosRequest, MAX_BATCH_SIZE, PROTOCOL_VERSION,
};
use crate::queue::OfflineQueue;
use crate::transport::{Client, TransportError};

#[derive(Debug, Clone, PartialEq)]
pub struct Binding {
    pub old_id: i64,
    pub old_name: String,
    /// 绑定版本（后端 `device.health_epoch`）。重绑后会变。
    pub version: i64,
}

#[derive(Debug)]
pub enum Failure {
    Transport(TransportError),
    Rejected { code: String, message: String },
}

impl Failure {
    pub fn message(&self) -> String {
        match self {
            Failure::Transport(e) => e.message(),
            Failure::Rejected { code, message } => format!("[{code}] {message}"),
        }
    }

    pub fn retryable(&self) -> bool {
        match self {
            Failure::Transport(e) => e.retryable(),
            Failure::Rejected { .. } => false,
        }
    }

    pub fn is_auth(&self) -> bool {
        matches!(self, Failure::Rejected { code, .. } if code == "DEVICE_AUTH_FAILED" || code == "DEVICE_DISABLED")
    }

    pub fn is_unbound(&self) -> bool {
        matches!(self, Failure::Rejected { code, .. } if code == "DEVICE_NOT_BOUND")
    }
}

fn classify(reply: Reply) -> Result<Reply, Failure> {
    if reply.ok() {
        Ok(reply)
    } else {
        Err(Failure::Rejected {
            code: reply.code(),
            message: reply.message(),
        })
    }
}

fn confirm_batch_reply(reply: Reply, count: usize) -> Result<Reply, Failure> {
    if reply.get_i64("accepted") == Some(count as i64)
        && reply
            .get_i64("duplicates")
            .is_some_and(|duplicates| duplicates >= 0 && duplicates <= count as i64)
    {
        Ok(reply)
    } else {
        Err(Failure::Rejected {
            code: "INVALID_RESPONSE".into(),
            message: "批次回执未确认全部样本或重复计数无效".into(),
        })
    }
}

#[derive(Debug, Default, Clone)]
pub struct Stats {
    pub requests: u64,
    pub ok: u64,
    pub rejected: u64,
    pub failed: u64,
    pub queued_in: u64,
    pub queued_flushed: u64,
    pub duplicates: u64,
    pub incidents: u64,
    pub stale_dropped: u64,
}

#[derive(Debug, Clone)]
pub struct ReportOutcome {
    pub rate_state: Option<i64>,
    pub pre_state: Option<i64>,
    pub queued: bool,
    pub server_time: String,
    /// 报告是否已由服务端确认重复；普通 report 不按文案推断成功。
    pub duplicate_request: bool,
}

#[derive(Debug, Clone)]
pub struct HeartbeatOutcome {
    pub report_interval: u64,
    pub pending_alarm: i64,
    pub bound: bool,
    /// 本次心跳是否发现历元变化（重绑）。上层据此清掉 SOS 会话号等旧归属状态。
    pub epoch_changed: bool,
}

pub struct DeviceSession {
    cred: Credentials,
    client: Arc<Client>,
    signer: Option<Box<crate::protocol::PreparedSigner>>,
    platform: Platform,
    app_version: String,
    binding: Option<Binding>,
    queue: OfflineQueue,
    wire: Vec<u8>,
    encoder: BatchEncoder,
    pub stats: Stats,
    pub max_batch_size: usize,
}

impl DeviceSession {
    pub fn new(
        cred: Credentials,
        client: Arc<Client>,
        platform: Platform,
        app_version: String,
    ) -> Self {
        let signer = Some(Box::new(crate::protocol::PreparedSigner::new(&cred)));
        DeviceSession {
            cred,
            client,
            signer,
            platform,
            app_version,
            binding: None,
            queue: OfflineQueue::default(),
            wire: Vec::with_capacity(1024),
            encoder: BatchEncoder::default(),
            stats: Stats::default(),
            max_batch_size: MAX_BATCH_SIZE,
        }
    }

    pub fn credentials(&self) -> &Credentials {
        &self.cred
    }

    fn post_raw(&self, path: &str, raw: &[u8]) -> Result<Reply, TransportError> {
        if let Some(signer) = self.signer.as_deref() {
            self.client.post_raw_prepared(path, raw, signer)
        } else {
            self.client.post_raw(path, raw, None)
        }
    }

    fn post_json(&self, path: &str, body: &serde_json::Value) -> Result<Reply, TransportError> {
        let raw = serde_json::to_vec(body).map_err(|e| TransportError::Decode(e.to_string()))?;
        self.post_raw(path, &raw)
    }

    pub fn binding(&self) -> Option<&Binding> {
        self.binding.as_ref()
    }

    pub fn queue(&self) -> &OfflineQueue {
        &self.queue
    }

    pub fn queue_mut(&mut self) -> &mut OfflineQueue {
        &mut self.queue
    }

    pub fn is_bound(&self) -> bool {
        self.binding.is_some()
    }

    /// 更新绑定身份；旧绑定队列保留，flush 会拒绝跨版本补传。
    fn sync_binding(&mut self, version: i64, old_id: i64, old_name: String) -> bool {
        let changed = self
            .binding
            .as_ref()
            .map(|b| b.version != version)
            .unwrap_or(false);
        self.binding = Some(Binding {
            old_id,
            old_name,
            version,
        });
        changed
    }

    fn touch(&mut self, result: &Result<Reply, Failure>) {
        self.stats.requests += 1;
        match result {
            Ok(_) => self.stats.ok += 1,
            Err(Failure::Rejected { .. }) => self.stats.rejected += 1,
            Err(Failure::Transport(_)) => self.stats.failed += 1,
        }
    }

    /// 登录。声明 HarmonyOS 平台时会核验后端是否真的支持协议 1 与 200 条批量。
    pub fn login(&mut self) -> Result<Reply, Failure> {
        let body = login_body(&self.cred, self.platform);
        let result = self
            .post_json("api/v1/device/login", &body)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);

        if let Ok(reply) = &result {
            if self.platform == Platform::HarmonyOs {
                let version = reply.get_i64("protocolVersion");
                let max_batch = reply.get_i64("maxBatchSize");
                if version != Some(PROTOCOL_VERSION) || max_batch != Some(MAX_BATCH_SIZE as i64) {
                    return Err(Failure::Rejected {
                        code: "PROTOCOL".into(),
                        message: "后端未协商出协议 1 / 200 条批量，请先确认后端版本".into(),
                    });
                }
                if let Some(n) = max_batch {
                    self.max_batch_size = n as usize;
                }
            }
            let version = reply.binding_version().unwrap_or(0);
            if reply.bound() && version > 0 {
                let old_id = reply.get_i64("oldId").unwrap_or(0);
                let old_name = reply.get_str("oldPersonName").unwrap_or_default();
                if old_id <= 0 {
                    return Err(Failure::Rejected {
                        code: "INVALID_RESPONSE".into(),
                        message: "登录缺少有效长者身份".into(),
                    });
                }
                self.sync_binding(version, old_id, old_name);
            } else {
                self.binding = None;
            }
        }
        result
    }

    pub fn bind(&mut self, code: &str) -> Result<Reply, Failure> {
        let body = json!({
            "deviceNo": self.cred.device_no,
            "secret": self.cred.secret,
            "code": code.trim().to_uppercase(),
        });
        let result = self
            .post_json("api/v1/device/bind", &body)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);
        // 绑定成功后必须重新登录才能拿到新的 bindingVersion。
        if result.is_ok() {
            self.login()?;
        }
        result
    }

    pub fn heartbeat(&mut self, battery: i64) -> Result<HeartbeatOutcome, Failure> {
        let body = json!({
            "deviceNo": self.cred.device_no,
            "secret": self.cred.secret,
            "battery": battery,
            "appVersion": self.app_version,
        });
        let result = self
            .post_json("api/v1/device/heartbeat", &body)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);
        let reply = result?;
        let previous = self.binding.clone();
        if !reply.bound() {
            self.binding = None;
        } else if let (Some(v), Some(id)) = (
            reply.binding_version().filter(|v| *v > 0),
            reply.get_i64("oldId").filter(|v| *v > 0),
        ) {
            let name = previous
                .as_ref()
                .filter(|b| b.old_id == id)
                .map(|b| b.old_name.clone())
                .unwrap_or_default();
            self.sync_binding(v, id, name);
        } else {
            return Err(Failure::Rejected {
                code: "INVALID_RESPONSE".into(),
                message: "心跳缺少绑定身份".into(),
            });
        }
        let epoch_changed = previous != self.binding;
        Ok(HeartbeatOutcome {
            // 后端下发上报节奏，旧版直接丢掉了。
            report_interval: reply.get_i64("reportInterval").unwrap_or(5).max(1) as u64,
            pending_alarm: reply.get_i64("pendingAlarm").unwrap_or(0),
            bound: reply.bound() || reply.get_i64("bound") == Some(1),
            epoch_changed,
        })
    }

    /// 旧版扁平遥测。带 `requestId` 且满足后端条件时会走持久队列分支。
    pub fn report(&mut self, payload: &ReportPayload) -> Result<ReportOutcome, Failure> {
        payload.validate().map_err(|m| Failure::Rejected {
            code: "CLIENT".into(),
            message: m,
        })?;
        self.wire.clear();
        serde_json::to_writer(&mut self.wire, payload).map_err(|e| Failure::Rejected {
            code: "CLIENT".into(),
            message: format!("负载序列化失败：{e}"),
        })?;
        let result = self
            .post_raw("api/v1/device/report", &self.wire)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);
        let reply = result?;
        Ok(ReportOutcome {
            rate_state: reply.get_i64("rateState"),
            pre_state: reply.get_i64("preState"),
            queued: reply.get_bool("queued").unwrap_or(false),
            server_time: reply.get_str("serverTime").unwrap_or_default(),
            duplicate_request: false,
        })
    }

    /// 把样本放进离线队列。
    pub fn enqueue(
        &mut self,
        samples: impl IntoIterator<Item = HealthSample>,
    ) -> Result<(), Failure> {
        let epoch = self.binding.as_ref().map(|b| b.version).unwrap_or(0);
        let samples: Vec<_> = samples.into_iter().collect();
        if self.queue.is_stale_for(epoch) || self.queue.len() + samples.len() > MAX_BATCH_SIZE {
            return Err(Failure::Rejected {
                code: "QUEUE_BLOCKED".into(),
                message: "待传队列已满或属于旧绑定，原记录保留；本拍未加入队列".into(),
            });
        }
        self.stats.queued_in += samples.len() as u64;
        for sample in samples {
            self.queue.push(sample, epoch);
        }
        Ok(())
    }

    /// 上传离线队列。绑定版本不匹配的一律不传（后端也会 409）。
    pub fn flush(&mut self) -> Result<(usize, usize), Failure> {
        let Some(binding) = self.binding.clone() else {
            return Err(Failure::Rejected {
                code: "DEVICE_NOT_BOUND".into(),
                message: "设备尚未绑定长者".into(),
            });
        };
        if self.queue.is_stale_for(binding.version) {
            return Err(Failure::Rejected {
                code: "CONFLICT".into(),
                message: "待传记录属于旧绑定，已停止补传；记录仍保留在本进程队列中".into(),
            });
        }
        if self.queue.is_empty() {
            return Ok((0, 0));
        }
        let batch = self.queue.peek(self.max_batch_size);
        let count = batch.len();
        let request = BatchRequest::new(&self.cred, binding.version, batch);
        request.validate().map_err(|m| Failure::Rejected {
            code: "CLIENT".into(),
            message: m,
        })?;
        let raw = serde_json::to_vec(&request).map_err(|e| Failure::Rejected {
            code: "CLIENT".into(),
            message: format!("批次序列化失败：{e}"),
        })?;
        let result = self
            .post_raw("api/v1/device/healthBatch", &raw)
            .map_err(Failure::Transport)
            .and_then(classify)
            .and_then(|reply| confirm_batch_reply(reply, count));
        self.touch(&result);
        let reply = result?;
        let duplicates = reply.get_i64("duplicates").unwrap_or(0);
        if duplicates > 0 {
            self.stats.duplicates += duplicates as u64;
        }
        // 完整回执确认后才出队；失败时保留原 recordId 供显式重试。
        self.queue.commit(count);
        self.stats.queued_flushed += count as u64;
        Ok((count, 0))
    }

    /// 直接把一批样本发出去，不经离线队列。
    ///
    /// 压测与批量演示用：队列是为"断网补偿"设计的，走队列会额外压一次
    /// VecDeque 的入队/出队和容量裁剪，对纯吞吐测量是干扰。
    pub fn send_batch(&mut self, samples: Vec<HealthSample>) -> Result<(usize, usize), Failure> {
        self.send_samples(&samples)
    }

    /// 直接序列化借用的设备、时间和批次字段，避免按样本复制公共字符串。
    pub fn send_samples<S: serde::Serialize>(
        &mut self,
        samples: &[HealthSampleData<S>],
    ) -> Result<(usize, usize), Failure> {
        let count = samples.len();
        let binding_version = self.batch_binding(count)?;
        let request = BatchView {
            device_no: &self.cred.device_no,
            secret: &self.cred.secret,
            binding_version,
            samples,
        };
        self.wire.clear();
        self.wire.reserve(count * 400 + 200);
        serde_json::to_writer(&mut self.wire, &request).map_err(|e| Failure::Rejected {
            code: "CLIENT".into(),
            message: format!("批次序列化失败：{e}"),
        })?;
        self.send_batch_buffer(count)
    }

    pub fn send_sample_views<R: serde::Serialize>(
        &mut self,
        samples: &[HealthSampleData<&str, R>],
    ) -> Result<(usize, usize), Failure> {
        let count = samples.len();
        let binding_version = self.batch_binding(count)?;
        self.wire.clear();
        self.wire.reserve(count * 400 + 200);
        self.encoder
            .write(&mut self.wire, &self.cred, binding_version, samples)
            .map_err(|error| Failure::Rejected {
                code: "CLIENT".into(),
                message: format!("批次序列化失败：{error}"),
            })?;
        self.send_batch_buffer(count)
    }

    fn batch_binding(&self, count: usize) -> Result<i64, Failure> {
        let binding_version =
            self.binding
                .as_ref()
                .map(|b| b.version)
                .ok_or_else(|| Failure::Rejected {
                    code: "DEVICE_NOT_BOUND".into(),
                    message: "设备尚未绑定长者".into(),
                })?;
        if count == 0 || count > MAX_BATCH_SIZE || binding_version < 1 {
            return Err(Failure::Rejected {
                code: "CLIENT".into(),
                message: "批次需要有效绑定版本和 1–200 条样本".into(),
            });
        }
        Ok(binding_version)
    }

    fn send_batch_buffer(&mut self, count: usize) -> Result<(usize, usize), Failure> {
        let result = self
            .post_raw("api/v1/device/healthBatch", &self.wire)
            .map_err(Failure::Transport)
            .and_then(classify)
            .and_then(|reply| confirm_batch_reply(reply, count));
        self.touch(&result);
        let reply = result?;
        let duplicates = reply.get_i64("duplicates").unwrap_or(0);
        if duplicates > 0 {
            self.stats.duplicates += duplicates as u64;
        }
        self.stats.queued_flushed += count as u64;
        Ok((count, duplicates as usize))
    }

    /// 发送已经序列化好的批次字节。流水线模式要自己拼 HTTP，用这个走"只发一次
    /// 请求"的路径，避免重复序列化。
    pub fn send_batch_raw(&mut self, raw: &[u8]) -> Result<Reply, Failure> {
        let result = self
            .post_raw("api/v1/device/healthBatch", raw)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);
        result
    }

    pub fn sos(
        &mut self,
        request_id: &str,
        lng: Option<f64>,
        lat: Option<f64>,
    ) -> Result<Reply, Failure> {
        let binding = self.binding.clone().ok_or_else(|| Failure::Rejected {
            code: "DEVICE_NOT_BOUND".into(),
            message: "设备尚未绑定长者，SOS 需要绑定版本".into(),
        })?;
        let request = SosRequest {
            position_source: "simulated",
            device_no: self.cred.device_no.clone(),
            secret: self.cred.secret.clone(),
            request_id: request_id.to_string(),
            binding_version: binding.version,
            lng: lng.map(|v| format!("{v:.6}")),
            lat: lat.map(|v| format!("{v:.6}")),
        };
        request.validate().map_err(|m| Failure::Rejected {
            code: "CLIENT".into(),
            message: m,
        })?;
        let raw = serde_json::to_vec(&request).map_err(|e| Failure::Rejected {
            code: "CLIENT".into(),
            message: format!("SOS 序列化失败：{e}"),
        })?;
        let result = self
            .post_raw("api/v1/device/sos", &raw)
            .map_err(Failure::Transport)
            .and_then(classify);
        self.touch(&result);
        if let Ok(reply) = &result {
            if reply.get_bool("duplicate").unwrap_or(false) {
                // 弱网重试同一 requestId：后端回了同一个 incident。
                self.stats.duplicates += 1;
            } else {
                self.stats.incidents += 1;
            }
        }
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::time::Duration;

    fn serve(bodies: Vec<&'static str>) -> String {
        serve_status(200, bodies)
    }

    fn serve_status(status: u16, bodies: Vec<&'static str>) -> String {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let addr = format!("http://{}", listener.local_addr().unwrap());
        std::thread::spawn(move || {
            for body in bodies {
                let Ok((mut stream, _)) = listener.accept() else {
                    return;
                };
                let mut buf = [0u8; 8192];
                let _ = stream.read(&mut buf);
                // 必须显式 Connection: close 并 shutdown，否则 ureq 会尝试复用
                // 这条已经结束的连接，第二次请求直接 RST。
                let response = format!(
                    "HTTP/1.1 {status} Response\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                    body.as_bytes().len(),
                    body
                );
                let _ = stream.write_all(response.as_bytes());
                let _ = stream.flush();
                let _ = stream.shutdown(std::net::Shutdown::Write);
                // 读干剩余输入再关闭：带未读数据 close 会发 RST，客户端会看到
                // os error 10053，表现为用例随机失败。
                let _ = stream.set_read_timeout(Some(std::time::Duration::from_millis(200)));
                let mut sink = [0u8; 1024];
                loop {
                    match stream.read(&mut sink) {
                        Ok(0) | Err(_) => break,
                        Ok(_) => continue,
                    }
                }
            }
        });
        addr
    }

    fn session(addr: &str) -> DeviceSession {
        let client = Arc::new(Client::new(addr, Duration::from_secs(5), false));
        let s = DeviceSession::new(
            Credentials::new("WATCH0001", "0123456789abcdef0123456789abcdef"),
            client,
            Platform::Legacy,
            "2.0.0-sim".into(),
        );
        s
    }

    #[test]
    fn heartbeat_retries_transient_http_status_without_accepting_bad_replies() {
        for status in [500, 502, 503, 504] {
            for body in ["", "<html>unavailable</html>", r#"{"returnCode":"0000"}"#] {
                let address = serve_status(status, vec![body]);
                let failure = session(&address).heartbeat(80).unwrap_err();
                assert!(
                    failure.retryable(),
                    "status={status} body={body:?}: {failure:?}"
                );
            }
        }
        let address = serve_status(200, vec!["not JSON"]);
        assert!(!session(&address).heartbeat(80).unwrap_err().retryable());
        for status in [200, 401] {
            let address = serve_status(
                status,
                vec![r#"{"returnCode":"0000","code":"DEVICE_AUTH_FAILED"}"#],
            );
            let failure = session(&address).heartbeat(80).unwrap_err();
            assert!(failure.is_auth());
            assert!(!failure.retryable());
        }
    }

    #[test]
    fn queued_batch_keeps_records_until_acknowledged() {
        for body in [
            r#"{"returnCode":"1111","accepted":0,"duplicates":0}"#,
            r#"{"returnCode":"1111","accepted":2,"duplicates":0}"#,
            r#"{"returnCode":"1111","accepted":1,"duplicates":2}"#,
            r#"{"returnCode":"1111","accepted":1,"duplicates":-1}"#,
            r#"{"returnCode":"1111","accepted":1}"#,
            r#"{"returnCode":"1111","accepted":1.5,"duplicates":0}"#,
            r#"{"returnCode":"1111"}"#,
        ] {
            let address = serve(vec![
                r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
                body,
                r#"{"returnCode":"1111","accepted":1,"duplicates":1}"#,
            ]);
            let mut session = session(&address);
            session.login().unwrap();
            session
                .enqueue([HealthSample::heart_rate(
                    "queued-ack".into(),
                    "WATCH0001",
                    70,
                    "2026-11-03T00:00:00Z",
                )])
                .unwrap();
            let original = serde_json::to_vec(&session.queue().peek(200)).unwrap();
            assert!(session.flush().is_err(), "invalid acknowledgement: {body}");
            assert_eq!(
                serde_json::to_vec(&session.queue().peek(200)).unwrap(),
                original
            );
            assert_eq!(session.queue().epoch(), Some(1));
            assert_eq!(session.stats.queued_flushed, 0);
            assert_eq!(session.stats.duplicates, 0);
            assert_eq!(session.stats.rejected, 1);
            assert_eq!(session.flush().unwrap(), (1, 0));
            assert!(session.queue().is_empty());
            assert_eq!(session.stats.queued_flushed, 1);
            assert_eq!(session.stats.duplicates, 1);
        }
    }

    #[test]
    fn batch_ack_must_confirm_sent_samples() {
        for body in [
            r#"{"returnCode":"1111","accepted":0,"duplicates":0}"#,
            r#"{"returnCode":"1111","accepted":1,"duplicates":2}"#,
            r#"{"returnCode":"1111","accepted":1,"duplicates":-1}"#,
            r#"{"returnCode":"1111"}"#,
        ] {
            let address = serve(vec![
                r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
                body,
            ]);
            let mut session = session(&address);
            session.login().unwrap();
            let samples = [crate::protocol::SampleView::heart_rate(
                "ack-1".into(),
                "WATCH0001",
                70,
                "2026-11-02T00:00:00Z",
            )];
            assert!(
                session.send_sample_views(&samples).is_err(),
                "invalid acknowledgement was accepted: {body}"
            );
            assert_eq!(session.stats.queued_flushed, 0);
            assert_eq!(session.stats.rejected, 1);
        }
    }

    #[test]
    fn login_binds_and_remembers_epoch() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","deviceId":9,"bound":1,"oldId":42,"oldPersonName":"李桂芳","bindingVersion":3}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        let b = s.binding().expect("应当记录绑定");
        assert_eq!(b.old_id, 42);
        assert_eq!(b.version, 3);
        assert_eq!(b.old_name, "李桂芳");
    }

    #[test]
    fn login_rejection_is_classified() {
        let addr = serve(vec![
            r#"{"returnCode":"0000","code":"DEVICE_AUTH_FAILED","message":"设备号或密钥不正确"}"#,
        ]);
        let mut s = session(&addr);
        let err = s.login().unwrap_err();
        assert!(err.is_auth(), "应识别为鉴权失败: {err:?}");
        assert_eq!(s.stats.rejected, 1);
    }

    #[test]
    fn heartbeat_keeps_server_timing_hints() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1,"reportInterval":9,"pendingAlarm":1}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        let hb = s.heartbeat(77).unwrap();
        assert_eq!(hb.report_interval, 9);
        assert_eq!(hb.pending_alarm, 1);
        assert!(hb.bound);
    }

    #[test]
    fn rebind_epoch_change_preserves_stale_queue() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":2}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        s.enqueue([HealthSample::heart_rate(
            "r1".into(),
            "W",
            70,
            "2026-11-01T00:00:00Z",
        )])
        .unwrap();
        assert_eq!(s.queue().len(), 1);
        s.login().unwrap();
        assert_eq!(s.queue().len(), 1, "旧绑定记录应保留");
        assert_eq!(s.stats.stale_dropped, 0);
    }

    #[test]
    fn heartbeat_epoch_change_flags_and_preserves_queue() {
        // 心跳是设备运行期唯一会周期性拿到历元的入口；重绑必须在这里被发现，
        // 否则旧队列会继续补传到新老人名下。
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":2}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        s.enqueue([HealthSample::heart_rate(
            "r1".into(),
            "W",
            70,
            "2026-11-01T00:00:00Z",
        )])
        .unwrap();
        assert_eq!(s.queue().len(), 1);
        let hb = s.heartbeat(90).unwrap();
        assert!(hb.epoch_changed, "心跳必须报告历元变化");
        assert_eq!(s.queue().len(), 1, "旧绑定记录应保留");
        assert_eq!(s.stats.stale_dropped, 0);
        assert_eq!(s.binding().map(|b| b.version), Some(2));
    }

    #[test]
    fn heartbeat_without_epoch_change_reports_false() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        let hb = s.heartbeat(90).unwrap();
        assert!(!hb.epoch_changed, "历元未变不应误报");
    }

    #[test]
    fn duplicate_message_does_not_override_failure() {
        let addr = serve(vec![
            r#"{"returnCode":"0000","code":"CONFLICT","message":"编号已存在"}"#,
        ]);
        let mut s = session(&addr);
        let mut p = ReportPayload::new(s.credentials(), "2026-11-01 12:00:00".into());
        p.step = Some(1200);
        assert!(s.report(&p).is_err());
        assert_eq!(s.stats.rejected, 1);
    }

    #[test]
    fn flush_refuses_without_binding() {
        let addr = serve(vec![]);
        let mut s = session(&addr);
        s.enqueue([HealthSample::heart_rate(
            "r1".into(),
            "W",
            70,
            "2026-11-01T00:00:00Z",
        )])
        .unwrap();
        let err = s.flush().unwrap_err();
        assert!(err.is_unbound());
    }

    #[test]
    fn flush_sends_batch_and_commits() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","accepted":2,"duplicates":0}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        s.enqueue([
            HealthSample::heart_rate("r1-hr".into(), "W", 70, "2026-11-01T00:00:00Z"),
            HealthSample::steps(
                "r1-steps".into(),
                "W",
                1200,
                "2026-11-01T00:00:00Z",
                "2026-11-01T08:00:00Z",
            ),
        ])
        .unwrap();
        let (sent, stale) = s.flush().unwrap();
        assert_eq!((sent, stale), (2, 0));
        assert!(s.queue().is_empty());
    }

    #[test]
    fn report_rejects_client_side_before_network() {
        let addr = serve(vec![]);
        let mut s = session(&addr);
        let cred = s.credentials().clone();
        let mut payload = ReportPayload::new(&cred, "2026-11-01 12:00:00".into());
        payload.hr = Some(999);
        let err = s.report(&payload).unwrap_err();
        assert!(matches!(err, Failure::Rejected { .. }));
        assert_eq!(s.stats.requests, 0, "本地校验失败不应产生请求");
    }

    #[test]
    fn sos_marks_duplicate_retry() {
        let addr = serve(vec![
            r#"{"returnCode":"1111","bound":1,"oldId":1,"bindingVersion":1}"#,
            r#"{"returnCode":"1111","incidentId":7,"incidentState":"received","duplicate":true}"#,
        ]);
        let mut s = session(&addr);
        s.login().unwrap();
        let reply = s.sos("sos-1", Some(113.0625), Some(28.1975)).unwrap();
        assert_eq!(reply.get_i64("incidentId"), Some(7));
        assert_eq!(s.stats.duplicates, 1);
        assert_eq!(s.stats.incidents, 0);
    }
}
