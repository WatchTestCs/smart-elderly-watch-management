use crate::protocol::{BatchView, Credentials, HealthSampleData};

/// Scratch space belongs to one device session; encoded strings never cross sessions.
#[derive(Default)]
pub struct BatchEncoder {
    device: Vec<u8>,
    start: Vec<u8>,
    end: Vec<u8>,
}

impl BatchEncoder {
    pub fn write<R: serde::Serialize>(
        &mut self,
        output: &mut Vec<u8>,
        credentials: &Credentials,
        binding_version: i64,
        samples: &[HealthSampleData<&str, R>],
    ) -> Result<(), serde_json::Error> {
        output.clear();
        let common = !samples.is_empty()
            && samples
                .iter()
                .all(|sample| canonical(sample) && sample.source_device == credentials.device_no);
        if !common {
            return serde_json::to_writer(
                output,
                &BatchView {
                    device_no: &credentials.device_no,
                    secret: &credentials.secret,
                    binding_version,
                    samples,
                },
            );
        }
        self.device.clear();
        serde_json::to_writer(&mut self.device, &credentials.device_no)?;
        output.extend_from_slice(b"{\"deviceNo\":");
        output.extend_from_slice(&self.device);
        output.extend_from_slice(b",\"secret\":");
        serde_json::to_writer(&mut *output, &credentials.secret)?;
        output.extend_from_slice(b",\"bindingVersion\":");
        let mut number = itoa::Buffer::new();
        output.extend_from_slice(number.format(binding_version).as_bytes());
        output.extend_from_slice(b",\"samples\":[");
        let mut cached_start = None;
        let mut cached_end = None;
        for (index, sample) in samples.iter().enumerate() {
            if cached_start != Some(sample.start) {
                self.start.clear();
                serde_json::to_writer(&mut self.start, sample.start)?;
                cached_start = Some(sample.start);
            }
            let end = if sample.start == sample.end {
                &self.start
            } else {
                if cached_end != Some(sample.end) {
                    self.end.clear();
                    serde_json::to_writer(&mut self.end, sample.end)?;
                    cached_end = Some(sample.end);
                }
                &self.end
            };
            if index != 0 {
                output.push(b',');
            }
            output.extend_from_slice(b"{\"recordId\":");
            serde_json::to_writer(&mut *output, &sample.record_id)?;
            output.extend_from_slice(b",\"source\":\"suining-watch-sim\",\"sourceDevice\":");
            output.extend_from_slice(&self.device);
            let heart_rate = sample.metric == "heart_rate";
            if heart_rate {
                output.extend_from_slice(b",\"sourceVersion\":\"2.0.0\",\"metric\":\"heart_rate\",\"unit\":\"count/min\",\"value\":");
            } else {
                output.extend_from_slice(b",\"sourceVersion\":\"2.0.0\",\"metric\":\"steps\",\"unit\":\"count\",\"value\":");
            }
            output.extend_from_slice(number.format(sample.value as i64).as_bytes());
            output.extend_from_slice(b",\"start\":");
            output.extend_from_slice(&self.start);
            output.extend_from_slice(b",\"end\":");
            output.extend_from_slice(end);
            if heart_rate {
                output.extend_from_slice(b",\"semantics\":\"instant\",\"metadata\":{\"simulated\":\"true\",\"sensor\":\"watch-sim:hr\",\"timestampBasis\":\"app-received\"},\"version\":1}");
            } else {
                output.extend_from_slice(b",\"semantics\":\"daily_total\",\"metadata\":{\"simulated\":\"true\",\"counterTimezone\":\"UTC\",\"sensor\":\"watch-sim:pedometer\",\"timestampBasis\":\"app-received\",\"counterBasis\":\"utc-day\"},\"version\":1}");
            }
        }
        output.extend_from_slice(b"]}");
        Ok(())
    }
}

