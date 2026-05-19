mod mutation;
mod plan;
mod time;

use std::time::SystemTime;

use crate::{app::Output, cli::MigrateCommand, errors::Error, git, paths};

const MIGRATION_COMMIT_MESSAGE: &str = "lnk: migrate v1 setup repo";
const BACKUP_TAG_PREFIX: &str = "lnk-v1-backup";

pub fn migrate(command: MigrateCommand) -> Result<Output, Error> {
    let MigrateCommand::V1 { write, yes } = command;

    if write && !yes {
        return Err(Error::MigrationConfirmationRequired);
    }

    let repo = paths::repo_dir()?;
    let timestamp = time::timestamp(SystemTime::now())?;
    let mut plan = plan::MigrationPlan::build(&repo, &timestamp)?;

    if write {
        if git::is_dirty(&repo)? {
            return Err(Error::MigrationDirtyTree);
        }

        mutation::apply(&repo, &plan)?;
        plan.output.wrote = true;
    }

    Ok(Output::MigrationPlan(plan.output))
}
