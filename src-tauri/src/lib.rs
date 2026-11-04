mod config;
mod error;
mod http;
mod jobs;
mod tools;

use config::{CredentialStatus, Credentials};
use error::CredentialKind;
use jobs::{Jobs, RunMode, Tool};
use tauri::{AppHandle, Manager, State};

// Credential values only ever flow into the backend. The webview can see
// whether a value is set, but never reads it back.
#[tauri::command]
fn credential_status(credentials: State<Credentials>) -> CredentialStatus {
    credentials.status()
}

#[tauri::command]
fn save_credential(
    credentials: State<Credentials>,
    kind: CredentialKind,
    value: String,
) -> Result<(), String> {
    credentials.save(kind, &value)
}

#[tauri::command]
fn start_tool(
    app: AppHandle,
    jobs: State<Jobs>,
    tool: Tool,
    mode: RunMode,
) -> Result<(), String> {
    jobs.start(&app, tool, mode)
}

#[tauri::command]
fn stop_tool(app: AppHandle, jobs: State<Jobs>, tool: Tool) {
    jobs.stop(&app, tool)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .manage(Jobs::default())
        .setup(|app| {
            let config_dir = app.path().app_config_dir()?;
            app.manage(Credentials::load(config_dir)?);
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            credential_status,
            save_credential,
            start_tool,
            stop_tool
        ])
        .run(tauri::generate_context!())
        .expect("error while running BPL Tools");
}
