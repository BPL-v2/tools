use crate::error::{CredentialKind, ToolError, check_bpl_response};
use crate::http::{self, BPL_BASE_URL};
use crate::jobs::{Ctx, ToolEvent, now_ms};
use regex::Regex;
use reqwest::StatusCode;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;

#[derive(Debug, Clone, Deserialize)]
struct Member {
    id: i64,
    #[serde(rename = "memberName")]
    member_name: String,
    role: String,
}

#[derive(Debug, Deserialize)]
struct MembersResponse {
    members: Vec<Member>,
}

#[derive(Debug, Deserialize)]
struct Player {
    user: User,
    team_id: Option<i64>,
}

#[derive(Debug, Deserialize)]
struct User {
    account_name: String,
}

#[derive(Debug, Deserialize)]
struct Event {
    name: String,
}

#[derive(Debug, Serialize)]
struct AcceptRequest {
    name: &'static str,
    value: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize)]
pub struct InviteReport {
    league: String,
    requested: usize,
    accepted_members: usize,
    newly_accepted: Vec<String>,
    unsorted_requests: Vec<String>,
    unsorted_members: Vec<String>,
}

struct Session<'a> {
    ctx: &'a Ctx,
    poe_session: String,
    bpl_token: String,
    league: String,
    league_id: String,
}

fn parse_league_id(name: &str) -> Option<String> {
    let regex = Regex::new(r"\s*\(PL(\d+)\)").expect("valid regex");
    Some(regex.captures(name)?[1].to_string())
}

/// Splits the league's members into the requests to accept and the report.
fn classify(
    league: &str,
    members: Vec<Member>,
    sorted_users: &HashSet<String>,
) -> (Vec<Member>, InviteReport) {
    let mut report = InviteReport {
        league: league.to_string(),
        ..InviteReport::default()
    };
    let mut to_accept = Vec::new();
    for member in members {
        let sorted = sorted_users.contains(&member.member_name);
        if member.role == "requested_invite" {
            report.requested += 1;
            if sorted {
                report.newly_accepted.push(member.member_name.clone());
                to_accept.push(member);
            } else {
                report.unsorted_requests.push(member.member_name);
            }
        } else {
            report.accepted_members += 1;
            if !sorted {
                report.unsorted_members.push(member.member_name);
            }
        }
    }
    (to_accept, report)
}

impl Session<'_> {
    async fn sorted_users(&self) -> Result<HashSet<String>, ToolError> {
        let url = format!("{BPL_BASE_URL}/events/current/signups");
        let request = http::with_bpl_token(self.ctx.http.get(url), &self.bpl_token);
        let players: Vec<Player> = check_bpl_response(request.send().await?)?.json().await?;
        Ok(players
            .into_iter()
            .filter(|player| player.team_id.is_some())
            .map(|player| player.user.account_name)
            .collect())
    }

    fn member_url(&self) -> String {
        format!(
            "https://www.pathofexile.com/api/private-league-member/{}",
            self.league_id
        )
    }

    async fn league_members(&self) -> Result<Vec<Member>, ToolError> {
        let request = self
            .ctx
            .http
            .get(self.member_url())
            .header("accept", "application/json")
            .query(&[
                ("sort", "roleDesc"),
                ("search", ""),
                ("offset", "0"),
                ("limit", "100"),
                ("_", &(now_ms() / 1000).to_string()),
            ]);
        let response = http::with_poe_session(request, &self.poe_session).send().await?;
        let status = response.status();
        if status == StatusCode::UNAUTHORIZED || status == StatusCode::FORBIDDEN {
            return Err(ToolError::credential(
                CredentialKind::PoeSession,
                format!("HttpStatusCode: {} (PoE Session ID invalid)", status.as_u16()),
            ));
        }
        if status != StatusCode::OK {
            return Err(ToolError::Other(format!(
                "HttpStatusCode: {} from PoE API",
                status.as_u16()
            )));
        }
        Ok(response.json::<MembersResponse>().await?.members)
    }

    async fn accept(&self, members: &[Member]) -> Result<(), ToolError> {
        let accepts: Vec<AcceptRequest> = members
            .iter()
            .map(|member| AcceptRequest {
                name: "accept",
                value: member.id.to_string(),
            })
            .collect();
        let request = self
            .ctx
            .http
            .post(self.member_url())
            .header("Accept", "application/json, text/javascript, */*; q=0.01")
            .header("Accept-Language", "en-US,en;q=0.5")
            .header("Cache-Control", "no-cache")
            .header("X-Requested-With", "XMLHttpRequest")
            .json(&accepts);
        let response = http::with_poe_session(request, &self.poe_session).send().await?;
        let status = response.status();
        if status == StatusCode::UNAUTHORIZED || status == StatusCode::FORBIDDEN {
            let body = response.text().await.unwrap_or_default();
            return Err(ToolError::credential(
                CredentialKind::PoeSession,
                format!(
                    "HttpStatusCode: {} (PoE Session ID invalid) - Response: {body}",
                    status.as_u16()
                ),
            ));
        }
        if status != StatusCode::OK {
            let body = response.text().await.unwrap_or_default();
            return Err(ToolError::Other(format!(
                "failed to accept invites. Status code: {}, Response: {body}",
                status.as_u16()
            )));
        }
        Ok(())
    }

    async fn handle_invites(&self) -> Result<(), ToolError> {
        self.ctx.info(format!("Checking requests for league {}", self.league));
        let sorted_users = self
            .sorted_users()
            .await
            .map_err(|error| error.context("failed to get sorted users"))?;
        let members = self
            .league_members()
            .await
            .map_err(|error| error.context("failed to get league join requests"))?;
        let (to_accept, report) = classify(&self.league, members, &sorted_users);
        self.ctx.info(format!(
            "Found {} requested invites and {} accepted members.",
            report.requested, report.accepted_members
        ));
        if to_accept.is_empty() {
            self.ctx.info("No new members to add.");
        } else {
            self.accept(&to_accept).await?;
            self.ctx
                .info(format!("{} invites accepted successfully.", to_accept.len()));
        }
        self.ctx.emit(ToolEvent::InviteReport {
            report,
            time: now_ms(),
        });
        Ok(())
    }
}

