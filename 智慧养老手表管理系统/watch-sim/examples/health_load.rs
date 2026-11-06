//! Finite, authenticated database workload. Every acknowledged sample must be counted in the database by the caller.
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::collections::BTreeMap;
use std::io::Read;
use std::path::PathBuf;
use std::{
    sync::{Arc, Barrier},
    time::{Duration, Instant},
};
use watch_sim::{
    device::DeviceSession,
    protocol::{record_id_into, Credentials, Platform, SampleRef},
    transport::Client,
};

#[derive(Deserialize)]
struct Device {
    device_no: String,
    secret: String,
}
#[derive(Deserialize)]
struct Config {
    server: String,
    devices: Vec<Device>,
    requests_per_device: usize,
    batch_size: usize,
    signed: bool,
    #[serde(default)]
    warmup_per_device: usize,
    #[serde(default)]
    trace_file: Option<PathBuf>,
    #[serde(default)]
    metrics_token: Option<String>,
}

fn pool_metrics(server: &str, token: &str) -> Result<BTreeMap<String, f64>, String> {
    let tls = native_tls::TlsConnector::new().map_err(|e| e.to_string())?;
    let agent = ureq::AgentBuilder::new()
        .timeout(Duration::from_secs(5))
        .redirects(0)
        .tls_connector(Arc::new(tls))
        .build();
    let response = agent
        .get(&format!("{}/metrics", server.trim_end_matches('/')))
        .set("Authorization", &format!("Bearer {token}"))
        .call()
        .map_err(|e| format!("连接池指标读取失败：{e}"))?;
    let mut bytes = Vec::new();
    response
        .into_reader()
        .take(1_048_577)
        .read_to_end(&mut bytes)
        .map_err(|e| e.to_string())?;
    if bytes.len() > 1_048_576 {
        return Err("连接池指标超过 1 MiB".into());
    }
    let text = std::str::from_utf8(&bytes).map_err(|e| e.to_string())?;
    let mut metrics = BTreeMap::new();
    for line in text
        .lines()
        .filter(|line| line.starts_with("suining_mysql_pool_"))
    {
        let mut fields = line.split_whitespace();
        let name = fields.next().ok_or("连接池指标名称缺失")?;
        let value = fields
            .next()
            .ok_or("连接池指标数值缺失")?
            .parse()
            .map_err(|_| "连接池指标数值无效")?;
        metrics.insert(name.to_owned(), value);
    }
    if !metrics.contains_key("suining_mysql_pool_wait_total") {
        return Err("后端未提供连接池等待指标".into());
    }
    Ok(metrics)
}

#[derive(Serialize)]
struct Observation {
    worker: usize,
    sequence: usize,
    warmup: bool,
    started_us: u128,
    duration_us: u128,
    success: bool,
}

