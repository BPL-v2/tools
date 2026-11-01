use reqwest::{Response, StatusCode};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum CredentialKind {
    PoeSession,
    BplToken,
}

impl CredentialKind {
    /// The key this credential is stored under in bpl-config.txt
    pub fn config_key(self) -> &'static str {
        match self {
            CredentialKind::PoeSession => "POESESSID",
            CredentialKind::BplToken => "BPL_TOKEN",
        }
    }

    pub fn display_name(self) -> &'static str {
        match self {
            CredentialKind::PoeSession => "PoE Session ID",
            CredentialKind::BplToken => "BPL Token",
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum ToolError {
    /// A credential is missing or was rejected. Ends the run so it can be replaced.
    #[error("{message}")]
    Credential {
        kind: CredentialKind,
        message: String,
    },
    /// Retrying on the next interval would make things worse, e.g. when rate limited.
    #[error("{0}")]
    Fatal(String),
    #[error("{0}")]
    Other(String),
}

impl ToolError {
    pub fn credential(kind: CredentialKind, message: impl Into<String>) -> Self {
        ToolError::Credential {
            kind,
            message: message.into(),
        }
    }

    pub fn context(self, context: &str) -> Self {
        match self {
            ToolError::Credential { kind, message } => ToolError::Credential {
                kind,
                message: format!("{context}: {message}"),
            },
            ToolError::Fatal(message) => ToolError::Fatal(format!("{context}: {message}")),
            ToolError::Other(message) => ToolError::Other(format!("{context}: {message}")),
        }
    }
}

impl From<reqwest::Error> for ToolError {
    fn from(error: reqwest::Error) -> Self {
        ToolError::Other(error.to_string())
    }
}

impl From<serde_json::Error> for ToolError {
    fn from(error: serde_json::Error) -> Self {
        ToolError::Other(error.to_string())
    }
}

/// Fails on non-2xx BPL responses, treating 401 and 403 as an invalid token.
pub fn check_bpl_response(response: Response) -> Result<Response, ToolError> {
    let status = response.status();
    if status == StatusCode::UNAUTHORIZED || status == StatusCode::FORBIDDEN {
        return Err(ToolError::credential(
            CredentialKind::BplToken,
            format!("HttpStatusCode: {} (BPL Token invalid or expired)", status.as_u16()),
        ));
    }
    if !status.is_success() {
        return Err(ToolError::Other(format!(
            "HttpStatusCode: {} from BPL backend",
            status.as_u16()
        )));
    }
    Ok(response)
}