async fn connect(ctx: &Ctx) -> Result<Session<'_>, ToolError> {
    let bpl_token = ctx.credential(CredentialKind::BplToken)?;
    let poe_session = ctx.credential(CredentialKind::PoeSession)?;
    let url = format!("{BPL_BASE_URL}/events/current");
    let event: Event = ctx.http.get(url).send().await?.error_for_status()?.json().await?;
    let league_id = parse_league_id(&event.name)
        .ok_or_else(|| ToolError::Other(format!("unexpected league name format: {}", event.name)))?;
    Ok(Session {
        ctx,
        poe_session,
        bpl_token,
        league: event.name,
        league_id,
    })
}

pub async fn run(ctx: &Ctx) -> Result<(), ToolError> {
    let session = connect(ctx).await?;
    while ctx.next_pass(session.handle_invites().await).await? {}
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn member(id: i64, name: &str, role: &str) -> Member {
        Member {
            id,
            member_name: name.to_string(),
            role: role.to_string(),
        }
    }

    #[test]
    fn extracts_the_league_id_from_the_event_name() {
        assert_eq!(parse_league_id("BPL 12 (PL12345)").as_deref(), Some("12345"));
        assert_eq!(parse_league_id("BPL 12"), None);
    }

    #[test]
    fn accepts_only_sorted_requests_and_reports_the_rest() {
        let sorted = HashSet::from(["alice".to_string(), "carol".to_string()]);
        let members = vec![
            member(1, "alice", "requested_invite"),
            member(2, "bob", "requested_invite"),
            member(3, "carol", "member"),
            member(4, "dave", "member"),
        ];
        let (to_accept, report) = classify("BPL", members, &sorted);
        assert_eq!(to_accept.iter().map(|m| m.id).collect::<Vec<_>>(), [1]);
        assert_eq!(report.requested, 2);
        assert_eq!(report.accepted_members, 2);
        assert_eq!(report.newly_accepted, ["alice"]);
        assert_eq!(report.unsorted_requests, ["bob"]);
        assert_eq!(report.unsorted_members, ["dave"]);
    }

    #[test]
    fn serializes_accepts_as_the_poe_form_payload() {
        let accepts = [AcceptRequest {
            name: "accept",
            value: "42".to_string(),
        }];
        assert_eq!(
            serde_json::to_string(&accepts).unwrap(),
            r#"[{"name":"accept","value":"42"}]"#
        );
    }
}
