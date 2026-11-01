use crate::error::CredentialKind;
use keyring::{Entry, Error as KeyringError};
use serde::Serialize;
use std::collections::HashMap;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::sync::Mutex;

const CONFIG_FILE: &str = "bpl-config.txt";
const KEYRING_SERVICE: &str = "com.bpl-poe.tools";
const KINDS: [CredentialKind; 2] = [CredentialKind::PoeSession, CredentialKind::BplToken];

#[derive(Debug, Serialize)]
pub struct CredentialStatus {
    poe_session: bool,
    bpl_token: bool,
    /// Set when no system keychain is available and values are kept in this file
    plaintext_path: Option<String>,
}

trait SecretStore: Send + Sync {
    fn get(&self, key: &str) -> Result<Option<String>, String>;
    fn set(&self, key: &str, value: &str) -> Result<(), String>;
}

/// macOS Keychain, Windows Credential Manager or the Linux Secret Service.
struct KeyringStore {
    service: &'static str,
}

impl SecretStore for KeyringStore {
    fn get(&self, key: &str) -> Result<Option<String>, String> {
        let entry = Entry::new(self.service, key).map_err(|error| error.to_string())?;
        match entry.get_password() {
            Ok(value) => Ok(Some(value)),
            Err(KeyringError::NoEntry) => Ok(None),
            Err(error) => Err(error.to_string()),
        }
    }

    fn set(&self, key: &str, value: &str) -> Result<(), String> {
        Entry::new(self.service, key)
            .and_then(|entry| entry.set_password(value))
            .map_err(|error| error.to_string())
    }
}

/// Fallback for systems without a keychain, e.g. Linux without a Secret Service.
struct FileStore {
    path: PathBuf,
}

impl SecretStore for FileStore {
    fn get(&self, key: &str) -> Result<Option<String>, String> {
        match fs::read_to_string(&self.path) {
            Ok(content) => Ok(parse_config(&content).remove(key)),
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
            Err(error) => Err(error.to_string()),
        }
    }

    fn set(&self, key: &str, value: &str) -> Result<(), String> {
        let existing = fs::read_to_string(&self.path).unwrap_or_default();
        write_private(&self.path, &update_config(&existing, key, value))
            .map_err(|error| format!("Could not write {}: {error}", self.path.display()))
    }
}

pub struct Credentials {
    store: Box<dyn SecretStore>,
    plaintext_path: Option<PathBuf>,
    values: Mutex<HashMap<CredentialKind, String>>,
}

impl Credentials {
    pub fn load(config_dir: PathBuf) -> io::Result<Self> {
        fs::create_dir_all(&config_dir)?;
        let path = config_dir.join(CONFIG_FILE);
        let legacy = legacy_config_files(&path);
        let keyring = KeyringStore {
            service: KEYRING_SERVICE,
        };
        match keyring.get(CredentialKind::PoeSession.config_key()) {
            Ok(_) => {
                let mut files = vec![path];
                files.extend(legacy);
                migrate(&keyring, &files);
                Ok(Self::open(Box::new(keyring), None))
            }
            Err(error) => {
                eprintln!("no system keychain available: {error}");
                let store = FileStore { path: path.clone() };
                migrate(&store, &legacy);
                Ok(Self::open(Box::new(store), Some(path)))
            }
        }
    }

    fn open(store: Box<dyn SecretStore>, plaintext_path: Option<PathBuf>) -> Self {
        let mut values = HashMap::new();
        for kind in KINDS {
            match store.get(kind.config_key()) {
                Ok(Some(value)) if !value.is_empty() => {
                    values.insert(kind, value);
                }
                Ok(_) => {}
                Err(error) => eprintln!("could not read {}: {error}", kind.config_key()),
            }
        }
        Self {
            store,
            plaintext_path,
            values: Mutex::new(values),
        }
    }

    pub fn get(&self, kind: CredentialKind) -> Option<String> {
        self.values.lock().unwrap().get(&kind).cloned()
    }

    pub fn status(&self) -> CredentialStatus {
        CredentialStatus {
            poe_session: self.get(CredentialKind::PoeSession).is_some(),
            bpl_token: self.get(CredentialKind::BplToken).is_some(),
            plaintext_path: self
                .plaintext_path
                .as_ref()
                .map(|path| path.display().to_string()),
        }
    }

