use crate::config::Credentials;
use crate::error::{CredentialKind, ToolError};
use crate::http;
use crate::tools;
use crate::tools::characters::Mismatch;
use crate::tools::invites::InviteReport;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Mutex;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tauri::async_runtime::{self, JoinHandle};
use tauri::{AppHandle, Emitter, Manager};

const INTERVAL: Duration = Duration::from_secs(5 * 60);

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Tool {
    Characters,
    Invites,
    Stash,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RunMode {
    Once,
    Continuous,
}

#[derive(Debug, Clone, Copy, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum LogLevel {
    Info,
    Warning,
    Error,
}

#[derive(Debug, Clone, Serialize)]
#[serde(tag = "state", rename_all = "snake_case")]
pub enum Status {
    Running,
    Waiting { next_run_at: u64 },
    Finished,
    Stopped,
    Failed {
        error: String,
        credential: Option<CredentialKind>,
    },
}

#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum ToolEvent {
    Status { status: Status },
    Log { level: LogLevel, message: String, time: u64 },
    Progress {
        percent: Option<u32>,
        current_time: i64,
        rate_limit: String,
        uploaded_entries: u64,
    },
    CharacterReport { mismatches: Vec<Mismatch>, time: u64 },
    InviteReport { report: InviteReport, time: u64 },
}

#[derive(Debug, Clone, Serialize)]
struct Envelope {
    tool: Tool,
    event: ToolEvent,
}

pub fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |elapsed| elapsed.as_millis() as u64)
}

/// What a running tool needs from the app: HTTP, credentials and reporting.
pub struct Ctx {
    app: AppHandle,
    tool: Tool,
    mode: RunMode,
    pub http: Client,
}

impl Ctx {
    pub fn emit(&self, event: ToolEvent) {
        let envelope = Envelope {
            tool: self.tool,
            event,
        };
        if let Err(error) = self.app.emit("tool-event", envelope) {
            eprintln!("could not emit tool event: {error}");
        }
    }

    pub fn log(&self, level: LogLevel, message: impl Into<String>) {
        self.emit(ToolEvent::Log {
            level,
            message: message.into(),
            time: now_ms(),
        });
    }

    pub fn info(&self, message: impl Into<String>) {
        self.log(LogLevel::Info, message);
    }

    pub fn warn(&self, message: impl Into<String>) {
        self.log(LogLevel::Warning, message);
    }

    pub fn credential(&self, kind: CredentialKind) -> Result<String, ToolError> {
        self.app.state::<Credentials>().get(kind).ok_or_else(|| {
            ToolError::credential(kind, format!("{} is not set", kind.display_name()))
        })
    }

    /// Handles the outcome of one pass and, in continuous mode, sleeps until
    /// the next one. Returns whether another pass should run.
    pub async fn next_pass(&self, result: Result<(), ToolError>) -> Result<bool, ToolError> {
        if self.mode == RunMode::Once {
            return result.map(|()| false);
        }
        match result {
            Ok(()) => {}
            Err(error @ (ToolError::Credential { .. } | ToolError::Fatal(_))) => return Err(error),
            Err(error) => self.log(
                LogLevel::Error,
                format!("{error} - trying again on the next interval"),
            ),
        }
        self.emit(ToolEvent::Status {
            status: Status::Waiting {
                next_run_at: now_ms() + INTERVAL.as_millis() as u64,
            },
        });
        tokio::time::sleep(INTERVAL).await;
        self.emit(ToolEvent::Status {
            status: Status::Running,
        });
        Ok(true)
    }
}

struct Running {
    id: u64,
    handle: JoinHandle<()>,
}

#[derive(Default)]
pub struct Jobs {
    running: Mutex<HashMap<Tool, Running>>,
    next_id: AtomicU64,
}

impl Jobs {
    pub fn start(&self, app: &AppHandle, tool: Tool, mode: RunMode) -> Result<(), String> {
        let mut running = self.running.lock().unwrap();
        if running.contains_key(&tool) {
            return Err("This tool is already running".to_string());
        }
        let ctx = Ctx {
            app: app.clone(),
            tool,
            mode,
            http: http::client().map_err(|error| error.to_string())?,
        };
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        ctx.emit(ToolEvent::Status {
            status: Status::Running,
        });
        let handle = async_runtime::spawn(async move {
            let result = tools::run(&ctx, tool).await;
            // stop() already removed the entry and reported the run as stopped
            let jobs = ctx.app.state::<Jobs>();
            if !jobs.finish(tool, id) {
                return;
            }
            let status = match result {
                Ok(()) => Status::Finished,
                Err(error) => {
                    let credential = match &error {
                        ToolError::Credential { kind, .. } => Some(*kind),
                        _ => None,
                    };
                    Status::Failed {
                        error: error.to_string(),
                        credential,
                    }
                }
            };
            ctx.emit(ToolEvent::Status { status });
        });
        running.insert(tool, Running { id, handle });
        Ok(())
    }

    pub fn stop(&self, app: &AppHandle, tool: Tool) {
        let Some(job) = self.running.lock().unwrap().remove(&tool) else {
            return;
        };
        job.handle.abort();
        let envelope = Envelope {
            tool,
            event: ToolEvent::Status {
                status: Status::Stopped,
            },
        };
        if let Err(error) = app.emit("tool-event", envelope) {
            eprintln!("could not emit tool event: {error}");
        }
    }

    fn finish(&self, tool: Tool, id: u64) -> bool {
        let mut running = self.running.lock().unwrap();
        if running.get(&tool).is_some_and(|job| job.id == id) {
            running.remove(&tool);
            return true;
        }
        false
    }
}
