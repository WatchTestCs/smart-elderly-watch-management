// ---------------------------------------------------------------------------
// 请求签名
// ---------------------------------------------------------------------------

use std::borrow::Cow;

use hmac::{Hmac, Mac};
use sha2::{Digest, Sha256};

use super::Credentials;

type HmacSha256 = Hmac<Sha256>;

/// 三个签名头。后端在 `SUINING_REQUIRE_SIGNED_DEVICE=1` 时强制要求，
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
    for (pair, byte) in output.as_chunks_mut::<2>().0.iter_mut().zip(bytes) {
        pair[0] = HEX[(byte >> 4) as usize];
        pair[1] = HEX[(byte & 0x0f) as usize];
    }
}