fn canonical<R>(sample: &HealthSampleData<&str, R>) -> bool {
    // Exhaustive patterns make protocol additions a compile error here until the fast path is reviewed.
    let crate::protocol::HealthSampleData {
        record_id: _,
        source: _,
        source_device: _,
        source_version: _,
        metric: _,
        unit: _,
        value: _,
        start: _,
        end: _,
        semantics: _,
        metadata,
        version: _,
    } = sample;
    let crate::protocol::SampleMetadata {
        simulated: _,
        counter_timezone: _,
        sensor: _,
        sensor_timestamp: _,
        timestamp_basis: _,
        counter_basis: _,
    } = metadata;
    if sample.source != "suining-watch-sim"
        || sample.source_version != "2.0.0"
        || sample.version != 1
        || !sample.value.is_finite()
        || sample.value.fract() != 0.0
        || sample.value.abs() >= 9.0e15
        || metadata.simulated != "true"
        || metadata.sensor_timestamp.is_some()
        || metadata.timestamp_basis.as_deref() != Some("app-received")
    {
        return false;
    }
    match sample.metric.as_ref() {
        "heart_rate" => {
            sample.unit == "count/min"
                && sample.semantics == "instant"
                && metadata.counter_timezone.is_none()
                && metadata.counter_basis.is_none()
                && metadata.sensor.as_deref() == Some("watch-sim:hr")
        }
        "steps" => {
            sample.unit == "count"
                && sample.semantics == "daily_total"
                && metadata.counter_timezone.as_deref() == Some("UTC")
                && metadata.counter_basis.as_deref() == Some("utc-day")
                && metadata.sensor.as_deref() == Some("watch-sim:pedometer")
        }
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::protocol::SampleView;
    use std::borrow::Cow;

    #[test]
    fn reused_record_ids_match_owned_requests_across_sequences_and_fallback() {
        use crate::protocol::{record_id_into, SampleRef};
        let credentials = Credentials::new("WATCH0001", "test-only");
        let mut ids = [String::with_capacity(64), String::with_capacity(64)];
        let mut encoder = BatchEncoder::default();
        let mut output = Vec::new();
        for prefix in ["0123456789abcdef0123456789abcdef", "escape-\"\\中文"] {
            for sequence in [0, 9, 10, 999, u64::MAX] {
                for (id, suffix) in ids.iter_mut().zip(["hr", "steps"]) {
                    record_id_into(id, prefix, sequence, suffix);
                    assert_eq!(id, &format!("{prefix}-{sequence}-{suffix}"));
                }
                let at = "2026-11-02T08:00:00Z";
                let start = "2026-11-02T00:00:00Z";
                let mut borrowed = [
                    SampleRef::heart_rate(&ids[0], &credentials.device_no, 76, at),
                    SampleRef::steps(&ids[1], &credentials.device_no, 800, start, at),
                ];
                let mut owned = [
                    SampleView::heart_rate(
                        format!("{prefix}-{sequence}-hr"),
                        &credentials.device_no,
                        76,
                        at,
                    ),
                    SampleView::steps(
                        format!("{prefix}-{sequence}-steps"),
                        &credentials.device_no,
                        800,
                        start,
                        at,
                    ),
                ];
                for fallback in [false, true] {
                    if fallback {
                        borrowed[0].metadata.sensor = None;
                        owned[0].metadata.sensor = None;
                    }
                    let expected = serde_json::to_vec(&BatchView {
                        device_no: &credentials.device_no,
                        secret: &credentials.secret,
                        binding_version: 7,
                        samples: &owned,
                    })
                    .unwrap();
                    encoder
                        .write(&mut output, &credentials, 7, &borrowed)
                        .unwrap();
                    assert_eq!(output, expected);
                }
            }
        }
    }

    fn check(samples: &[SampleView<'_>]) {
        let credentials = Credentials::new("WATCH0001", "test\"\\secret");
        let expected = serde_json::to_vec(&BatchView {
            device_no: &credentials.device_no,
            secret: &credentials.secret,
            binding_version: 3,
            samples,
        })
        .unwrap();
        let mut actual = b"previous body".to_vec();
        let mut encoder = BatchEncoder::default();
        for _ in 0..2 {
            encoder
                .write(&mut actual, &credentials, 3, samples)
                .unwrap();
            assert_eq!(actual, expected);
        }
    }

    #[test]
    fn canonical_batches_preserve_exact_bytes_and_reused_buffers() {
        let at = "2026-11-02T08:00:00Z";
        for count in [1, 2, 8, 200] {
            let samples: Vec<_> = (0..count)
                .map(|index| {
                    SampleView::heart_rate(format!("id-\"\\中文-{index}"), "WATCH0001", 76, at)
                })
                .collect();
            check(&samples);
        }
        check(&[SampleView::steps(
            "step".into(),
            "WATCH0001",
            1234,
            "2026-11-02T00:00:00Z",
            at,
        )]);
        check(&[
            SampleView::heart_rate("mixed-hr".into(), "WATCH0001", 76, at),
            SampleView::steps(
                "mixed-step".into(),
                "WATCH0001",
                1234,
                "2026-11-02T00:00:00Z",
                at,
            ),
        ]);
    }

    #[test]
    fn every_noncanonical_field_keeps_the_generic_encoding() {
        let base = SampleView::heart_rate("id".into(), "WATCH0001", 76, "2026-11-02T08:00:00Z");
        let mut changes = vec![base.clone(); 17];
        changes[0].source = Cow::Borrowed("custom");
        changes[1].source_device = "other";
        changes[2].source_version = Cow::Borrowed("new");
        changes[3].metric = Cow::Borrowed("spo2");
        changes[4].unit = Cow::Borrowed("other");
        changes[5].value = 7.5;
        changes[6].semantics = Cow::Borrowed("custom");
        changes[7].version = 2;
        changes[8].metadata.simulated = Cow::Borrowed("false");
        changes[9].metadata.counter_timezone = Some(Cow::Borrowed("Asia/Shanghai"));
        changes[10].metadata.sensor = Some(Cow::Borrowed("custom"));
        changes[11].metadata.sensor_timestamp = Some(Cow::Borrowed("device time"));
        changes[12].metadata.timestamp_basis = None;
        changes[13].metadata.counter_basis = Some(Cow::Borrowed("custom"));
        changes[14].value = f64::NAN;
        changes[15].value = 9.0e15;
        changes[16].value = -0.0;
        for changed in changes {
            check(&[changed]);
        }
        let mut later = base.clone();
        later.start = "2026-11-02T09:00:00Z";
        check(&[base.clone(), later]);
        let mut later = base.clone();
        later.end = "2026-11-02T09:00:00Z";
        check(&[base, later]);
        check(&[]);
    }

    #[test]
    fn identity_epoch_and_shared_strings_are_refreshed_between_writes() {
        let mut encoder = BatchEncoder::default();
        let mut output = Vec::new();
        for (index, device) in ["WATCH0001", "WATCH\"设备\\2", "WATCH0001"]
            .into_iter()
            .enumerate()
        {
            let credentials = Credentials::new(device, format!("test-{index}"));
            let at = if index % 2 == 0 {
                "2026-11-02T08:00:00Z"
            } else {
                "2026-11-03T00:00:00Z"
            };
            let samples = [
                SampleView::heart_rate("hr".into(), &credentials.device_no, 76, at),
                SampleView::steps("steps".into(), &credentials.device_no, 1234, at, at),
            ];
            let version = index as i64 + 1;
            encoder
                .write(&mut output, &credentials, version, &samples)
                .unwrap();
            assert_eq!(
                output,
                serde_json::to_vec(&BatchView {
                    device_no: &credentials.device_no,
                    secret: &credentials.secret,
                    binding_version: version,
                    samples: &samples
                })
                .unwrap()
            );
        }
    }
}
