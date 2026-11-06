//! 设备协议：负载类型、请求签名、返回包络。
//!
//! 字段上限与取值范围逐条对齐后端 `telemetry_validation.go`、`telemetry_parser.go`、
//! `position_metadata.go`、`handler_device_sos.go`。改动这里之前先读那四个文件。

use hmac::{Hmac, Mac};
use std::borrow::Cow;

use serde::{Deserialize, Serialize};
use serde_json::{json, Map, Value};
use sha2::{Digest, Sha256};

type HmacSha256 = Hmac<Sha256>;

/// 整数值按整数写，省掉 `.0` 两个字节。
fn ser_compact_number<S: serde::Serializer>(value: &f64, ser: S) -> Result<S::Ok, S::Error> {
    if value.fract() == 0.0 && value.abs() < 9.0e15 {
        ser.serialize_i64(*value as i64)
    } else {
        ser.serialize_f64(*value)
    }
}

// ---------------------------------------------------------------------------
// 凭据与登录
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Credentials {
    pub device_no: String,
    pub secret: String,
}

impl Credentials {
    pub fn new(device_no: impl Into<String>, secret: impl Into<String>) -> Self {
        Credentials {
            device_no: device_no.into().trim().to_uppercase(),
            secret: secret.into(),
        }
    }

    /// 后端 `checkDeviceCredentials`：deviceNo ≤32 字符、secret ≤128 字符，均非空。
    pub fn validate(&self) -> Result<(), String> {
        if self.device_no.is_empty() || self.device_no.chars().count() > 32 {
            return Err("设备号必须为 1–32 个字符".into());
        }
        if self.secret.is_empty() || self.secret.chars().count() > 128 {
            return Err("设备密钥必须为 1–128 个字符".into());
        }
        Ok(())
    }
}

/// 登录时声明的平台。`HarmonyOs` 会让后端回 `protocolVersion` 与 `maxBatchSize`。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Platform {
    /// 不传 platform，走旧合同。
    Legacy,
    /// platform=harmonyos + protocolVersion=1。
    HarmonyOs,
}

pub const PROTOCOL_VERSION: i64 = 1;
pub const MAX_BATCH_SIZE: usize = 200;

pub fn login_body(cred: &Credentials, platform: Platform) -> Value {
    let mut m = Map::new();
    m.insert("deviceNo".into(), json!(cred.device_no));
    m.insert("secret".into(), json!(cred.secret));
    if platform == Platform::HarmonyOs {
        m.insert("platform".into(), json!("harmonyos"));
        m.insert("protocolVersion".into(), json!(PROTOCOL_VERSION));
    }
    Value::Object(m)
}

// ---------------------------------------------------------------------------
// 返回包络
// ---------------------------------------------------------------------------

/// 后端统一包络：HTTP 200 也可能是业务失败，必须看 `returnCode`。
#[derive(Debug, Clone)]
pub struct Reply {
    pub status: u16,
    pub body: Value,
}

impl Reply {
    pub fn code(&self) -> String {
        if self.ok() {
            return crate::CODE_OK.into();
        }
        self.body
            .get("code")
            .and_then(Value::as_str)
            .unwrap_or("INVALID_RESPONSE")
            .into()
    }
    pub fn ok(&self) -> bool {
        self.status == 200
            && self.body.get("returnCode").and_then(Value::as_str) == Some(crate::CODE_OK)
    }
    pub fn message(&self) -> String {
        let message = self
            .body
            .get("message")
            .and_then(Value::as_str)
            .unwrap_or("请求未完成");
        match self.body.get("reference").and_then(Value::as_str) {
            Some(id) if !id.is_empty() => format!("{message}（编号：{id}）"),
            _ => message.into(),
        }
    }

    pub fn get_i64(&self, key: &str) -> Option<i64> {
        self.body.get(key).and_then(|v| match v {
            Value::Number(n) => n.as_i64(),
            Value::String(s) => s.parse().ok(),
            _ => None,
        })
    }

    pub fn get_str(&self, key: &str) -> Option<String> {
        self.body.get(key).and_then(|v| match v {
            Value::String(s) => Some(s.clone()),
            Value::Number(n) => Some(n.to_string()),
            _ => None,
        })
    }

    pub fn get_bool(&self, key: &str) -> Option<bool> {
        self.body.get(key).and_then(|v| match v {
            Value::Bool(b) => Some(*b),
            Value::Number(n) => Some(n.as_i64().unwrap_or(0) != 0),
            _ => None,
        })
    }

