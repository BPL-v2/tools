use super::guild_info::{GuildInfo, fetch_guild_info};
use super::rate_limiter::RateLimiter;
use crate::error::{CredentialKind, ToolError, check_bpl_response};
use crate::http::{self, BPL_BASE_URL};
use crate::jobs::{Ctx, ToolEvent, now_ms};
use reqwest::header::RETRY_AFTER;
use reqwest::{Response, StatusCode};
use serde::Deserialize;
use std::time::{Duration, Instant};

const DAY_SECONDS: i64 = 24 * 60 * 60;

#[derive(Debug, Deserialize)]
struct StashEntry {
    id: String,
    time: i64,
}

#[derive(Debug, Deserialize)]
struct StashChanges {
    entries: Vec<StashEntry>,
    truncated: bool,
}

#[derive(Debug, Deserialize)]
struct Timestamps {
    earliest: Option<i64>,
    latest: Option<i64>,
    league_start: i64,
    league_end: i64,
}

#[derive(Debug, Deserialize)]
struct AddResponse {
    number_of_added_entries: u64,
}

struct Session<'a> {
    ctx: &'a Ctx,
    poe_session: String,
    bpl_token: String,
    guild_id: i64,
    limiter: RateLimiter,
    league_start: i64,
    league_end: i64,
    /// History older than this still has to be fetched once
    backfill_from: Option<i64>,
    /// History newer than this has not been fetched yet
    latest: i64,
    uploaded_entries: u64,
}

/// How far a walk from the league end back to its start has come. None without valid league times.
fn progress_percent(league_start: i64, league_end: i64, current: i64) -> Option<u32> {
    let total = league_end - league_start;
    if total <= 0 {
        return None;
    }
    let done = (league_end - current).clamp(0, total);
    Some((done * 100 / total) as u32)
}

fn history_url(guild_id: i64, newest: i64, oldest: i64, from_id: Option<&str>) -> String {
    let mut url = format!(
        "https://www.pathofexile.com/api/guild/{guild_id}/stash/history?from={oldest}&end={newest}"
    );
    if let Some(from_id) = from_id {
        url.push_str(&format!("&fromid={from_id}"));
    }
    url
}

fn check_history_response(response: &Response) -> Result<(), ToolError> {
    let status = response.status();
    if status == StatusCode::TOO_MANY_REQUESTS {
        let retry_after = response
            .headers()
            .get(RETRY_AFTER)
            .and_then(|value| value.to_str().ok())
            .and_then(|value| value.parse::<u64>().ok());
        let wait = retry_after.map_or("30m".to_string(), |seconds| format!("{seconds}s"));
        return Err(ToolError::Fatal(format!(
            "HttpStatusCode: 429 (Too many requests - Wait {wait} before trying again)"
        )));
    }
    if status != StatusCode::OK {
        return Err(ToolError::credential(
            CredentialKind::PoeSession,
            format!(
                "HttpStatusCode: {} (PoE Session ID most likely invalid or you are not a guild officer)",
                status.as_u16()
            ),
        ));
    }
    Ok(())
}

async fn register_guild(ctx: &Ctx, token: &str, guild: &GuildInfo) -> Result<(), ToolError> {
    let url = format!("{BPL_BASE_URL}/current/guilds/{}", guild.id);
    let request = http::with_bpl_token(ctx.http.put(url), token).json(guild);
    let response = request.send().await?;
    let status = response.status();
    if status == StatusCode::UNAUTHORIZED {
        return Err(ToolError::credential(
            CredentialKind::BplToken,
            "HttpStatusCode: 401 (BPL Token invalid or expired)",
        ));
    }
    if status != StatusCode::OK {
        let body = response.text().await.unwrap_or_default();
        return Err(ToolError::Other(format!(
            "failed to register guild: {status} - You are not registered as a team lead ({body})"
        )));
    }
    Ok(())
}

async fn connect(ctx: &Ctx) -> Result<Session<'_>, ToolError> {
    let poe_session = ctx.credential(CredentialKind::PoeSession)?;
    let bpl_token = ctx.credential(CredentialKind::BplToken)?;
    let guild = fetch_guild_info(&ctx.http, &poe_session)
        .await
        .map_err(|error| error.context("Failed to fetch guild info"))?;
    ctx.info(format!("Syncing guild {} <{}>", guild.name, guild.tag));
    register_guild(ctx, &bpl_token, &guild).await?;

    let url = format!(
        "{BPL_BASE_URL}/current/guilds/{}/stash-history/latest_timestamp",
        guild.id
    );
    let request = http::with_bpl_token(ctx.http.get(url), &bpl_token);
    let timestamps: Timestamps = check_bpl_response(request.send().await?)
        .map_err(|error| error.context("error getting latest timestamp"))?
        .json()
        .await?;

    let (backfill_from, latest) = match (timestamps.earliest, timestamps.latest) {
        (Some(earliest), Some(latest)) => (Some(earliest), latest),
        _ => (None, timestamps.league_start),
    };
    Ok(Session {
        ctx,
        poe_session,
        bpl_token,
        guild_id: guild.id,
        limiter: RateLimiter::new(),
        league_start: timestamps.league_start,
        league_end: timestamps.league_end,
        backfill_from,
        latest,
        uploaded_entries: 0,
    })
}

