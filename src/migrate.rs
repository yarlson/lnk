use crate::{app::Output, cli::MigrateCommand, errors::Error};

pub fn migrate(command: MigrateCommand) -> Result<Output, Error> {
    let _ = command;
    Err(Error::not_implemented("migrate"))
}