    /// 绑定版本（历元）。重绑后后端会返回新值，旧批次必须作废。
    pub fn binding_version(&self) -> Option<i64> {
        self.get_i64("bindingVersion")
    }

    pub fn bound(&self) -> bool {
        self.get_bool("bound").unwrap_or(false) || self.get_i64("bound") == Some(1)
    }
}

// ---------------------------------------------------------------------------
// 请求签名
// ---------------------------------------------------------------------------

/// 三个签名头。后端在 `CAREOLD_REQUIRE_SIGNED_DEVICE=1` 时强制要求，
/// 未设置时允许不带签名，但**一旦带上就必须验证通过**——所以要么整体不带，
/// 要么每个请求都重新生成 nonce 与时间戳。
#[derive(Debug, Clone)]
pub struct SignatureHeaders {
    pub signature: String,
    pub timestamp: String,
    pub nonce: String,
}

impl SignatureHeaders {
    pub fn pairs_ref(&self) -> [(&'static str, &str); 3] {
        [
            ("X-Device-Signature", &self.signature),
            ("X-Device-Timestamp", &self.timestamp),
            ("X-Device-Nonce", &self.nonce),
        ]
    }

    pub fn pairs(&self) -> [(&'static str, String); 3] {
        [
            ("X-Device-Signature", self.signature.clone()),
            ("X-Device-Timestamp", self.timestamp.clone()),
            ("X-Device-Nonce", self.nonce.clone()),
        ]
    }
}

/// 按后端 `prepareSignedDeviceRequest` 的公式计算签名：
/// `HMAC-SHA256(secret, deviceNo + ":" + ts + ":" + nonce + ":" + hex(sha256(body)))`。
pub fn sign(secret: &str, device_no: &str, timestamp: i64, nonce: &str, body: &[u8]) -> String {
    let body_hash = Sha256::digest(body);
    let device_no = device_no.trim();
    let normalized = if device_no.is_ascii() && !device_no.bytes().any(|c| c.is_ascii_lowercase()) {
        Cow::Borrowed(device_no)
    } else {
        Cow::Owned(device_no.to_uppercase())
    };
    let mut timestamp_buffer = itoa::Buffer::new();
    let timestamp = timestamp_buffer.format(timestamp);
    let mut body_hex = [0; 64];
    write_hex(&body_hash, &mut body_hex);
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes()).expect("HMAC 接受任意长度密钥");
    mac.update(normalized.as_bytes());
    mac.update(b":");
    mac.update(timestamp.as_bytes());
    mac.update(b":");
    mac.update(nonce.as_bytes());
    mac.update(b":");
    mac.update(&body_hex);
    hex_lower(&mac.finalize().into_bytes())
}

#[inline]
pub fn signature_headers(
    cred: &Credentials,
    body: &[u8],
    timestamp: i64,
    nonce: String,
) -> SignatureHeaders {
    SignatureHeaders {
        signature: sign(&cred.secret, &cred.device_no, timestamp, &nonce, body),
        timestamp: timestamp.to_string(),
        nonce,
    }
}

/// Immutable signing state for one device and secret; each request clones the HMAC state.
pub struct PreparedSigner {
    mac: HmacSha256,
}

pub(crate) trait RequestSigning: Send + Sync {
    fn headers(&self, body: &[u8], timestamp: i64, nonce: String) -> SignatureHeaders;
}

impl RequestSigning for Credentials {
    fn headers(&self, body: &[u8], timestamp: i64, nonce: String) -> SignatureHeaders {
        signature_headers(self, body, timestamp, nonce)
    }
}

impl RequestSigning for PreparedSigner {
    fn headers(&self, body: &[u8], timestamp: i64, nonce: String) -> SignatureHeaders {
        PreparedSigner::headers(self, body, timestamp, nonce)
    }
}

impl PreparedSigner {
    pub fn new(cred: &Credentials) -> Self {
        let mut mac =
            HmacSha256::new_from_slice(cred.secret.as_bytes()).expect("HMAC 接受任意长度密钥");
        mac.update(cred.device_no.trim().to_uppercase().as_bytes());
        mac.update(b":");
        Self { mac }
    }

