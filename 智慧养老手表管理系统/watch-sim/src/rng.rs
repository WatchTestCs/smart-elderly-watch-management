//! 确定性伪随机数（PCG32）。
//!
//! 旧模拟器用 `Math.random()`，同一场景每次跑出来的数据都不一样，无法写成断言。
//! 这里换成显式种子的 PCG32：`--seed` 固定后整条时间轴可复现，CI 才敢断言数值。

pub struct Pcg32 {
    state: u64,
    inc: u64,
}

const MULTIPLIER: u64 = 6_364_136_223_846_793_005;

impl Pcg32 {
    pub fn new(seed: u64) -> Self {
        let mut rng = Pcg32 {
            state: 0,
            inc: (seed << 1) | 1,
        };
        rng.next_u32();
        rng.state = rng.state.wrapping_add(seed);
        rng.next_u32();
        rng
    }

    #[inline]
    pub fn next_u32(&mut self) -> u32 {
        let old = self.state;
        self.state = old.wrapping_mul(MULTIPLIER).wrapping_add(self.inc);
        let xorshifted = (((old >> 18) ^ old) >> 27) as u32;
        let rot = (old >> 59) as u32;
        xorshifted.rotate_right(rot)
    }

    /// 均匀分布 `[0, 1)`。
    #[inline]
    pub fn next_f64(&mut self) -> f64 {
        (self.next_u32() as f64) / (u32::MAX as f64 + 1.0)
    }

    /// 均匀整数 `[lo, hi]`，两端闭合。
    pub fn range_i64(&mut self, lo: i64, hi: i64) -> i64 {
        if hi <= lo {
            return lo;
        }
        let span = (hi - lo + 1) as u64;
        lo + (self.next_u32() as u64 % span) as i64
    }

    pub fn range_f64(&mut self, lo: f64, hi: f64) -> f64 {
        lo + (hi - lo) * self.next_f64()
    }

    /// 标准正态，Box–Muller。
    pub fn gauss(&mut self) -> f64 {
        let u1 = self.next_f64().max(1e-12);
        let u2 = self.next_f64();
        (-2.0 * u1.ln()).sqrt() * (std::f64::consts::TAU * u2).cos()
    }

    /// `base64url` 字母表里取 `len` 位，用于请求 nonce（后端要求 16–64 位）。
    pub fn nonce(&mut self, len: usize) -> String {
        const ALPHABET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
        let len = len.clamp(16, 64);
        (0..len)
            .map(|_| ALPHABET[self.next_u32() as usize % ALPHABET.len()] as char)
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn same_seed_gives_same_stream() {
        let mut a = Pcg32::new(42);
        let mut b = Pcg32::new(42);
        for _ in 0..1000 {
            assert_eq!(a.next_u32(), b.next_u32());
        }
    }

    #[test]
    fn different_seed_diverges() {
        let mut a = Pcg32::new(1);
        let mut b = Pcg32::new(2);
        let sa: Vec<u32> = (0..16).map(|_| a.next_u32()).collect();
        let sb: Vec<u32> = (0..16).map(|_| b.next_u32()).collect();
        assert_ne!(sa, sb);
    }

    #[test]
    fn range_is_inclusive_and_bounded() {
        let mut r = Pcg32::new(7);
        for _ in 0..5000 {
            let v = r.range_i64(3, 9);
            assert!((3..=9).contains(&v), "out of range: {v}");
        }
    }

    #[test]
    fn nonce_matches_backend_pattern() {
        let mut r = Pcg32::new(9);
        for _ in 0..50 {
            let n = r.nonce(24);
            assert_eq!(n.len(), 24);
            assert!(n
                .chars()
                .all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '-'));
        }
    }

    #[test]
    fn gauss_has_reasonable_moments() {
        let mut r = Pcg32::new(11);
        let n = 20_000;
        let samples: Vec<f64> = (0..n).map(|_| r.gauss()).collect();
        let mean = samples.iter().sum::<f64>() / n as f64;
        let var = samples.iter().map(|x| (x - mean).powi(2)).sum::<f64>() / n as f64;
        assert!(mean.abs() < 0.05, "mean drifted: {mean}");
        assert!((var - 1.0).abs() < 0.1, "variance drifted: {var}");
    }
}
