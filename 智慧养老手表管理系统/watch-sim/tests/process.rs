use std::{
    io::{Read, Write},
    net::TcpListener,
    process::Command,
};

fn binary() -> Command {
    Command::new(env!("CARGO_BIN_EXE_watch-sim"))
}

#[derive(Clone, Copy)]
enum AddressMode {
    Explicit,
    LiveDiscovery,
    ExplicitOverStale,
}

fn once_report(signed: bool, address_mode: AddressMode) {
    let server = TcpListener::bind("127.0.0.1:0").unwrap();
    let address = format!("http://{}", server.local_addr().unwrap());
    let port = server.local_addr().unwrap().port();
    let worker = std::thread::spawn(move || {
        let mut requests = Vec::new();
        let mut nonces = std::collections::HashSet::new();
        for _ in 0..3 {
            let (mut stream, _) = server.accept().unwrap();
            stream
                .set_read_timeout(Some(std::time::Duration::from_secs(5)))
                .unwrap();
            let mut bytes = Vec::new();
            let mut chunk = [0; 4096];
            loop {
                let n = stream.read(&mut chunk).unwrap();
                assert!(n > 0);
                bytes.extend_from_slice(&chunk[..n]);
                if let Some(end) = bytes.windows(4).position(|v| v == b"\r\n\r\n") {
                    let header = String::from_utf8_lossy(&bytes[..end]);
                    let size: usize = header
                        .lines()
                        .find_map(|l| {
                            l.to_ascii_lowercase()
                                .strip_prefix("content-length:")
                                .map(|v| v.trim().parse().unwrap())
                        })
                        .unwrap();
                    if bytes.len() >= end + 4 + size {
                        break;
                    }
                }
            }
            let end = bytes.windows(4).position(|v| v == b"\r\n\r\n").unwrap();
            let headers = String::from_utf8_lossy(&bytes[..end]);
            let field = |name: &str| {
                headers.lines().find_map(|line| {
                    line.split_once(':')
                        .filter(|(key, _)| key.eq_ignore_ascii_case(name))
                        .map(|(_, value)| value.trim())
                })
            };
            if signed || headers.starts_with("POST /api/v1/device/report ") {
                let nonce = field("X-Device-Nonce").unwrap();
                assert!(nonces.insert(nonce.to_owned()));
                assert_eq!(
                    field("X-Device-Signature").unwrap(),
                    watch_sim::protocol::sign(
                        "test-only-secret",
                        "TEST0001",
                        field("X-Device-Timestamp").unwrap().parse().unwrap(),
                        nonce,
                        &bytes[end + 4..]
                    )
                );
            } else {
                for name in ["X-Device-Nonce", "X-Device-Signature", "X-Device-Timestamp"] {
                    assert!(field(name).is_none());
                }
            }
            let body: serde_json::Value = serde_json::from_slice(&bytes[end + 4..]).unwrap();
            requests.push((String::from_utf8_lossy(&bytes[..end]).to_string(), body));
            let reply = r#"{"returnCode":"1111","bound":1,"oldId":42,"bindingVersion":3}"#;
            write!(
                stream,
                "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                reply.len(),
                reply
            )
            .unwrap();
            stream.flush().unwrap();
        }
        requests
    });
    let config = std::env::temp_dir().join(format!("watch-config-{}.json", uuid::Uuid::new_v4()));
    std::fs::write(&config,r#"{"secret":"","interval":17,"scenario":"normal","longitude":1.0,"latitude":2.0,"app_version":"from-file"}"#).unwrap();
    let mut command = binary();
    if signed {
        command.arg("--sign");
    }
    let discovery =
        std::env::temp_dir().join(format!("watch-live-discovery-{}", uuid::Uuid::new_v4()));
    if !matches!(address_mode, AddressMode::Explicit) {
        std::fs::create_dir_all(discovery.join("runtime")).unwrap();
        let pid = if matches!(address_mode, AddressMode::LiveDiscovery) {
            std::process::id()
        } else {
            0
        };
        std::fs::write(
            discovery.join("runtime/server.json"),
            serde_json::to_vec(&serde_json::json!({"port":port,"pid":pid,"desktopPid":pid}))
                .unwrap(),
        )
        .unwrap();
        command.env("SUINING_USER_DATA", &discovery);
    }
    if !matches!(address_mode, AddressMode::LiveDiscovery) {
        command.args(["--server", &address]);
    }
    let output = command
        .env("WATCH_DEVICE_SECRET", "test-only-secret")
        .args(["--config", config.to_str().unwrap()])
        .args([
            "--device-no",
            "test0001",
            "--once",
            "--scenario",
            "normal",
            "--app-version",
            "test-custom",
        ])
        .output()
        .unwrap();
    std::fs::remove_file(config).unwrap();
    if !matches!(address_mode, AddressMode::Explicit) {
        std::fs::remove_file(discovery.join("runtime/server.json")).unwrap();
        std::fs::remove_dir(discovery.join("runtime")).unwrap();
        std::fs::remove_dir(discovery).unwrap();
    }
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
    let requests = worker.join().unwrap();
    assert!(requests[0].0.starts_with("POST /api/v1/device/login "));
    assert!(requests[1].0.starts_with("POST /api/v1/device/heartbeat "));
    assert!(requests[2].0.starts_with("POST /api/v1/device/report "));
    let body = &requests[2].1;
    for field in [
        "hr",
        "bp",
        "step",
        "sleep",
        "sleepState",
        "lng",
        "lat",
        "source",
        "battery",
        "appVersion",
    ] {
        assert!(body.get(field).is_some(), "missing {field}");
    }
    assert!(body.get("beTime").is_none());
    assert_eq!(body["bindingVersion"], 3);
    assert_eq!(body["positionSource"], "simulated");
    assert_eq!(body["appVersion"], "test-custom");
    assert_eq!(body["deviceNo"], "TEST0001");
    assert_eq!(body["sleepState"], "0");
    let lng: f64 = body["lng"].as_str().unwrap().parse().unwrap();
    let lat: f64 = body["lat"].as_str().unwrap().parse().unwrap();
    assert!((lng - 1.0).abs() < 0.01 && (lat - 2.0).abs() < 0.01);
}

#[test]
fn once_preserves_full_report_and_config_overrides() {
    once_report(false, AddressMode::Explicit);
}

#[test]
fn once_signs_login_heartbeat_and_report_with_unique_nonces() {
    once_report(true, AddressMode::Explicit);
}

#[test]
fn invalid_discovery_fails_before_network() {
    let dir = std::env::temp_dir().join(format!("watch-discovery-{}", uuid::Uuid::new_v4()));
    std::fs::create_dir_all(dir.join("runtime")).unwrap();
    std::fs::write(
        dir.join("runtime/server.json"),
        r#"{"port":8795,"pid":0,"desktopPid":0}"#,
    )
    .unwrap();
    let result = binary()
        .env("SUINING_USER_DATA", &dir)
        .args(["--once", "--secret", "test-only-secret"])
        .output()
        .unwrap();
    std::fs::remove_file(dir.join("runtime/server.json")).unwrap();
    std::fs::remove_dir(dir.join("runtime")).unwrap();
    std::fs::remove_dir(dir).unwrap();
    assert!(!result.status.success());
    assert!(String::from_utf8_lossy(&result.stderr).contains("端口记录已失效"));
}

#[test]
fn sos_requires_explicit_authorization() {
    let result = binary()
        .args([
            "--once",
            "--scenario",
            "sos",
            "--secret",
            "test-only-secret",
        ])
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(String::from_utf8_lossy(&result.stderr).contains("--allow-sos"));
}

#[test]
fn local_mode_runs_without_credentials_or_network_and_keeps_each_run() {
    let dir = std::env::temp_dir().join(format!("watch-local-{}", uuid::Uuid::new_v4()));
    let server = TcpListener::bind("127.0.0.1:0").unwrap();
    server.set_nonblocking(true).unwrap();
    let address = format!("http://{}", server.local_addr().unwrap());
    for scene in ["normal", "sos", "fall"] {
        let output = binary()
            .env_remove("WATCH_DEVICE_SECRET")
            .args([
                "--local",
                dir.to_str().unwrap(),
                "--once",
                "--scenario",
                scene,
                "--health-samples",
                "--server",
                &address,
            ])
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
    }
    assert_eq!(
        server.accept().unwrap_err().kind(),
        std::io::ErrorKind::WouldBlock,
        "local mode contacted a server"
    );
    let runs: Vec<_> = std::fs::read_dir(&dir)
        .unwrap()
        .map(|e| e.unwrap().path())
        .collect();
    assert_eq!(runs.len(), 3, "each run must retain its own output");
    let mut kinds = Vec::new();
    for run in &runs {
        let latest: serde_json::Value =
            serde_json::from_slice(&std::fs::read(run.join("latest.json")).unwrap()).unwrap();
        assert_eq!(latest["mode"], "local");
        assert_eq!(latest["sampleCount"], 1);
        assert!(latest["data"].get("secret").is_none());
        for field in [
            "hr",
            "bp",
            "step",
            "sleep",
            "sleepState",
            "lng",
            "lat",
            "source",
            "battery",
            "appVersion",
            "beTime",
        ] {
            assert!(latest["data"].get(field).is_some(), "missing {field}");
        }
        let records = std::fs::read_to_string(run.join("records.jsonl")).unwrap();
        assert!(!records.contains("secret"));
        let rows: Vec<serde_json::Value> = records
            .lines()
            .map(|line| serde_json::from_str(line).unwrap())
            .collect();
        assert!(rows.iter().any(
            |row| row["type"] == "healthBatch" && row["samples"].as_array().unwrap().len() == 2
        ));
        let html = std::fs::read_to_string(run.join("index.html")).unwrap();
        assert!(html.contains("本地运行"));
        let events = std::fs::read_to_string(run.join("events.jsonl")).unwrap();
        for row in events.lines() {
            let event: serde_json::Value = serde_json::from_str(row).unwrap();
            kinds.push(event["kind"].as_str().unwrap().to_owned());
        }
        // Every path was created inside this unique test directory.
        for file in std::fs::read_dir(run).unwrap() {
            std::fs::remove_file(file.unwrap().path()).unwrap();
        }
        std::fs::remove_dir(run).unwrap();
    }
    assert!(kinds.contains(&"sos".to_owned()) && kinds.contains(&"fall".to_owned()));
    std::fs::remove_dir(dir).unwrap();
}

#[test]
fn local_mode_continuously_replaces_snapshots_without_exporting_secrets() {
    use std::io::{BufRead, BufReader};
    let dir = std::env::temp_dir().join(format!("watch 本地 数据 {}", uuid::Uuid::new_v4()));
    let mut child = binary()
        .args([
            "--local",
            dir.to_str().unwrap(),
            "--interval",
            "1",
            "--scenario",
            "normal",
            "--health-samples",
            "--secret",
            "test-only-must-stay-private",
        ])
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .spawn()
        .unwrap();
    let mut reader = BufReader::new(child.stdout.take().unwrap());
    let mut count = 0;
    loop {
        let mut line = String::new();
        if reader.read_line(&mut line).unwrap() == 0 {
            break;
        }
        if line.starts_with("本地 normal") {
            count += 1;
        }
        if count == 2 {
            break;
        }
    }
    child.kill().unwrap();
    child.wait().unwrap();
    assert_eq!(count, 2);
    let run = std::fs::read_dir(&dir)
        .unwrap()
        .next()
        .unwrap()
        .unwrap()
        .path();
    let latest: serde_json::Value =
        serde_json::from_slice(&std::fs::read(run.join("latest.json")).unwrap()).unwrap();
    assert!(latest["sampleCount"].as_u64().unwrap() >= 2);
    let records = std::fs::read_to_string(run.join("records.jsonl")).unwrap();
    assert!(records.lines().count() >= 2);
    let mut ids = std::collections::HashSet::new();
    let mut batches = 0;
    for line in records.lines() {
        let row: serde_json::Value = serde_json::from_str(line).unwrap();
        if row["type"] != "healthBatch" {
            continue;
        }
        batches += 1;
        for sample in row["samples"].as_array().unwrap() {
            let id = sample["recordId"].as_str().unwrap();
            assert!(watch_sim::protocol::request_id_ok(id));
            assert!(
                ids.insert(id.to_owned()),
                "record ID reused across samples: {id}"
            );
            let suffix = if sample["metric"] == "heart_rate" {
                "hr"
            } else {
                "steps"
            };
            assert!(id.ends_with(&format!("-{batches}-{suffix}")));
        }
    }
    assert!(batches >= 2);
    assert_eq!(ids.len(), batches * 2);
    for file in std::fs::read_dir(&run).unwrap() {
        let path = file.unwrap().path();
        assert!(!std::fs::read_to_string(&path)
            .unwrap()
            .contains("test-only-must-stay-private"));
        std::fs::remove_file(path).unwrap();
    }
    std::fs::remove_dir(run).unwrap();
    std::fs::remove_dir(dir).unwrap();
}

#[test]
fn fall_requires_explicit_authorization() {
    let result = binary()
        .args([
            "--once",
            "--scenario",
            "fall",
            "--secret",
            "test-only-secret",
        ])
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(String::from_utf8_lossy(&result.stderr).contains("--allow-sos"));
}

fn event_retry(scene: &str, health_samples: bool) {
    let server = TcpListener::bind("127.0.0.1:0").unwrap();
    let address = format!("http://{}", server.local_addr().unwrap());
    server.set_nonblocking(true).unwrap();
    let server_scene = scene.to_owned();
    let handle = std::thread::spawn(move || {
        let scene = server_scene;
        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(12);
        let mut events = Vec::new();
        let mut nonces = std::collections::HashSet::new();
        let mut ordinary_after_ack = false;
        while std::time::Instant::now() < deadline {
            let Ok((mut stream, _)) = server.accept() else {
                std::thread::sleep(std::time::Duration::from_millis(10));
                continue;
            };
            stream
                .set_read_timeout(Some(std::time::Duration::from_secs(2)))
                .unwrap();
            let mut bytes = Vec::new();
            let end;
            loop {
                let mut chunk = [0; 2048];
                let n = stream.read(&mut chunk).unwrap();
                if n == 0 {
                    panic!("truncated request");
                }
                bytes.extend_from_slice(&chunk[..n]);
                if let Some(pos) = bytes.windows(4).position(|v| v == b"\r\n\r\n") {
                    let size: usize = String::from_utf8_lossy(&bytes[..pos])
                        .lines()
                        .find_map(|line| {
                            line.to_ascii_lowercase()
                                .strip_prefix("content-length:")
                                .map(|v| v.trim().parse().unwrap())
                        })
                        .unwrap();
                    if bytes.len() >= pos + 4 + size {
                        end = pos;
                        break;
                    }
                }
            }
            let headers = String::from_utf8_lossy(&bytes[..end]);
            let body: serde_json::Value = serde_json::from_slice(&bytes[end + 4..]).unwrap();
            let event = if scene == "fall" {
                body["alarm"] == "fall"
            } else {
                headers.starts_with("POST /api/v1/device/sos ")
            };
            if event {
                assert_eq!(body["bindingVersion"], 3);
                assert_eq!(body["positionSource"], "simulated");
                assert!(body["requestId"].as_str().is_some_and(|v| !v.is_empty()));
                let nonce = headers
                    .lines()
                    .find_map(|line| {
                        line.split_once(':')
                            .filter(|(k, _)| k.eq_ignore_ascii_case("X-Device-Nonce"))
                            .map(|(_, v)| v.trim().to_owned())
                    })
                    .unwrap();
                assert!(nonces.insert(nonce));
                events.push(body.clone());
                if events.len() == 1 {
                    continue;
                } // lost acknowledgement
            } else if events.len() == 3 && headers.starts_with("POST /api/v1/device/report ") {
                assert!(body.get("alarm").is_none());
                ordinary_after_ack = true;
            }
            let reply = if event && events.len() == 2 {
                r#"{"returnCode":"0000","message":"temporary rejection"}"#
            } else {
                r#"{"returnCode":"1111","bound":1,"oldId":42,"bindingVersion":3,"incidentId":7,"accepted":2,"duplicates":0}"#
            };
            write!(
                stream,
                "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                reply.len(),
                reply
            )
            .unwrap();
            stream.flush().unwrap();
            if ordinary_after_ack {
                break;
            }
        }
        (events, ordinary_after_ack)
    });
    let mut command = binary();
    if health_samples {
        command.arg("--health-samples");
    }
    let mut child = command
        .args([
            "--server",
            &address,
            "--device-no",
            "TEST0001",
            "--secret",
            "test-only-secret",
            "--scenario",
            scene,
            "--interval",
            "1",
            "--allow-sos",
            "--sign",
        ])
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .spawn()
        .unwrap();
    let result = handle.join();
    let _ = child.kill();
    let _ = child.wait();
    let (events, resumed) = result.unwrap();
    assert_eq!(events.len(), 3);
    assert_eq!(events[0], events[1]);
    assert_eq!(events[1], events[2]);
    assert!(
        resumed,
        "normal telemetry must resume after the event acknowledgement"
    );
}

#[test]
fn fall_retries_original_event_until_ack_then_stops_repeating() {
    event_retry("fall", false);
}

#[test]
fn sos_retries_original_event_until_ack_then_stops_repeating() {
    event_retry("sos", false);
}

#[test]
fn pending_care_survives_optional_health_uploads() {
    event_retry("fall", true);
}

#[test]
fn live_desktop_discovery_delivers_the_full_report() {
    once_report(false, AddressMode::LiveDiscovery);
}

#[test]
fn explicit_address_overrides_a_stale_desktop_record() {
    once_report(false, AddressMode::ExplicitOverStale);
}