    pub fn headers(&self, body: &[u8], timestamp: i64, nonce: String) -> SignatureHeaders {
        let mut timestamp_buffer = itoa::Buffer::new();
        let timestamp_text = timestamp_buffer.format(timestamp);
        let mut body_hex = [0; 64];
        write_hex(&Sha256::digest(body), &mut body_hex);
        let mut mac = self.mac.clone();
        mac.update(timestamp_text.as_bytes());
        mac.update(b":");
        mac.update(nonce.as_bytes());
        mac.update(b":");
        mac.update(&body_hex);
        SignatureHeaders {
            signature: hex_lower(&mac.finalize().into_bytes()),
            timestamp: timestamp_text.into(),
            nonce,
        }
    }
}

fn hex_lower(bytes: &[u8]) -> String {
    let mut output = vec![0; bytes.len() * 2];
    write_hex(bytes, &mut output);
    String::from_utf8(output).expect("十六进制字符均为 ASCII")
}

fn write_hex(bytes: &[u8], output: &mut [u8]) {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    for (pair, byte) in output.chunks_exact_mut(2).zip(bytes) {
        pair[0] = HEX[(byte >> 4) as usize];
        pair[1] = HEX[(byte & 0x0f) as usize];
    }
}

// ---------------------------------------------------------------------------
// 旧版扁平遥测（POST /api/v1/device/report）
// ---------------------------------------------------------------------------

/// 扁平遥测负载。字段可为空，但**一旦提供**就必须落在后端允许的范围内。
#[derive(Debug, Clone, Default, Serialize)]
pub struct ReportPayload {
    #[serde(rename = "deviceNo")]
    pub device_no: String,
    pub secret: String,
    #[serde(rename = "beTime", skip_serializing_if = "String::is_empty")]
    pub be_time: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub step: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sleep: Option<i64>,
    #[serde(rename = "sleepState", skip_serializing_if = "Option::is_none")]
    pub sleep_state: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hr: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bp: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lng: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lat: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub battery: Option<i64>,
    #[serde(rename = "appVersion", skip_serializing_if = "Option::is_none")]
    pub app_version: Option<String>,
    /// `"1"` 或 `"fall"`。
    #[serde(skip_serializing_if = "Option::is_none")]
    pub alarm: Option<String>,
    #[serde(rename = "positionSource", skip_serializing_if = "Option::is_none")]
    pub position_source: Option<String>,
    #[serde(rename = "positionSampledAt", skip_serializing_if = "Option::is_none")]
    pub position_sampled_at: Option<String>,
    #[serde(rename = "positionAccuracy", skip_serializing_if = "Option::is_none")]
    pub position_accuracy: Option<f64>,
    /// 带上它就进入后端持久队列分支（需服务端 `CAREOLD_DURABLE_TELEMETRY=1`）。
    #[serde(rename = "requestId", skip_serializing_if = "Option::is_none")]
    pub request_id: Option<String>,
    #[serde(rename = "bindingVersion", skip_serializing_if = "Option::is_none")]
    pub binding_version: Option<i64>,
}

impl ReportPayload {
    pub fn new(cred: &Credentials, be_time: String) -> Self {
        ReportPayload {
            device_no: cred.device_no.clone(),
            secret: cred.secret.clone(),
            be_time,
            ..Default::default()
        }
    }

