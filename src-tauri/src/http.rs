use reqwest::{Client, RequestBuilder};

pub const BPL_BASE_URL: &str = "https://bpl-poe.com/api";

const USER_AGENT: &str = "Contact: liberatorist@gmail.com";

pub fn client() -> Result<Client, reqwest::Error> {
    Client::builder().user_agent(USER_AGENT).build()
}

pub fn with_bpl_token(request: RequestBuilder, token: &str) -> RequestBuilder {
    request.header("Authorization", format!("Bearer {token}"))
}

pub fn with_poe_session(request: RequestBuilder, session_id: &str) -> RequestBuilder {
    request.header("Cookie", format!("POESESSID={session_id}"))
}

/// Adds the session plus the headers that make the request look like a browser
/// navigation, which the guild pages on pathofexile.com expect.
pub fn with_browser_headers(request: RequestBuilder, session_id: &str) -> RequestBuilder {
    with_poe_session(request, session_id)
        .header("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
        .header("accept-language", "en-US,en;q=0.6")
        .header("cache-control", "no-cache")
        .header("pragma", "no-cache")
        .header("sec-ch-ua", r#""Not:A-Brand";v="99", "Brave";v="145", "Chromium";v="145""#)
        .header("sec-ch-ua-mobile", "?0")
        .header("sec-ch-ua-platform", r#""Windows""#)
        .header("sec-fetch-dest", "document")
        .header("sec-fetch-mode", "navigate")
        .header("sec-fetch-site", "none")
        .header("sec-fetch-user", "?1")
        .header("sec-gpc", "1")
        .header("upgrade-insecure-requests", "1")
        .header("POESESSID", session_id)
}
