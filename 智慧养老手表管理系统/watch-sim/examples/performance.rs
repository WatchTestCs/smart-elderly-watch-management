use serde_json::json;
use std::{
    alloc::{GlobalAlloc, Layout, System},
    hint::black_box,
    io::{Read, Write},
    net::TcpListener,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc,
    },
    time::{Duration, Instant},
};
use watch_sim::{
    device::DeviceSession,
    physio::{Physiology, Targets},
    protocol::{
        record_id_into, signature_headers, BatchRequest, BatchView, Credentials, HealthSample,
        Platform, ReportPayload, SampleRef, SampleView,
    },
    transport::Client,
};

struct CountingAllocator;
static ALLOCATIONS: AtomicU64 = AtomicU64::new(0);
static BYTES: AtomicU64 = AtomicU64::new(0);
unsafe impl GlobalAlloc for CountingAllocator {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        ALLOCATIONS.fetch_add(1, Ordering::Relaxed);
        BYTES.fetch_add(layout.size() as u64, Ordering::Relaxed);
        System.alloc(layout)
    }
    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        System.dealloc(ptr, layout);
    }
    unsafe fn realloc(&self, ptr: *mut u8, layout: Layout, size: usize) -> *mut u8 {
        ALLOCATIONS.fetch_add(1, Ordering::Relaxed);
        BYTES.fetch_add(size as u64, Ordering::Relaxed);
        System.realloc(ptr, layout, size)
    }
}
#[global_allocator]
static ALLOCATOR: CountingAllocator = CountingAllocator;

fn measure<T>(
    name: &str,
    seconds: f64,
    samples: usize,
    threaded: bool,
    mut work: impl FnMut() -> T,
) {
    for _ in 0..128 {
        black_box(work());
    }
    let before_alloc = ALLOCATIONS.load(Ordering::Relaxed);
    let before_bytes = BYTES.load(Ordering::Relaxed);
    let start = Instant::now();
    let mut iterations = 0u64;
    while start.elapsed().as_secs_f64() < seconds {
        for _ in 0..64 {
            black_box(work());
        }
        iterations += 64;
    }
    let elapsed = start.elapsed().as_secs_f64();
    let allocs = ALLOCATIONS.load(Ordering::Relaxed) - before_alloc;
    let bytes = BYTES.load(Ordering::Relaxed) - before_bytes;
    println!(
        "{}",
        json!({"case":name,"iterations":iterations,"seconds":elapsed,
        "operations_per_second":iterations as f64/elapsed,"ns_per_operation":elapsed*1e9/iterations as f64,
        "samples_per_second":iterations as f64*samples as f64/elapsed,
        "allocations_per_operation":if threaded {None}else{Some(allocs as f64/iterations as f64)},
        "allocated_bytes_per_operation":if threaded {None}else{Some(bytes as f64/iterations as f64)}})
    );
}