impl Session<'_> {
    async fn acquire_rate_limit(&mut self) {
        if self.limiter.try_acquire(Instant::now()) {
            return;
        }
        self.ctx.info("Rate limit reached, waiting...");
        while !self.limiter.try_acquire(Instant::now()) {
            tokio::time::sleep(Duration::from_millis(100)).await;
        }
    }

    async fn upload(&mut self, body: Vec<u8>) -> Result<(), ToolError> {
        let url = format!(
            "{BPL_BASE_URL}/current/guilds/{}/stash-history",
            self.guild_id
        );
        let request = http::with_bpl_token(self.ctx.http.post(url), &self.bpl_token)
            .header("Content-Type", "application/json")
            .body(body);
        let added: AddResponse = check_bpl_response(request.send().await?)
            .map_err(|error| error.context("failed to upload stash history"))?
            .json()
            .await?;
        self.uploaded_entries += added.number_of_added_entries;
        Ok(())
    }

    /// Walks the history from newest to oldest, one page at a time.
    async fn sync_range(&mut self, mut newest: i64, oldest: i64) -> Result<(), ToolError> {
        let mut from_id = None;
        loop {
            self.acquire_rate_limit().await;
            let url = history_url(self.guild_id, newest, oldest, from_id.as_deref());
            let request = self.ctx.http.get(url);
            let response = http::with_browser_headers(request, &self.poe_session)
                .send()
                .await?;
            check_history_response(&response)?;

            let header = |name: &str| {
                let value = response.headers().get(name)?;
                value.to_str().ok().map(str::to_string)
            };
            let limits = header("X-Rate-Limit-Account");
            let state = header("X-Rate-Limit-Account-State");
            match (limits, state) {
                (Some(limits), Some(state)) => {
                    if let Err(error) = self.limiter.update(&limits, &state, Instant::now()) {
                        self.ctx.warn(format!("Could not update rate limiter: {error}"));
                    }
                }
                _ => self.ctx.warn("Could not update rate limiter: headers missing"),
            }

            let body = response.bytes().await?.to_vec();
            let changes: StashChanges = serde_json::from_slice(&body)?;
            let Some(last) = changes.entries.last() else {
                return Ok(());
            };
            let current_time = changes.entries[0].time;
            newest = last.time;
            from_id = Some(last.id.clone());

            self.upload(body).await?;
            self.ctx.emit(ToolEvent::Progress {
                percent: progress_percent(self.league_start, self.league_end, current_time),
                current_time,
                rate_limit: self.limiter.state(Instant::now()),
                uploaded_entries: self.uploaded_entries,
            });
            if !changes.truncated {
                return Ok(());
            }
        }
    }

    async fn sync(&mut self) -> Result<(), ToolError> {
        let day_after_league_end = self.league_end + DAY_SECONDS;
        let started_at = (now_ms() / 1000) as i64;
        if let Some(earliest) = self.backfill_from {
            self.sync_range(earliest, self.league_start).await?;
            self.backfill_from = None;
        }
        self.sync_range(day_after_league_end, self.latest).await?;
        self.latest = started_at;
        self.ctx.info(format!(
            "Guild stash sync completed, {} entries uploaded so far",
            self.uploaded_entries
        ));
        Ok(())
    }
}

pub async fn run(ctx: &Ctx) -> Result<(), ToolError> {
    ctx.info("Connecting to Path of Exile and BPL...");
    let mut session = connect(ctx).await?;
    while ctx
        .next_pass(session.sync().await.map_err(|error| error.context("error getting history")))
        .await?
    {}
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn progress_counts_from_league_end_towards_start() {
        assert_eq!(progress_percent(100, 200, 200), Some(0));
        assert_eq!(progress_percent(100, 200, 150), Some(50));
        assert_eq!(progress_percent(100, 200, 100), Some(100));
    }

    #[test]
    fn progress_stays_in_range_outside_the_league() {
        assert_eq!(progress_percent(100, 200, 500), Some(0));
        assert_eq!(progress_percent(100, 200, 50), Some(100));
        assert_eq!(progress_percent(0, 0, 50), None);
    }

    #[test]
    fn history_url_matches_the_poe_api() {
        assert_eq!(
            history_url(7, 200, 100, None),
            "https://www.pathofexile.com/api/guild/7/stash/history?from=100&end=200"
        );
        assert_eq!(
            history_url(7, 200, 100, Some("abc")),
            "https://www.pathofexile.com/api/guild/7/stash/history?from=100&end=200&fromid=abc"
        );
    }

    #[test]
    fn parses_a_history_page_with_extra_fields() {
        let page = r#"{"entries":[{"id":"9","time":5,"item":"x","x":1}],"truncated":true}"#;
        let changes: StashChanges = serde_json::from_str(page).unwrap();
        assert_eq!(changes.entries[0].id, "9");
        assert!(changes.truncated);
    }
}