    /// 本地预校验，把后端会 400 的情况提前拦下来，压测时不必浪费往返。
    pub fn validate(&self) -> Result<(), String> {
        if !self.be_time.is_empty() && self.be_time.len() != 19 {
            return Err(format!(
                "beTime 必须是 yyyy-MM-dd HH:mm:ss，实际 {:?}",
                self.be_time
            ));
        }
        check_metric("hr", self.hr, 20, 250)?;
        check_metric("bp", self.bp, 40, 300)?;
        check_metric("step", self.step, 0, 200_000)?;
        check_metric("sleep", self.sleep, 0, 24)?;
        check_metric("battery", self.battery, 0, 100)?;
        if let Some(s) = &self.sleep_state {
            if s != "0" && s != "1" {
                return Err("sleepState 应为 0 或 1".into());
            }
        }
        if let (Some(lng), Some(lat)) = (&self.lng, &self.lat) {
            if lng.chars().count() > 20 || lat.chars().count() > 20 {
                return Err("经纬度超过 20 字符".into());
            }
            let lo: f64 = lng.parse().map_err(|_| "lng 不是数字")?;
            let la: f64 = lat.parse().map_err(|_| "lat 不是数字")?;
            if !lo.is_finite() || !la.is_finite() || lo.abs() > 180.0 || la.abs() > 90.0 {
                return Err("经纬度超出有效范围".into());
            }
        } else if self.lng.is_some() || self.lat.is_some() {
            return Err("经纬度必须成对提供".into());
        }
        if let Some(src) = &self.position_source {
            const ALLOWED: [&str; 6] =
                ["gps", "network", "fused", "manual", "simulated", "unknown"];
            if !ALLOWED.contains(&src.as_str()) {
                return Err(format!("positionSource 非法: {src}"));
            }
        }
        if let Some(v) = &self.app_version {
            if v.chars().count() > 20 {
                return Err("appVersion 超过 20 字符".into());
            }
        }
        if let Some(v) = &self.source {
            if v.chars().count() > 20 {
                return Err("source 超过 20 字符".into());
            }
        }
        if let Some(a) = &self.alarm {
            if a != "1" && !a.eq_ignore_ascii_case("fall") {
                return Err("alarm 只能是 \"1\" 或 \"fall\"".into());
            }
        }
        if self.position_accuracy.is_some() && self.lng.is_none() {
            return Err("定位元数据必须与完整经纬度一起提供".into());
        }
        Ok(())
    }

    /// 是否够格进入后端持久队列（`durableTelemetryEligible` 的客户端镜像）。
    ///
    /// 后端判定是**排他白名单**：只要带了 `hr/bp/alarm/lng/lat/battery/appVersion`
    /// 任意一个，就静默走普通上报，**不报错**。等回执发现已经太晚，所以这里先自查。
    pub fn durable_eligible(&self) -> bool {
        let high_value_absent = self.hr.is_none()
            && self.bp.is_none()
            && self.alarm.is_none()
            && self.lng.is_none()
            && self.lat.is_none()
            && self.battery.is_none()
            && self.app_version.is_none();
        let low_value_present =
            self.step.is_some() || self.sleep.is_some() || self.sleep_state.is_some();
        let id_ok = self
            .request_id
            .as_deref()
            .map(request_id_ok)
            .unwrap_or(false);
        high_value_absent && low_value_present && id_ok
    }
}

/// 后端 `durableRequestIDPattern` / `sosRequestIDPattern`：`^[A-Za-z0-9_.:-]{1,64}$`。
pub fn request_id_ok(id: &str) -> bool {
    !id.is_empty()
        && id.len() <= 64
        && id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '.' | ':' | '-'))
}

fn check_metric(name: &str, value: Option<i64>, min: i64, max: i64) -> Result<(), String> {
    match value {
        // 后端 `parseMetric` 对 0 直接放行，不参与区间判断。
        Some(0) | None => Ok(()),
        Some(v) if v < min || v > max => Err(format!("{name}={v} 超出 {min}..{max}")),
        Some(_) => Ok(()),
    }
}

// ---------------------------------------------------------------------------
// HarmonyOS 批量采样（POST /api/v1/device/healthBatch）
// ---------------------------------------------------------------------------

/// 常量元数据借用静态字符串；设备和时间字段可由同一批次共享。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct SampleMetadata {
    pub simulated: Cow<'static, str>,
    #[serde(rename = "counterTimezone", skip_serializing_if = "Option::is_none")]
    pub counter_timezone: Option<Cow<'static, str>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sensor: Option<Cow<'static, str>>,
    #[serde(rename = "sensorTimestamp", skip_serializing_if = "Option::is_none")]
    pub sensor_timestamp: Option<Cow<'static, str>>,
    #[serde(rename = "timestampBasis", skip_serializing_if = "Option::is_none")]
    pub timestamp_basis: Option<Cow<'static, str>>,
    #[serde(rename = "counterBasis", skip_serializing_if = "Option::is_none")]
    pub counter_basis: Option<Cow<'static, str>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct HealthSampleData<S, R = String> {
    #[serde(rename = "recordId")]
    pub record_id: R,
    pub source: Cow<'static, str>,
    #[serde(rename = "sourceDevice")]
    pub source_device: S,
    #[serde(rename = "sourceVersion")]
    pub source_version: Cow<'static, str>,
    pub metric: Cow<'static, str>,
    pub unit: Cow<'static, str>,
    /// 整数指标（心率、步数、血氧）序列化成 `72` 而不是 `72.0`。
    /// 200 条批次能省约 400 字节，同时后端仍按 float64 解析。
    #[serde(serialize_with = "ser_compact_number")]
    pub value: f64,
    pub start: S,
    pub end: S,
    pub semantics: Cow<'static, str>,
    pub metadata: SampleMetadata,
    pub version: i64,
}