fn main() {
    let args: Vec<_> = std::env::args().collect();
    let seconds = args.get(1).and_then(|s| s.parse().ok()).unwrap_or(1.0f64);
    assert!(seconds.is_finite() && seconds > 0.0);
    let selected = args.get(2).map(String::as_str).unwrap_or("all");
    let enabled = |name: &str| selected == "all" || selected == name;
    let cred = Credentials::new("BENCH0001", "test-only-0123456789abcdef01234567");
    let at = "2026-11-02T00:00:00Z";
    let mut model = Physiology::new(42, 113.0625, 28.1975, 9.0);
    if enabled("physiology") {
        measure("physiology", seconds, 1, false, || {
            model.tick(3.0, Targets::default())
        });
    }
    let mut report = ReportPayload::new(&cred, "2026-11-02 08:00:00".into());
    report.hr = Some(76);
    report.bp = Some(120);
    report.step = Some(800);
    report.sleep = Some(8);
    report.sleep_state = Some("0".into());
    report.battery = Some(88);
    report.lng = Some("113.062500".into());
    report.lat = Some("28.197500".into());
    report.source = Some("gps".into());
    report.app_version = Some("2.0.0-sim".into());
    if enabled("report_json") {
        measure("report_json", seconds, 1, false, || {
            serde_json::to_vec(black_box(&report)).unwrap()
        });
    }
    for count in [2, 200] {
        let name = format!("batch_{count}_construct_json");
        if enabled(&name) {
            measure(&name, seconds, count, false, || {
                let samples = (0..count)
                    .map(|i| {
                        HealthSample::heart_rate(format!("bench-{i}"), &cred.device_no, 76, at)
                    })
                    .collect();
                let request = BatchRequest::new(&cred, 1, samples);
                let mut bytes = Vec::with_capacity(count * 340 + 128);
                serde_json::to_writer(&mut bytes, &request).unwrap();
                bytes
            });
        }
        let name = format!("batch_{count}_borrowed_json");
        if enabled(&name) {
            let mut samples = Vec::with_capacity(count);
            let mut bytes = Vec::with_capacity(count * 400 + 200);
            measure(&name, seconds, count, false, || {
                samples.clear();
                bytes.clear();
                for i in 0..count {
                    samples.push(SampleView::heart_rate(
                        format!("bench-{i}"),
                        &cred.device_no,
                        76,
                        at,
                    ));
                }
                let request = BatchView {
                    device_no: &cred.device_no,
                    secret: &cred.secret,
                    binding_version: 1,
                    samples: &samples,
                };
                serde_json::to_writer(&mut bytes, &request).unwrap();
                black_box(&bytes);
                bytes.len()
            });
        }
        let name = format!("batch_{count}_cached_json");
        if enabled(&name) {
            let mut encoder = watch_sim::encoding::BatchEncoder::default();
            let mut samples = Vec::with_capacity(count);
            let mut bytes = Vec::with_capacity(count * 400 + 200);
            measure(&name, seconds, count, false, || {
                samples.clear();
                for i in 0..count {
                    samples.push(SampleView::heart_rate(
                        format!("bench-{i}"),
                        &cred.device_no,
                        76,
                        at,
                    ));
                }
                encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
                black_box(&bytes);
                bytes.len()
            });
        }
    }
    for cached in [false, true] {
        let name = if cached {
            "batch_2_mixed_cached_json"
        } else {
            "batch_2_mixed_borrowed_json"
        };
        if enabled(name) {
            let mut encoder = watch_sim::encoding::BatchEncoder::default();
            let mut bytes = Vec::with_capacity(1000);
            measure(name, seconds, 2, false, || {
                let samples = [
                    SampleView::heart_rate("run-1-hr".into(), &cred.device_no, 76, at),
                    SampleView::steps(
                        "run-1-steps".into(),
                        &cred.device_no,
                        800,
                        "2026-11-01T00:00:00Z",
                        at,
                    ),
                ];
                if cached {
                    encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
                } else {
                    bytes.clear();
                    serde_json::to_writer(
                        &mut bytes,
                        &BatchView {
                            device_no: &cred.device_no,
                            secret: &cred.secret,
                            binding_version: 1,
                            samples: &samples,
                        },
                    )
                    .unwrap();
                }
                black_box(&bytes);
                bytes.len()
            });
        }
    }
    for count in [2, 200] {
        for reused in [false, true] {
            let name = format!(
                "batch_{count}_ids_{}_json",
                if reused { "reused" } else { "owned" }
            );
            if !enabled(&name) {
                continue;
            }
            let prefix = String::from("0123456789abcdef0123456789abcdef-0");
            let mut sequence = 0u64;
            let mut ids: Vec<_> = (0..count).map(|_| String::with_capacity(64)).collect();
            let mut encoder = watch_sim::encoding::BatchEncoder::default();
            let mut bytes = Vec::with_capacity(count * 450 + 200);
            measure(&name, seconds, count, false, || {
                sequence += 1;
                if reused {
                    let mut samples = Vec::with_capacity(count);
                    for (i, id) in ids.iter_mut().enumerate() {
                        record_id_into(id, &prefix, sequence, itoa::Buffer::new().format(i));
                        samples.push(SampleRef::heart_rate(id, &cred.device_no, 76, at));
                    }
                    encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
                } else {
                    let samples: Vec<_> = (0..count)
                        .map(|i| {
                            SampleView::heart_rate(
                                format!("{prefix}-{sequence}-{i}"),
                                &cred.device_no,
                                76,
                                at,
                            )
                        })
                        .collect();
                    encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
                }
                black_box(&bytes);
                bytes.len()
            });
        }
    }
    for reused in [false, true] {
        let name = if reused {
            "batch_2_mixed_ids_reused_json"
        } else {
            "batch_2_mixed_ids_owned_json"
        };
        if !enabled(name) {
            continue;
        }
        let prefix = String::from("0123456789abcdef0123456789abcdef");
        let mut sequence = 0u64;
        let mut ids = [String::with_capacity(64), String::with_capacity(64)];
        let mut encoder = watch_sim::encoding::BatchEncoder::default();
        let mut bytes = Vec::with_capacity(1200);
        let at = "2026-11-02T08:00:00Z";
        let start = "2026-11-02T00:00:00Z";
        measure(name, seconds, 2, false, || {
            sequence += 1;
            if reused {
                for (id, suffix) in ids.iter_mut().zip(["hr", "steps"]) {
                    record_id_into(id, &prefix, sequence, suffix);
                }
                let samples = [
                    SampleRef::heart_rate(&ids[0], &cred.device_no, 76, at),
                    SampleRef::steps(&ids[1], &cred.device_no, 800, start, at),
                ];
                encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
            } else {
                let samples = [
                    SampleView::heart_rate(
                        format!("{prefix}-{sequence}-hr"),
                        &cred.device_no,
                        76,
                        at,
                    ),
                    SampleView::steps(
                        format!("{prefix}-{sequence}-steps"),
                        &cred.device_no,
                        800,
                        start,
                        at,
                    ),
                ];
                encoder.write(&mut bytes, &cred, 1, &samples).unwrap();
            }
            black_box(&bytes);
            bytes.len()
        });
    }
    for size in [512, 65536] {
        let name = format!("signature_{size}");
        if enabled(&name) {
            let payload = vec![b'a'; size];
            measure(&name, seconds, 1, false, || {
                signature_headers(
                    &cred,
                    black_box(&payload),
                    1793577600,
                    "0123456789abcdef0123456789abcdef".into(),
                )
            });
        }
        let name = format!("signature_{size}_prepared");
        if enabled(&name) {
            let payload = vec![b'a'; size];
            let signer = watch_sim::protocol::PreparedSigner::new(&cred);
            measure(&name, seconds, 1, false, || {
                signer.headers(
                    black_box(&payload),
                    1793577600,
                    "0123456789abcdef0123456789abcdef".into(),
                )
            });
        }
    }
    for signed in [false, true] {
        let name = if signed {
            "http_report_signed"
        } else {
            "http_report"
        };
        if !enabled(name) {
            continue;
        }
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let base = format!("http://{}", listener.local_addr().unwrap());
        let server = std::thread::spawn(move || {
            let (mut socket, _) = listener.accept().unwrap();
            socket.set_nodelay(true).unwrap();
            let reply=br#"{"returnCode":"1111","bound":1,"oldId":42,"bindingVersion":1,"rateState":0,"preState":0}"#;
            let header = format!(
                "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n",
                reply.len()
            );
            let mut pending = Vec::with_capacity(4096);
            let mut chunk = [0; 4096];
            loop {
                let n = socket.read(&mut chunk).unwrap();
                if n == 0 {
                    break;
                }
                pending.extend_from_slice(&chunk[..n]);
                let Some(end) = pending.windows(4).position(|w| w == b"\r\n\r\n") else {
                    continue;
                };
                let headers = String::from_utf8_lossy(&pending[..end]);
                let size: usize = headers
                    .lines()
                    .find_map(|line| {
                        line.to_ascii_lowercase()
                            .strip_prefix("content-length:")
                            .map(|n| n.trim().parse().unwrap())
                    })
                    .unwrap();
                if pending.len() < end + 4 + size {
                    continue;
                }
                let body: &[u8] = &pending[end + 4..end + 4 + size];
                let value: serde_json::Value = serde_json::from_slice(body).unwrap();
                assert_eq!(value["deviceNo"], "BENCH0001");
                if signed {
                    let field = |name: &str| {
                        headers
                            .lines()
                            .find_map(|line| {
                                line.split_once(':')
                                    .filter(|(key, _)| key.eq_ignore_ascii_case(name))
                                    .map(|(_, v)| v.trim())
                            })
                            .unwrap()
                    };
                    let actual = watch_sim::protocol::sign(
                        "test-only-0123456789abcdef01234567",
                        "BENCH0001",
                        field("X-Device-Timestamp").parse().unwrap(),
                        field("X-Device-Nonce"),
                        body,
                    );
                    assert_eq!(actual, field("X-Device-Signature"));
                }
                socket.write_all(header.as_bytes()).unwrap();
                socket.write_all(reply).unwrap();
                pending.drain(..end + 4 + size);
            }
        });
        let client = Arc::new(Client::new(&base, Duration::from_secs(5), signed));
        let mut session =
            DeviceSession::new(cred.clone(), client, Platform::Legacy, "2.0.0-sim".into());
        measure(name, seconds, 1, true, || session.report(&report).unwrap());
        drop(session);
        server.join().unwrap();
    }
}
