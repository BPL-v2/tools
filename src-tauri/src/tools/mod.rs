pub mod characters;
mod guild_info;
pub mod invites;
mod rate_limiter;
mod stash;

use crate::error::ToolError;
use crate::jobs::{Ctx, Tool};

pub async fn run(ctx: &Ctx, tool: Tool) -> Result<(), ToolError> {
    match tool {
        Tool::Characters => characters::run(ctx).await,
        Tool::Invites => invites::run(ctx).await,
        Tool::Stash => stash::run(ctx).await,
    }
}