pub type HealthSample = HealthSampleData<String>;
pub type SampleView<'a> = HealthSampleData<&'a str>;
pub type SampleRef<'a> = HealthSampleData<&'a str, &'a str>;

/// Reuse caller-owned storage while retaining the run/sequence/metric ID grammar.
pub fn record_id_into(output: &mut String, prefix: &str, sequence: u64, suffix: &str) {
    output.clear();
    output.push_str(prefix);
    output.push('-');
    output.push_str(itoa::Buffer::new().format(sequence));
    output.push('-');
    output.push_str(suffix);
}

impl<S: Clone, R> HealthSampleData<S, R> {
    fn heart_rate_fields(record_id: R, device_no: S, value: i64, at: S) -> Self {
        Self {
            record_id,
            source: Cow::Borrowed("suining-watch-sim"),
            source_device: device_no,
            source_version: Cow::Borrowed("2.0.0"),
            metric: Cow::Borrowed("heart_rate"),
            unit: Cow::Borrowed("count/min"),
            value: value as f64,
            start: at.clone(),
            end: at,
            semantics: Cow::Borrowed("instant"),
            metadata: SampleMetadata {
                simulated: Cow::Borrowed("true"),
                counter_timezone: None,
                sensor: Some(Cow::Borrowed("watch-sim:hr")),
                sensor_timestamp: None,
                timestamp_basis: Some(Cow::Borrowed("app-received")),
                counter_basis: None,
            },
            version: 1,
        }
    }

    fn steps_fields(record_id: R, device_no: S, value: i64, day_start: S, at: S) -> Self {
        Self {
            record_id,
            source: Cow::Borrowed("suining-watch-sim"),
            source_device: device_no,
            source_version: Cow::Borrowed("2.0.0"),
            metric: Cow::Borrowed("steps"),
            unit: Cow::Borrowed("count"),
            value: value as f64,
            start: day_start,
            end: at,
            semantics: Cow::Borrowed("daily_total"),
            metadata: SampleMetadata {
                simulated: Cow::Borrowed("true"),
                counter_timezone: Some(Cow::Borrowed("UTC")),
                sensor: Some(Cow::Borrowed("watch-sim:pedometer")),
                sensor_timestamp: None,
                timestamp_basis: Some(Cow::Borrowed("app-received")),
                counter_basis: Some(Cow::Borrowed("utc-day")),
            },
            version: 1,
        }
    }
}

impl HealthSample {
    pub fn heart_rate(record_id: String, device_no: &str, value: i64, at: &str) -> Self {
        Self::heart_rate_fields(record_id, device_no.into(), value, at.into())
    }
    pub fn steps(
        record_id: String,
        device_no: &str,
        value: i64,
        day_start: &str,
        at: &str,
    ) -> Self {
        Self::steps_fields(
            record_id,
            device_no.into(),
            value,
            day_start.into(),
            at.into(),
        )
    }
    /// 精简版：省略 `sensor` 与 `timestampBasis` 两个可选元数据。
    ///
    /// 单条约省 60 字节（占比 18%）。后端不校验这两个字段，但前端"数据来源"
    /// 面板会少两项信息，所以只在压测/容量评估时开，默认不启用。
    pub fn heart_rate_lean(record_id: String, device_no: &str, value: i64, at: &str) -> Self {
        let mut sample = Self::heart_rate(record_id, device_no, value, at);
        sample.metadata.sensor = None;
        sample.metadata.timestamp_basis = None;
        sample
    }
}
impl<'a> SampleView<'a> {
    pub fn heart_rate(record_id: String, device_no: &'a str, value: i64, at: &'a str) -> Self {
        Self::heart_rate_fields(record_id, device_no, value, at)
    }
    pub fn steps(
        record_id: String,
        device_no: &'a str,
        value: i64,
        day_start: &'a str,
        at: &'a str,
    ) -> Self {
        Self::steps_fields(record_id, device_no, value, day_start, at)
    }
}

