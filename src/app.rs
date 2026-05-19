use serde::Serialize;

use crate::{
    apply, cli::Command, doctor, errors::Error, git, migrate, profile, repo, scripts, status,
};

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Output {
    Empty,
    ApplyPlan(ApplyPlanOutput),
    Doctor(DoctorOutput),
    MigrationPlan(MigrationPlanOutput),
    Profiles(ProfilesOutput),
    Status(StatusOutput),
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct ApplyPlanOutput {
    pub profiles: Vec<String>,
    pub dry_run: bool,
    pub backup: bool,
    pub actions: Vec<ApplyPlanAction>,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
#[serde(tag = "action", rename_all = "snake_case")]
pub enum ApplyPlanAction {
    Override {
        profile: String,
        target: String,
        overrides: String,
    },
    Backup {
        target: String,
        reason: String,
    },
    Conflict {
        target: String,
        reason: String,
    },
    CopyDrift {
        profile: String,
        source: String,
        target: String,
        drift: String,
    },
    Symlink {
        profile: String,
        source: String,
        target: String,
    },
    Copy {
        profile: String,
        source: String,
        target: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        drift: Option<String>,
    },
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct DoctorOutput {
    pub issues: Vec<DoctorIssueOutput>,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct DoctorIssueOutput {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub path: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub profile: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<StatusTargetStatus>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MigrationPlanOutput {
    pub backup_tag: String,
    pub commit_message: String,
    pub local_profiles: Vec<String>,
    pub actions: Vec<MigrationActionOutput>,
    pub wrote: bool,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum MigrationActionOutput {
    WriteRootManifest,
    MoveFile {
        source: String,
        target: String,
    },
    MoveScript {
        source: String,
        target: String,
    },
    WriteOverride {
        profile: String,
        target: String,
        overrides: String,
    },
    RemoveTrackingFile {
        path: String,
    },
    RemoveEmptyHostDir {
        path: String,
    },
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct ProfilesOutput {
    pub profiles: Vec<String>,
    pub active_profiles: Vec<String>,
    #[serde(skip)]
    pub local_state_exists: bool,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct StatusOutput {
    pub active_profiles: Vec<String>,
    pub targets: Vec<StatusTargetOutput>,
    pub repo_dirty: bool,
    pub repo_changes: Vec<String>,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct StatusTargetOutput {
    pub target: String,
    pub profile: String,
    pub source: String,
    pub mode: String,
    pub status: StatusTargetStatus,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum StatusTargetStatus {
    Clean,
    MissingSymlink,
    StaleSymlink,
    WrongSymlink,
    MissingCopy,
    WrongCopyType,
    SourceChanged,
    LocalCopyDrift,
    DivergentCopy,
    UnknownCopyState,
}

pub type Result<T> = std::result::Result<T, Error>;

pub fn dispatch(command: Command) -> Result<Output> {
    if command.requires_git() {
        git::preflight()?;
    }

    match command {
        Command::Init => repo::init(),
        Command::Profiles { .. } => profile::profiles(),
        Command::Apply(command) => apply::apply(command),
        Command::Status { .. } => status::status(),
        Command::Run(command) => scripts::run(command),
        Command::Doctor { .. } => doctor::doctor(),
        Command::Migrate(command) => migrate::migrate(command),
    }
}
