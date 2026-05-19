use std::str::FromStr;

use clap::{Args, Parser, Subcommand};

#[derive(Debug, Parser)]
#[command(
    name = "lnk",
    version,
    about = "Git-backed userland machine setup repo",
    long_about = None
)]
pub struct Cli {
    #[command(subcommand)]
    command: CliCommand,
}

impl Cli {
    pub fn into_command(self) -> Command {
        self.command.into_command()
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Command {
    Init,
    Profiles { output: OutputFormat },
    Apply(ApplyCommand),
    Status { output: OutputFormat },
    Run(RunCommand),
    Doctor { output: OutputFormat },
    Migrate(MigrateCommand),
}

impl Command {
    pub fn output_format(&self) -> OutputFormat {
        match self {
            Self::Profiles { output }
            | Self::Apply(ApplyCommand { output, .. })
            | Self::Status { output }
            | Self::Doctor { output } => *output,
            Self::Init | Self::Run(_) | Self::Migrate(_) => OutputFormat::Human,
        }
    }

    pub fn requires_git(&self) -> bool {
        match self {
            Self::Init
            | Self::Profiles { .. }
            | Self::Apply(_)
            | Self::Status { .. }
            | Self::Run(_)
            | Self::Doctor { .. }
            | Self::Migrate(_) => true,
        }
    }
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum OutputFormat {
    Human,
    Json,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ApplyCommand {
    pub profiles: Vec<String>,
    pub dry_run: bool,
    pub backup: bool,
    pub output: OutputFormat,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum MigrateCommand {
    V1 { write: bool, yes: bool },
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum RunCommand {
    Script { script: String },
    ProfileScript { profile: String, script: String },
}

impl FromStr for RunCommand {
    type Err = &'static str;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value.split_once(':') {
            Some((profile, script))
                if !profile.is_empty() && !script.is_empty() && !script.contains(':') =>
            {
                Ok(Self::ProfileScript {
                    profile: profile.to_string(),
                    script: script.to_string(),
                })
            }
            Some(_) => Err("expected SCRIPT or PROFILE:SCRIPT"),
            None if !value.is_empty() => Ok(Self::Script {
                script: value.to_string(),
            }),
            None => Err("expected SCRIPT or PROFILE:SCRIPT"),
        }
    }
}

#[derive(Debug, Subcommand)]
enum CliCommand {
    Init,
    Profiles(JsonArgs),
    Apply(ApplyArgs),
    Status(JsonArgs),
    Run(RunArgs),
    Doctor(JsonArgs),
    Migrate(MigrateArgs),
}

impl CliCommand {
    fn into_command(self) -> Command {
        match self {
            Self::Init => Command::Init,
            Self::Profiles(args) => Command::Profiles {
                output: args.output_format(),
            },
            Self::Apply(args) => Command::Apply(ApplyCommand {
                output: args.output_format(),
                profiles: args.profiles,
                dry_run: args.dry_run,
                backup: args.backup,
            }),
            Self::Status(args) => Command::Status {
                output: args.output_format(),
            },
            Self::Run(args) => Command::Run(args.target),
            Self::Doctor(args) => Command::Doctor {
                output: args.output_format(),
            },
            Self::Migrate(args) => Command::Migrate(args.into_command()),
        }
    }
}

#[derive(Debug, Args)]
struct JsonArgs {
    #[arg(long)]
    json: bool,
}

impl JsonArgs {
    fn output_format(&self) -> OutputFormat {
        if self.json {
            OutputFormat::Json
        } else {
            OutputFormat::Human
        }
    }
}

#[derive(Debug, Args)]
struct ApplyArgs {
    #[arg(value_name = "PROFILE")]
    profiles: Vec<String>,

    #[arg(long)]
    dry_run: bool,

    #[arg(long)]
    backup: bool,

    #[arg(long)]
    json: bool,
}

impl ApplyArgs {
    fn output_format(&self) -> OutputFormat {
        if self.json {
            OutputFormat::Json
        } else {
            OutputFormat::Human
        }
    }
}

#[derive(Debug, Args)]
struct RunArgs {
    #[arg(value_name = "SCRIPT_OR_PROFILE_SCRIPT")]
    target: RunCommand,
}

#[derive(Debug, Args)]
struct MigrateArgs {
    #[command(subcommand)]
    command: MigrateSubcommand,
}

impl MigrateArgs {
    fn into_command(self) -> MigrateCommand {
        match self.command {
            MigrateSubcommand::V1(args) => MigrateCommand::V1 {
                write: args.write,
                yes: args.yes,
            },
        }
    }
}

#[derive(Debug, Subcommand)]
enum MigrateSubcommand {
    V1(MigrateV1Args),
}

#[derive(Debug, Args)]
struct MigrateV1Args {
    #[arg(long, requires = "yes")]
    write: bool,

    #[arg(long, requires = "write")]
    yes: bool,
}

#[cfg(test)]
mod tests {
    use clap::Parser;

    use super::{ApplyCommand, Cli, Command, MigrateCommand, OutputFormat, RunCommand};

    fn parse(args: &[&str]) -> Command {
        Cli::try_parse_from(args)
            .expect("args should parse")
            .into_command()
    }

    #[test]
    fn routes_init() {
        assert_eq!(parse(&["lnk", "init"]), Command::Init);
    }

    #[test]
    fn routes_profiles_json() {
        assert_eq!(
            parse(&["lnk", "profiles", "--json"]),
            Command::Profiles {
                output: OutputFormat::Json
            }
        );
    }

    #[test]
    fn routes_apply_with_profiles_and_flags() {
        assert_eq!(
            parse(&[
                "lnk",
                "apply",
                "base",
                "mac",
                "--dry-run",
                "--backup",
                "--json",
            ]),
            Command::Apply(ApplyCommand {
                profiles: vec!["base".to_string(), "mac".to_string()],
                dry_run: true,
                backup: true,
                output: OutputFormat::Json,
            })
        );
    }

    #[test]
    fn routes_apply_without_profiles() {
        assert_eq!(
            parse(&["lnk", "apply"]),
            Command::Apply(ApplyCommand {
                profiles: Vec::new(),
                dry_run: false,
                backup: false,
                output: OutputFormat::Human,
            })
        );
    }

    #[test]
    fn routes_status_json() {
        assert_eq!(
            parse(&["lnk", "status", "--json"]),
            Command::Status {
                output: OutputFormat::Json
            }
        );
    }

    #[test]
    fn routes_run_script() {
        assert_eq!(
            parse(&["lnk", "run", "setup"]),
            Command::Run(RunCommand::Script {
                script: "setup".to_string()
            })
        );
    }

    #[test]
    fn routes_run_profile_script() {
        assert_eq!(
            parse(&["lnk", "run", "work:setup"]),
            Command::Run(RunCommand::ProfileScript {
                profile: "work".to_string(),
                script: "setup".to_string()
            })
        );
    }

    #[test]
    fn routes_doctor_json() {
        assert_eq!(
            parse(&["lnk", "doctor", "--json"]),
            Command::Doctor {
                output: OutputFormat::Json
            }
        );
    }

    #[test]
    fn routes_migrate_v1_read_only() {
        assert_eq!(
            parse(&["lnk", "migrate", "v1"]),
            Command::Migrate(MigrateCommand::V1 {
                write: false,
                yes: false,
            })
        );
    }

    #[test]
    fn routes_migrate_v1_write_yes() {
        assert_eq!(
            parse(&["lnk", "migrate", "v1", "--write", "--yes"]),
            Command::Migrate(MigrateCommand::V1 {
                write: true,
                yes: true,
            })
        );
    }

    #[test]
    fn rejects_unsupported_flags() {
        assert!(Cli::try_parse_from(["lnk", "apply", "--save"]).is_err());
    }

    #[test]
    fn rejects_incomplete_profile_script_target() {
        assert!(Cli::try_parse_from(["lnk", "run", ""]).is_err());
        assert!(Cli::try_parse_from(["lnk", "run", "work:"]).is_err());
        assert!(Cli::try_parse_from(["lnk", "run", ":setup"]).is_err());
        assert!(Cli::try_parse_from(["lnk", "run", "work:setup:again"]).is_err());
    }

    #[test]
    fn rejects_partial_migration_confirmation() {
        assert!(Cli::try_parse_from(["lnk", "migrate", "v1", "--write"]).is_err());
        assert!(Cli::try_parse_from(["lnk", "migrate", "v1", "--yes"]).is_err());
    }
}