impl<'a> SampleRef<'a> {
    pub fn heart_rate(record_id: &'a str, device_no: &'a str, value: i64, at: &'a str) -> Self {
        Self::heart_rate_fields(record_id, device_no, value, at)
    }
    pub fn steps(
        record_id: &'a str,
        device_no: &'a str,
        value: i64,
        day_start: &'a str,
        at: &'a str,
    ) -> Self {
        Self::steps_fields(record_id, device_no, value, day_start, at)
    }
}

#[derive(Serialize)]
pub struct BatchView<'a, S, R = String> {
    #[serde(rename = "deviceNo")]
    pub device_no: &'a str,
    pub secret: &'a str,
    #[serde(rename = "bindingVersion")]
    pub binding_version: i64,
    pub samples: &'a [HealthSampleData<S, R>],
}

#[derive(Debug, Clone, Serialize)]
pub struct BatchRequest {
    #[serde(rename = "deviceNo")]
    pub device_no: String,
    pub secret: String,
    #[serde(rename = "bindingVersion")]
    pub binding_version: i64,
    pub samples: Vec<HealthSample>,
}

impl BatchRequest {
    pub fn new(cred: &Credentials, binding_version: i64, samples: Vec<HealthSample>) -> Self {
        BatchRequest {
            device_no: cred.device_no.clone(),
            secret: cred.secret.clone(),
            binding_version,
            samples,
        }
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.binding_version < 1 {
            return Err("bindingVersion 必须 ≥1，请先登录取得绑定版本".into());
        }
        if self.samples.len() > MAX_BATCH_SIZE {
            return Err(format!(
                "单批最多 {MAX_BATCH_SIZE} 条，实际 {}",
                self.samples.len()
            ));
        }
        if self.samples.is_empty() {
            return Err("批次不能为空".into());
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// SOS（POST /api/v1/device/sos）
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize)]
pub struct SosRequest {
    #[serde(rename = "deviceNo")]
    pub device_no: String,
    pub secret: String,
    #[serde(rename = "requestId")]
    pub request_id: String,
    #[serde(rename = "bindingVersion")]
    pub binding_version: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lng: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub lat: Option<String>,
    #[serde(rename = "positionSource")]
    pub position_source: &'static str,
}

impl SosRequest {
    pub fn validate(&self) -> Result<(), String> {
        if self.binding_version < 1 {
            return Err("SOS 必须携带 bindingVersion".into());
        }
        if self.request_id.is_empty() || self.request_id.len() > 64 {
            return Err("requestId 长度必须为 1–64".into());
        }
        if !self
            .request_id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '.' | ':' | '-'))
        {
            return Err("requestId 只允许字母、数字与 _ . : - ".into());
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn prepared_signer_matches_all_hash_and_key_boundaries() {
        for device in ["WATCH0001", " watch0001 ", " ǅß手表 ", ""] {
            for key_size in [0, 1, 64, 65, 129] {
                let cred = Credentials {
                    device_no: device.into(),
                    secret: "k".repeat(key_size),
                };
                let signer = PreparedSigner::new(&cred);
                for size in [0, 1, 55, 56, 63, 64, 65, 512, 65536] {
                    let body = vec![0x9a; size];
                    for timestamp in [i64::MIN, -1, 0, i64::MAX] {
                        let nonce = "test-nonce-0123456789abcdef";
                        let actual = signer.headers(&body, timestamp, nonce.into());
                        assert_eq!(
                            actual.signature,
                            sign(&cred.secret, &cred.device_no, timestamp, nonce, &body)
                        );
                        assert_eq!(actual.timestamp, timestamp.to_string());
                        assert_eq!(actual.nonce, nonce);
                    }
                }
            }
        }
    }

    #[test]
    fn prepared_signers_keep_devices_and_secret_generations_independent() {
        let mut cred = Credentials::new("WATCH0001", "first");
        let old = PreparedSigner::new(&cred);
        cred.secret = "rotated".into();
        let rotated = PreparedSigner::new(&cred);
        cred.device_no = "WATCH0002".into();
        let other = PreparedSigner::new(&cred);
        for _ in 0..3 {
            assert_eq!(
                (&cred as &dyn RequestSigning)
                    .headers(b"body", 1, "nonce".into())
                    .signature,
                sign("rotated", "WATCH0002", 1, "nonce", b"body")
            );
            assert_eq!(
                old.headers(b"body", 1, "nonce".into()).signature,
                sign("first", "WATCH0001", 1, "nonce", b"body")
            );
            assert_eq!(
                rotated.headers(b"body", 1, "nonce".into()).signature,
                sign("rotated", "WATCH0001", 1, "nonce", b"body")
            );
            assert_eq!(
                other.headers(b"body", 1, "nonce".into()).signature,
                sign("rotated", "WATCH0002", 1, "nonce", b"body")
            );
        }
    }

    #[test]
    fn prepared_signer_supports_concurrent_independent_requests() {
        let cred = Credentials::new("WATCH0001", "concurrent-test");
        let signer = PreparedSigner::new(&cred);
        std::thread::scope(|scope| {
            for worker in 0..8 {
                let signer = &signer;
                let cred = &cred;
                scope.spawn(move || {
                    for sequence in 0..32 {
                        let nonce = format!("{worker}-{sequence}");
                        let body = format!("body-{sequence}");
                        assert_eq!(
                            signer
                                .headers(body.as_bytes(), sequence, nonce.clone())
                                .signature,
                            sign(
                                &cred.secret,
                                &cred.device_no,
                                sequence,
                                &nonce,
                                body.as_bytes()
                            )
                        );
                    }
                });
            }
        });
    }

    #[test]
    fn signature_streaming_matches_reference_at_boundaries() {
        for device in ["WATCH0001", " watch0001 ", " ǅß手表 ", ""] {
            for timestamp in [i64::MIN, -1, 0, i64::MAX] {
                for body in [&b""[..], &b"{}"[..], &b"\0\xff\n"[..]] {
                    let key = "k".repeat(129);
                    let reference = format!(
                        "{}:{}:{}:{}",
                        device.trim().to_uppercase(),
                        timestamp,
                        "nonce",
                        hex_lower(&Sha256::digest(body))
                    );
                    let mut mac = HmacSha256::new_from_slice(key.as_bytes()).unwrap();
                    mac.update(reference.as_bytes());
                    assert_eq!(
                        sign(&key, device, timestamp, "nonce", body),
                        hex_lower(&mac.finalize().into_bytes())
                    );
                }
            }
        }
    }

    #[test]
    fn borrowed_batches_preserve_every_wire_field() {
        let cred = Credentials::new("WATCH设备\"1", "test\\secret");
        let at = "2026-11-02T08:00:00Z";
        let start = "2026-11-02T00:00:00Z";
        let owned = BatchRequest::new(
            &cred,
            3,
            vec![
                HealthSample::heart_rate("id-hr".into(), &cred.device_no, 76, at),
                HealthSample::steps("id-step".into(), &cred.device_no, 1200, start, at),
            ],
        );
        let samples = [
            SampleView::heart_rate("id-hr".into(), &cred.device_no, 76, at),
            SampleView::steps("id-step".into(), &cred.device_no, 1200, start, at),
        ];
        let borrowed = BatchView {
            device_no: &cred.device_no,
            secret: &cred.secret,
            binding_version: 3,
            samples: &samples,
        };
        assert_eq!(
            serde_json::to_vec(&owned).unwrap(),
            serde_json::to_vec(&borrowed).unwrap()
        );
    }

    #[test]
    fn signature_matches_backend_formula() {
        // 手算一遍公式，防止某次重构悄悄改变了拼接顺序。
        let body = br#"{"deviceNo":"WATCH0001","secret":"s"}"#;
        let digest = Sha256::digest(body);
        let expected_msg = format!(
            "WATCH0001:1700000000:abcdefghijklmnop:{}",
            hex_lower(&digest)
        );
        let mut mac = HmacSha256::new_from_slice(b"secret").unwrap();
        mac.update(expected_msg.as_bytes());
        let expected = hex_lower(&mac.finalize().into_bytes());
        assert_eq!(
            sign(
                "secret",
                "watch0001",
                1_700_000_000,
                "abcdefghijklmnop",
                body
            ),
            expected
        );
    }

    #[test]
    fn signature_is_lowercase_hex_64() {
        let s = sign("secret", "W1", 0, "nnnnnnnnnnnnnnnn", b"{}");
        assert_eq!(s.len(), 64);
        assert!(s
            .chars()
            .all(|c| c.is_ascii_hexdigit() && !c.is_ascii_uppercase()));
    }

    #[test]
    fn device_no_is_uppercased_in_signature() {
        let a = sign("k", "watch0001", 1, "nnnnnnnnnnnnnnnn", b"{}");
        let b = sign("k", "WATCH0001", 1, "nnnnnnnnnnnnnnnn", b"{}");
        assert_eq!(a, b);
    }

    #[test]
    fn report_validation_catches_backend_rejections() {
        let cred = Credentials::new("WATCH0001", "x");
        let mut p = ReportPayload::new(&cred, "2026-11-01 12:00:00".into());
        assert!(p.validate().is_ok());

        p.hr = Some(300);
        assert!(p.validate().is_err(), "hr 超过 250 必须提前拦下");
        p.hr = Some(72);

        p.lng = Some("113.062500".into());
        assert!(p.validate().is_err(), "只有经度没有纬度必须拦下");
        p.lat = Some("28.197500".into());
        assert!(p.validate().is_ok());

        p.alarm = Some("sos".into());
        assert!(p.validate().is_err(), "alarm 只接受 1 或 fall");
        p.alarm = Some("fall".into());
        assert!(p.validate().is_ok());

        p.position_source = Some("wifi".into());
        assert!(p.validate().is_err());
        p.position_source = Some("fused".into());
        assert!(p.validate().is_ok());
    }

    #[test]
    fn durable_eligibility_matches_backend_rules() {
        // 后端 durableTelemetryEligible 是排他判定：混入任一高价值字段即静默降级。
        let cred = Credentials::new("WATCH0001", "x");
        let mut p = ReportPayload::new(&cred, "2026-11-01 12:00:00".into());
        p.step = Some(1200);
        assert!(!p.durable_eligible(), "缺 requestId 不够格");

        p.request_id = Some("dur-watch0001-1".into());
        assert!(p.durable_eligible(), "step-only + 合法号应够格");

        p.hr = Some(72);
        assert!(!p.durable_eligible(), "带心率必须不够格");
        p.hr = None;

        p.battery = Some(80);
        assert!(!p.durable_eligible(), "带电量必须不够格");
        p.battery = None;

        p.lng = Some("113.062500".into());
        p.lat = Some("28.197500".into());
        assert!(!p.durable_eligible(), "带位置必须不够格");
        p.lng = None;
        p.lat = None;

        p.request_id = Some("bad id".into());
        assert!(!p.durable_eligible(), "requestId 含空格不够格");
        p.request_id = Some("x".repeat(65));
        assert!(!p.durable_eligible(), "requestId 超 64 不够格");

        p.request_id = Some("dur-watch0001-2".into());
        p.step = None;
        p.sleep = None;
        p.sleep_state = None;
        assert!(
            !p.durable_eligible(),
            "只有 requestId、没有低价值指标不够格"
        );
    }

    #[test]
    fn zero_metrics_are_allowed_like_backend() {
        let cred = Credentials::new("W", "x");
        let mut p = ReportPayload::new(&cred, "2026-11-01 12:00:00".into());
        p.hr = Some(0);
        p.bp = Some(0);
        // 后端 parseMetric 对 0 直接放行。
        assert!(p.validate().is_ok());
    }

    #[test]
    fn sos_request_id_rules() {
        let mut s = SosRequest {
            position_source: "simulated",
            device_no: "W".into(),
            secret: "s".into(),
            request_id: "sos-2026-11-01.abc:1".into(),
            binding_version: 1,
            lng: None,
            lat: None,
        };
        assert!(s.validate().is_ok());
        s.request_id = "bad id!".into();
        assert!(s.validate().is_err());
        s.request_id = "ok".into();
        s.binding_version = 0;
        assert!(s.validate().is_err());
    }

    #[test]
    fn batch_enforces_order_and_size() {
        let cred = Credentials::new("W", "s");
        assert!(BatchRequest::new(&cred, 0, vec![]).validate().is_err());
        let sample = HealthSample::heart_rate("r-hr".into(), "W", 72, "2026-11-01T00:00:00Z");
        let many = vec![sample.clone(); MAX_BATCH_SIZE + 1];
        assert!(BatchRequest::new(&cred, 1, many).validate().is_err());
    }

    #[test]
    fn steps_sample_declares_utc_day_basis() {
        let s = HealthSample::steps(
            "r-steps".into(),
            "W",
            1200,
            "2026-11-01T00:00:00Z",
            "2026-11-01T08:00:00Z",
        );
        assert_eq!(s.semantics, "daily_total");
        assert_eq!(s.metadata.counter_timezone.as_deref(), Some("UTC"));
        assert!(s.start.ends_with("T00:00:00Z"));
    }
}