fn run() -> Result<(), String> {
    let path = std::env::args().nth(1).ok_or("需要压测配置文件路径")?;
    let config: Config = serde_json::from_slice(&std::fs::read(path).map_err(|e| e.to_string())?)
        .map_err(|e| e.to_string())?;
    if config.devices.is_empty()
        || config.requests_per_device == 0
        || !(1..=200).contains(&config.batch_size)
    {
        return Err("需要设备、非零请求数及 1–200 条批次".into());
    }
    let mut sessions = Vec::with_capacity(config.devices.len());
    for d in config.devices {
        let client = Arc::new(Client::with_ca(
            &config.server,
            Duration::from_secs(30),
            config.signed,
            None,
        )?);
        let mut session = DeviceSession::new(
            Credentials::new(d.device_no, d.secret),
            client,
            Platform::HarmonyOs,
            "2.0.0-perf".into(),
        );
        session.login().map_err(|e| e.message())?;
        if !session.is_bound() {
            return Err("压测设备必须已经绑定".into());
        }
        sessions.push(session);
    }
    let concurrency = sessions.len();
    let barrier = Arc::new(Barrier::new(concurrency + 1));
    let ready = Arc::new(Barrier::new(concurrency + 1));
    let zero = Instant::now();
    let tracing = config.trace_file.is_some();
    let run_id = uuid::Uuid::new_v4().simple().to_string();
    let mut workers = Vec::with_capacity(concurrency);
    for (worker, mut session) in sessions.into_iter().enumerate() {
        let barrier = barrier.clone();
        let ready = ready.clone();
        let warmup = config.warmup_per_device;
        let prefix = format!("{run_id}-{worker}");
        let requests = config.requests_per_device;
        let batch_size = config.batch_size;
        workers.push(std::thread::spawn(move || {
            let device = session.credentials().device_no.clone();
            let mut latencies = Vec::with_capacity(requests);
            let mut failures = Vec::new();
            let mut warmup_success = 0;
            let mut warmup_failed = 0;
            let mut observations = Vec::new();
            let mut record_ids: Vec<_> =
                (0..batch_size).map(|_| String::with_capacity(64)).collect();
            barrier.wait();
            for sequence in 0..warmup + requests {
                if sequence == warmup {
                    ready.wait();
                    barrier.wait();
                }
                let now = watch_sim::format_utc(watch_sim::now_unix());
                let mut samples = Vec::with_capacity(batch_size);
                for (i, id) in record_ids.iter_mut().enumerate() {
                    record_id_into(id, &prefix, sequence as u64, itoa::Buffer::new().format(i));
                    samples.push(SampleRef::heart_rate(id, &device, 76, &now));
                }
                let start = Instant::now();
                let result = session.send_sample_views(&samples);
                let duration = start.elapsed();
                let successful = result.is_ok();
                match result {
                    Ok(_) if sequence < warmup => warmup_success += 1,
                    Ok(_) => latencies.push(duration.as_secs_f64() * 1000.0),
                    Err(e) => {
                        if sequence < warmup {
                            warmup_failed += 1;
                        }
                        failures.push(e.message());
                    }
                }
                if tracing {
                    observations.push(Observation {
                        worker,
                        sequence,
                        warmup: sequence < warmup,
                        started_us: start.duration_since(zero).as_micros(),
                        duration_us: duration.as_micros(),
                        success: successful,
                    });
                }
            }
            (
                latencies,
                failures,
                warmup_success,
                warmup_failed,
                observations,
            )
        }));
    }
    barrier.wait();
    ready.wait();
    let measured_pool_before = config
        .metrics_token
        .as_deref()
        .map(|token| pool_metrics(&config.server, token))
        .transpose()?;
    let start = Instant::now();
    let measured_start_unix_us = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_micros();
    barrier.wait();
    let mut latencies = Vec::new();
    let mut failures = Vec::new();
    let mut warmup_success = 0;
    let mut warmup_failed = 0;
    let mut observations = Vec::new();
    for worker in workers {
        let (mut l, mut f, w, wf, mut trace) = worker.join().map_err(|_| "压测线程异常")?;
        latencies.append(&mut l);
        failures.append(&mut f);
        warmup_success += w;
        warmup_failed += wf;
        observations.append(&mut trace);
    }
    let end = Instant::now();
    let measured_end_unix_us = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_err(|e| e.to_string())?
        .as_micros();
    let elapsed = end.duration_since(start).as_secs_f64();
    let measured_pool_after = config
        .metrics_token
        .as_deref()
        .map(|token| pool_metrics(&config.server, token))
        .transpose()?;
    if let Some(path) = config.trace_file {
        std::fs::write(
            path,
            serde_json::to_vec(&observations).map_err(|e| e.to_string())?,
        )
        .map_err(|e| e.to_string())?;
    }
    latencies.sort_by(f64::total_cmp);
    let percentile = |p: f64| {
        latencies
            .get(((latencies.len().saturating_sub(1)) as f64 * p).ceil() as usize)
            .copied()
    };
    println!(
        "{}",
        json!({"concurrency":concurrency,"batch_size":config.batch_size,"requests":concurrency*config.requests_per_device,
        "warmup_success":warmup_success,"warmup_samples":warmup_success*config.batch_size,"success":latencies.len(),"failed":failures.len()-warmup_failed,"warmup_failed":warmup_failed,"successful_samples":latencies.len()*config.batch_size,"seconds":elapsed,
        "samples_per_second":latencies.len() as f64*config.batch_size as f64/elapsed,
        "measured_start_us":start.duration_since(zero).as_micros(),"measured_end_us":end.duration_since(zero).as_micros(),
        "measured_start_unix_us":measured_start_unix_us,"measured_end_unix_us":measured_end_unix_us,
        "measured_pool_before":measured_pool_before,"measured_pool_after":measured_pool_after,
        "p50_ms":percentile(0.5),"p95_ms":percentile(0.95),"p99_ms":percentile(0.99),"errors":failures})
    );
    if failures.is_empty() {
        Ok(())
    } else {
        Err("存在投递失败，结果中已列出".into())
    }
}
fn main() {
    if let Err(e) = run() {
        eprintln!("{e}");
        std::process::exit(1);
    }
}