    pub fn save(&self, kind: CredentialKind, value: &str) -> Result<(), String> {
        let value = value.trim();
        if value.is_empty() {
            return Err(format!("{} must not be empty", kind.display_name()));
        }
        if value.chars().any(char::is_control) {
            return Err(format!("{} contains invalid characters", kind.display_name()));
        }
        let mut values = self.values.lock().unwrap();
        self.store.set(kind.config_key(), value)?;
        values.insert(kind, value.to_string());
        Ok(())
    }
}

/// Where earlier versions kept their config: the CLI next to the binary.
fn legacy_config_files(config_path: &Path) -> Vec<PathBuf> {
    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|exe| exe.parent().map(Path::to_path_buf));
    let cwd = std::env::current_dir().ok();
    let mut files: Vec<PathBuf> = [exe_dir, cwd]
        .into_iter()
        .flatten()
        .map(|dir| dir.join(CONFIG_FILE))
        .filter(|path| path != config_path)
        .collect();
    files.dedup();
    files
}

/// Moves values from plaintext files into the store. A file is only deleted
/// once its values are confirmed readable from the store, and values already
/// in the store win over older copies.
fn migrate(store: &dyn SecretStore, files: &[PathBuf]) {
    for path in files {
        let Ok(content) = fs::read_to_string(path) else {
            continue;
        };
        let values = parse_config(&content);
        if let Err(error) = import(store, &values) {
            eprintln!("could not migrate {}: {error}", path.display());
            continue;
        }
        let known = KINDS.map(CredentialKind::config_key);
        if !values.keys().all(|key| known.contains(&key.as_str())) {
            eprintln!("keeping {}: it has other settings", path.display());
        } else if let Err(error) = fs::remove_file(path) {
            eprintln!("could not delete {}: {error}", path.display());
        }
    }
}

fn import(store: &dyn SecretStore, values: &HashMap<String, String>) -> Result<(), String> {
    for kind in KINDS {
        let key = kind.config_key();
        let Some(value) = values.get(key).filter(|value| !value.is_empty()) else {
            continue;
        };
        if store.get(key)?.is_some() {
            continue;
        }
        store.set(key, value)?;
        if store.get(key)?.as_ref() != Some(value) {
            return Err(format!("{key} did not read back correctly"));
        }
    }
    Ok(())
}

fn parse_config(content: &str) -> HashMap<String, String> {
    content
        .lines()
        .map(str::trim)
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
        .filter_map(|line| line.split_once('='))
        .map(|(key, value)| (key.trim().to_string(), value.trim().to_string()))
        .collect()
}

fn update_config(content: &str, key: &str, value: &str) -> String {
    let prefix = format!("{key}=");
    let replacement = format!("{key}={value}");
    let mut found = false;
    let mut lines: Vec<String> = content
        .lines()
        .map(|line| {
            if !found && line.starts_with(&prefix) {
                found = true;
                replacement.clone()
            } else {
                line.to_string()
            }
        })
        .collect();
    if !found {
        lines.push(replacement);
    }
    lines.join("\n") + "\n"
}

#[cfg(unix)]
fn write_private(path: &Path, content: &str) -> io::Result<()> {
    use std::io::Write;
    use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};

    let mut file = fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(0o600)
        .open(path)?;
    file.set_permissions(fs::Permissions::from_mode(0o600))?;
    file.write_all(content.as_bytes())
}

