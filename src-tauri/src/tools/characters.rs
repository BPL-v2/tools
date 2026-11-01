use crate::error::ToolError;
use crate::http::BPL_BASE_URL;
use crate::jobs::{Ctx, ToolEvent, now_ms};
use reqwest::Client;
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

const BASE_CLASSES: [&str; 7] = [
    "Scion", "Marauder", "Ranger", "Shadow", "Templar", "Witch", "Duelist",
];
const MIN_LEVEL_FOR_NAME_CHECK: u32 = 10;

#[derive(Debug, Default, Deserialize)]
struct Character {
    ascendancy: Option<String>,
}

#[derive(Debug, Deserialize)]
struct LadderEntry {
    user_id: i64,
    character_name: String,
    level: u32,
    character: Option<Character>,
}

#[derive(Debug, Deserialize)]
struct User {
    id: i64,
}

#[derive(Debug, Deserialize)]
struct Team {
    id: i64,
    name: String,
    abbreviation: Option<String>,
    allowed_classes: Option<Vec<String>>,
}

#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum Mismatch {
    Name {
        character: String,
        level: u32,
        team: String,
    },
    Ascendancy {
        character: String,
        level: u32,
        team: String,
        ascendancy: String,
    },
}

async fn get_json<T: DeserializeOwned>(http: &Client, path: &str) -> Result<T, ToolError> {
    let url = format!("{BPL_BASE_URL}{path}");
    Ok(http.get(url).send().await?.error_for_status()?.json().await?)
}

fn team_short(team: &Team) -> String {
    match team.abbreviation.as_deref() {
        Some(abbreviation) if !abbreviation.is_empty() => abbreviation.to_lowercase(),
        _ => team.name.chars().take(3).collect::<String>().to_lowercase(),
    }
}

fn find_mismatches(
    ladder: &[LadderEntry],
    user_teams: &HashMap<i64, i64>,
    teams: &[Team],
) -> Vec<Mismatch> {
    let teams: HashMap<i64, &Team> = teams.iter().map(|team| (team.id, team)).collect();
    let mut mismatches = Vec::new();
    for entry in ladder {
        let Some(team) = user_teams.get(&entry.user_id).and_then(|id| teams.get(id)) else {
            continue;
        };
        if entry.level >= MIN_LEVEL_FOR_NAME_CHECK
            && !entry.character_name.to_lowercase().contains(&team_short(team))
        {
            mismatches.push(Mismatch::Name {
                character: entry.character_name.clone(),
                level: entry.level,
                team: team.name.clone(),
            });
        }
        let ascendancy = entry
            .character
            .as_ref()
            .and_then(|character| character.ascendancy.as_deref())
            .unwrap_or_default();
        let allowed = team.allowed_classes.as_deref().unwrap_or_default();
        if !ascendancy.is_empty()
            && !BASE_CLASSES.contains(&ascendancy)
            && !allowed.iter().any(|class| class == ascendancy)
        {
            mismatches.push(Mismatch::Ascendancy {
                character: entry.character_name.clone(),
                level: entry.level,
                team: team.name.clone(),
                ascendancy: ascendancy.to_string(),
            });
        }
    }
    mismatches
}

async fn check(ctx: &Ctx) -> Result<(), ToolError> {
    ctx.info("Checking for player name mismatches...");
    let team_users: HashMap<i64, Vec<User>> = get_json(&ctx.http, "/events/current/users")
        .await
        .map_err(|error| error.context("failed to get users"))?;
    let teams: Vec<Team> = get_json(&ctx.http, "/events/current/teams")
        .await
        .map_err(|error| error.context("failed to get teams"))?;
    let ladder: Vec<LadderEntry> = get_json(&ctx.http, "/events/current/ladder")
        .await
        .map_err(|error| error.context("failed to get ladder"))?;

    let user_teams: HashMap<i64, i64> = team_users
        .into_iter()
        .flat_map(|(team, users)| users.into_iter().map(move |user| (user.id, team)))
        .collect();
    let mismatches = find_mismatches(&ladder, &user_teams, &teams);
    ctx.info(format!(
        "Checked {} ladder entries, mismatches found: {}",
        ladder.len(),
        mismatches.len()
    ));
    ctx.emit(ToolEvent::CharacterReport {
        mismatches,
        time: now_ms(),
    });
    Ok(())
}

pub async fn run(ctx: &Ctx) -> Result<(), ToolError> {
    while ctx.next_pass(check(ctx).await).await? {}
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn team(abbreviation: Option<&str>, name: &str) -> Team {
        Team {
            id: 1,
            name: name.to_string(),
            abbreviation: abbreviation.map(str::to_string),
            allowed_classes: Some(vec!["Juggernaut".to_string()]),
        }
    }

    fn entry(name: &str, level: u32, ascendancy: Option<&str>) -> LadderEntry {
        LadderEntry {
            user_id: 7,
            character_name: name.to_string(),
            level,
            character: Some(Character {
                ascendancy: ascendancy.map(str::to_string),
            }),
        }
    }

    fn check(entry: LadderEntry, team: Team) -> Vec<Mismatch> {
        find_mismatches(&[entry], &HashMap::from([(7, 1)]), &[team])
    }

    #[test]
    fn accepts_valid_characters() {
        let team = team(Some("ABC"), "Alpha");
        assert!(check(entry("my_abc_jugg", 90, Some("Juggernaut")), team).is_empty());
    }

    #[test]
    fn flags_a_missing_abbreviation_from_level_10() {
        let found = check(entry("nope", 10, None), team(Some("ABC"), "Alpha"));
        assert_eq!(found.len(), 1);
        assert!(matches!(found[0], Mismatch::Name { .. }));
        assert!(check(entry("nope", 9, None), team(Some("ABC"), "Alpha")).is_empty());
    }

    #[test]
    fn falls_back_to_the_team_name_and_handles_short_names() {
        assert!(check(entry("xalpyy", 50, None), team(None, "Alpha")).is_empty());
        assert!(check(entry("xéyy", 50, None), team(None, "é")).is_empty());
    }

    #[test]
    fn allows_base_classes_but_flags_foreign_ascendancies() {
        let team_a = team(Some("ABC"), "Alpha");
        assert!(check(entry("abc", 1, Some("Marauder")), team_a).is_empty());
        let found = check(entry("abc", 1, Some("Slayer")), team(Some("ABC"), "Alpha"));
        assert!(matches!(found[0], Mismatch::Ascendancy { .. }));
    }

    // Run with `cargo test -- --ignored` to check the models against production
    #[test]
    #[ignore = "needs network access"]
    fn parses_the_live_public_api() {
        tauri::async_runtime::block_on(async {
            let http = crate::http::client().unwrap();
            let _: HashMap<i64, Vec<User>> =
                get_json(&http, "/events/current/users").await.unwrap();
            let teams: Vec<Team> = get_json(&http, "/events/current/teams").await.unwrap();
            let ladder: Vec<LadderEntry> =
                get_json(&http, "/events/current/ladder").await.unwrap();
            println!("{} teams, {} ladder entries", teams.len(), ladder.len());
        });
    }

    #[test]
    fn ignores_players_without_a_team() {
        let found = find_mismatches(
            &[entry("nope", 90, None)],
            &HashMap::new(),
            &[team(Some("ABC"), "Alpha")],
        );
        assert!(found.is_empty());
    }
}
