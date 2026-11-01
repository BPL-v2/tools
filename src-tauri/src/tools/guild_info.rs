use crate::error::{CredentialKind, ToolError};
use crate::http;
use regex::Regex;
use reqwest::{Client, StatusCode};
use scraper::{Html, Selector};
use serde::Serialize;

const NOT_IN_GUILD: &str = "You are not currently a member of a Guild";

#[derive(Debug, PartialEq, Serialize)]
pub struct GuildInfo {
    pub id: i64,
    pub name: String,
    pub tag: String,
}

pub async fn fetch_guild_info(http: &Client, session_id: &str) -> Result<GuildInfo, ToolError> {
    let request = http.get("https://www.pathofexile.com/my-guild");
    let response = http::with_browser_headers(request, session_id).send().await?;
    let status = response.status();
    if status == StatusCode::UNAUTHORIZED || status == StatusCode::FORBIDDEN {
        return Err(ToolError::credential(
            CredentialKind::PoeSession,
            format!("HttpStatusCode: {} (PoE Session ID invalid)", status.as_u16()),
        ));
    }
    if status != StatusCode::OK {
        return Err(ToolError::Other(format!(
            "unexpected status code: {}",
            status.as_u16()
        )));
    }
    let html = response.text().await?;
    guild_from_page(&html)
}

/// Reads a 200 response, so PoE accepted the session: invalid ones get a 401.
fn guild_from_page(html: &str) -> Result<GuildInfo, ToolError> {
    if html.contains(NOT_IN_GUILD) {
        return Err(ToolError::Other(
            "Your PoE session is valid, but this account is not a member of a guild on PoE 1 PC."
                .to_string(),
        ));
    }
    parse_guild_info(html).map_err(|missing| {
        ToolError::Other(format!(
            "Could not read the {missing} from pathofexile.com/my-guild. The page may have changed."
        ))
    })
}

fn last_text(document: &Html, selector: &str) -> Option<String> {
    let selector = Selector::parse(selector).expect("valid selector");
    let element = document.select(&selector).next_back()?;
    Some(element.text().next()?.trim().to_string())
}

fn parse_guild_info(html: &str) -> Result<GuildInfo, &'static str> {
    let id = Regex::new(r"/guild/profile/(\d+)")
        .expect("valid regex")
        .captures(html)
        .and_then(|captures| captures[1].parse().ok())
        .ok_or("guild ID")?;
    let document = Html::parse_document(html);
    let name = last_text(&document, "h1.name")
        .filter(|name| !name.is_empty())
        .ok_or("guild name")?;
    let tag = last_text(&document, "p.guild-tag")
        .map(|tag| tag.trim_matches(['<', '>']).to_string())
        .filter(|tag| !tag.is_empty())
        .ok_or("guild tag")?;
    Ok(GuildInfo { id, name, tag })
}

#[cfg(test)]
mod tests {
    use super::*;

    const PAGE: &str = r#"<html><body>
        <a href="/guild/profile/408208/stash-history">Stash</a>
        <h1 class="name"> The Guild </h1>
        <p class="guild-tag">&lt;TAG&gt;</p>
    </body></html>"#;

    #[test]
    fn parses_id_name_and_decoded_tag() {
        let expected = GuildInfo {
            id: 408208,
            name: "The Guild".to_string(),
            tag: "TAG".to_string(),
        };
        assert_eq!(parse_guild_info(PAGE), Ok(expected));
    }

    fn page_error(html: &str) -> String {
        match guild_from_page(html) {
            Err(ToolError::Other(message)) => message,
            other => panic!("expected a non credential error, got {other:?}"),
        }
    }

    #[test]
    fn recognizes_the_not_in_a_guild_page() {
        let page = "<h2>Guild Not Found</h2>\
            <p>You are not currently a member of a Guild on PoE 1 PC.</p>";
        assert!(page_error(page).contains("not a member of a guild"));
    }

    #[test]
    fn an_unknown_page_is_reported_as_changed() {
        assert!(page_error("<html></html>").contains("page may have changed"));
    }

    #[test]
    fn reports_what_is_missing() {
        assert_eq!(parse_guild_info("<html></html>"), Err("guild ID"));
        let without_name = r#"<a href="/guild/profile/1"></a>"#;
        assert_eq!(parse_guild_info(without_name), Err("guild name"));
    }
}