#[cfg(not(unix))]
fn write_private(path: &Path, content: &str) -> io::Result<()> {
    fs::write(path, content)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[derive(Default)]
    struct MemoryStore {
        values: Mutex<HashMap<String, String>>,
        reject_writes: bool,
    }

    impl SecretStore for MemoryStore {
        fn get(&self, key: &str) -> Result<Option<String>, String> {
            Ok(self.values.lock().unwrap().get(key).cloned())
        }

        fn set(&self, key: &str, value: &str) -> Result<(), String> {
            if self.reject_writes {
                return Err("locked".to_string());
            }
            let mut values = self.values.lock().unwrap();
            values.insert(key.to_string(), value.to_string());
            Ok(())
        }
    }

    fn write_config(dir: &tempfile::TempDir, content: &str) -> PathBuf {
        let path = dir.path().join(CONFIG_FILE);
        fs::write(&path, content).unwrap();
        path
    }

    #[test]
    fn migration_moves_values_and_deletes_the_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = write_config(&dir, "# old\nPOESESSID=abc\nBPL_TOKEN=token\n");
        let store = MemoryStore::default();
        migrate(&store, std::slice::from_ref(&path));
        assert_eq!(store.get("POESESSID").unwrap().as_deref(), Some("abc"));
        assert_eq!(store.get("BPL_TOKEN").unwrap().as_deref(), Some("token"));
        assert!(!path.exists());
    }

    #[test]
    fn migration_keeps_the_file_when_the_store_rejects_it() {
        let dir = tempfile::tempdir().unwrap();
        let path = write_config(&dir, "POESESSID=abc\n");
        let store = MemoryStore {
            reject_writes: true,
            ..MemoryStore::default()
        };
        migrate(&store, std::slice::from_ref(&path));
        assert!(path.exists());
    }

    #[test]
    fn migration_does_not_overwrite_newer_values() {
        let dir = tempfile::tempdir().unwrap();
        let path = write_config(&dir, "POESESSID=old\n");
        let store = MemoryStore::default();
        store.set("POESESSID", "new").unwrap();
        migrate(&store, std::slice::from_ref(&path));
        assert_eq!(store.get("POESESSID").unwrap().as_deref(), Some("new"));
        assert!(!path.exists());
    }

    #[test]
    fn migration_keeps_files_with_unknown_settings() {
        let dir = tempfile::tempdir().unwrap();
        let path = write_config(&dir, "POESESSID=abc\nOTHER=1\n");
        let store = MemoryStore::default();
        migrate(&store, std::slice::from_ref(&path));
        assert_eq!(store.get("POESESSID").unwrap().as_deref(), Some("abc"));
        assert!(path.exists());
    }

    #[test]
    fn file_store_round_trips_and_is_private() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join(CONFIG_FILE);
        let store = FileStore { path: path.clone() };
        assert_eq!(store.get("BPL_TOKEN").unwrap(), None);
        store.set("BPL_TOKEN", "token").unwrap();
        store.set("POESESSID", "abc").unwrap();
        assert_eq!(store.get("BPL_TOKEN").unwrap().as_deref(), Some("token"));
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mode = fs::metadata(&path).unwrap().permissions().mode();
            assert_eq!(mode & 0o777, 0o600);
        }
    }

    #[test]
    fn credentials_serve_saved_values_and_reject_blank_ones() {
        let credentials = Credentials::open(Box::new(MemoryStore::default()), None);
        assert_eq!(credentials.get(CredentialKind::BplToken), None);
        assert!(credentials.save(CredentialKind::BplToken, "  ").is_err());
        credentials.save(CredentialKind::BplToken, " token\n").unwrap();
        assert_eq!(
            credentials.get(CredentialKind::BplToken).as_deref(),
            Some("token")
        );
    }

    // Uses a separate service name so it never touches the app's real entries
    #[test]
    #[ignore = "needs a system keychain"]
    fn round_trips_through_the_system_keychain() {
        let store = KeyringStore {
            service: "com.bpl-poe.tools.test",
        };
        store.set("round-trip", "value").unwrap();
        let read = store.get("round-trip");
        Entry::new(store.service, "round-trip")
            .unwrap()
            .delete_credential()
            .unwrap();
        assert_eq!(read.unwrap().as_deref(), Some("value"));
        assert_eq!(store.get("round-trip").unwrap(), None);
    }

    #[test]
    fn parses_keys_and_skips_comments() {
        let values = parse_config("# comment\n\nPOESESSID = abc\nBPL_TOKEN=a=b\nbroken\n");
        assert_eq!(values.len(), 2);
        assert_eq!(values["POESESSID"], "abc");
        assert_eq!(values["BPL_TOKEN"], "a=b");
    }

    #[test]
    fn updates_a_key_in_place_and_keeps_other_lines() {
        let updated = update_config("# keep\nPOESESSID=old\nBPL_TOKEN=token\n", "POESESSID", "new");
        assert_eq!(updated, "# keep\nPOESESSID=new\nBPL_TOKEN=token\n");
    }

    #[test]
    fn appends_a_missing_key() {
        assert_eq!(update_config("", "BPL_TOKEN", "t"), "BPL_TOKEN=t\n");
        assert_eq!(update_config("A=1\n", "BPL_TOKEN", "t"), "A=1\nBPL_TOKEN=t\n");
    }
}
