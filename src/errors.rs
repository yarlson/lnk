#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Error {
    MissingHome,
    InvalidHome {
        path: String,
    },
    InvalidTargetPath {
        path: String,
        reason: &'static str,
    },
    SchemaReadFailed {
        path: String,
        message: String,
    },
    SchemaWriteFailed {
        path: String,
        message: String,
    },
    SchemaInvalid {
        path: String,
        message: String,
    },
    UnsupportedSchemaFormat {
        path: String,
        format: u32,
    },
    DuplicateStateTarget {
        target: String,
    },
    NoActiveProfiles,
    ProfileDiscoveryFailed {
        path: String,
        message: String,
    },
    FileReadFailed {
        path: String,
        message: String,
    },
    FileWriteFailed {
        path: String,
        message: String,
    },
    ApplyConflict {
        target: String,
        reason: String,
    },
    CopyDrift {
        target: String,
        drift: String,
    },
    InvalidProfileName {
        path: String,
    },
    UnknownProfile {
        profile: String,
    },
    DuplicateProfileSelection {
        profile: String,
    },
    ManifestFileMissing {
        profile: String,
        source: String,
    },
    ManifestScriptMissing {
        profile: String,
        script: String,
    },
    DuplicateTargetClaim {
        target: String,
        first_profile: String,
        second_profile: String,
    },
    InvalidOverride {
        profile: String,
        target: String,
        overrides: String,
    },
    InvalidScriptPlatform {
        profile: String,
        script: String,
        platform: String,
    },
    ScriptMissing {
        profile: String,
        script: String,
    },
    ScriptExecFailed {
        profile: String,
        script: String,
        path: String,
        message: String,
    },
    ScriptFailed {
        profile: String,
        script: String,
        code: Option<i32>,
    },
    MigrationConfirmationRequired,
    MigrationDirtyTree,
    MigrationInvalid {
        message: &'static str,
    },
    RepoCreateFailed {
        path: String,
        message: String,
    },
    UnrelatedGitRepo {
        path: String,
    },
    GitUnavailable,
    GitCommandFailed {
        command: String,
        stderr: String,
    },
}

impl Error {
    pub fn code(&self) -> &'static str {
        match self {
            Self::MissingHome => "path.home_missing",
            Self::InvalidHome { .. } => "path.home_invalid",
            Self::InvalidTargetPath { .. } => "path.target_invalid",
            Self::SchemaReadFailed { .. } => "schema.read_failed",
            Self::SchemaWriteFailed { .. } => "schema.write_failed",
            Self::SchemaInvalid { .. } => "schema.invalid",
            Self::UnsupportedSchemaFormat { .. } => "schema.unsupported_format",
            Self::DuplicateStateTarget { .. } => "state.duplicate_target",
            Self::NoActiveProfiles => "profile.none_active",
            Self::ProfileDiscoveryFailed { .. } => "profile.discovery_failed",
            Self::FileReadFailed { .. } => "file.read_failed",
            Self::FileWriteFailed { .. } => "file.write_failed",
            Self::ApplyConflict { .. } => "apply.conflict",
            Self::CopyDrift { .. } => "apply.copy_drift",
            Self::InvalidProfileName { .. } => "profile.name_invalid",
            Self::UnknownProfile { .. } => "profile.unknown",
            Self::DuplicateProfileSelection { .. } => "profile.duplicate_selection",
            Self::ManifestFileMissing { .. } => "profile.file_missing",
            Self::ManifestScriptMissing { .. } => "profile.script_missing",
            Self::DuplicateTargetClaim { .. } => "ownership.duplicate_target",
            Self::InvalidOverride { .. } => "ownership.override_invalid",
            Self::InvalidScriptPlatform { .. } => "script.platform_invalid",
            Self::ScriptMissing { .. } => "script.missing",
            Self::ScriptExecFailed { .. } => "script.exec_failed",
            Self::ScriptFailed { .. } => "script.failed",
            Self::MigrationConfirmationRequired => "migration.confirmation_required",
            Self::MigrationDirtyTree => "migration.dirty_tree",
            Self::MigrationInvalid { .. } => "migration.invalid",
            Self::RepoCreateFailed { .. } => "repo.create_failed",
            Self::UnrelatedGitRepo { .. } => "repo.unrelated_git",
            Self::GitUnavailable => "git.unavailable",
            Self::GitCommandFailed { .. } => "git.command_failed",
        }
    }

    pub fn message(&self) -> &'static str {
        match self {
            Self::MissingHome => "HOME is not set",
            Self::InvalidHome { .. } => "HOME must be an absolute path",
            Self::InvalidTargetPath { .. } => "target path is invalid",
            Self::SchemaReadFailed { .. } => "schema file could not be read",
            Self::SchemaWriteFailed { .. } => "schema file could not be written",
            Self::SchemaInvalid { .. } => "schema file is invalid",
            Self::UnsupportedSchemaFormat { .. } => "schema format is unsupported",
            Self::DuplicateStateTarget { .. } => "state target is duplicated",
            Self::NoActiveProfiles => "run 'lnk apply <profile...>' first",
            Self::ProfileDiscoveryFailed { .. } => "profiles could not be discovered",
            Self::FileReadFailed { .. } => "file could not be read",
            Self::FileWriteFailed { .. } => "file could not be written",
            Self::ApplyConflict { .. } => "apply conflict",
            Self::CopyDrift { .. } => "copy drift blocks apply",
            Self::InvalidProfileName { .. } => "profile name is invalid",
            Self::UnknownProfile { .. } => "profile does not exist",
            Self::DuplicateProfileSelection { .. } => "profile is selected more than once",
            Self::ManifestFileMissing { .. } => "profile manifest references a missing file",
            Self::ManifestScriptMissing { .. } => "profile manifest references a missing script",
            Self::DuplicateTargetClaim { .. } => "target is claimed by multiple profiles",
            Self::InvalidOverride { .. } => "override does not match an earlier target claim",
            Self::InvalidScriptPlatform { .. } => "script platform is unsupported",
            Self::ScriptMissing { .. } => "script does not exist",
            Self::ScriptExecFailed { .. } => "script could not be executed",
            Self::ScriptFailed { .. } => "script failed",
            Self::MigrationConfirmationRequired => "migration write requires --yes",
            Self::MigrationDirtyTree => "working tree must be clean before migration",
            Self::MigrationInvalid { message } => message,
            Self::RepoCreateFailed { .. } => "repo directory could not be created",
            Self::UnrelatedGitRepo { .. } => {
                "existing Git repo has commits but no lnk.toml; use 'lnk migrate v1' or an empty directory"
            }
            Self::GitUnavailable => "git is required but was not found",
            Self::GitCommandFailed { .. } => "git command failed",
        }
    }

    pub fn command(&self) -> Option<&'static str> {
        match self {
            Self::MissingHome
            | Self::InvalidHome { .. }
            | Self::InvalidTargetPath { .. }
            | Self::SchemaReadFailed { .. }
            | Self::SchemaWriteFailed { .. }
            | Self::SchemaInvalid { .. }
            | Self::UnsupportedSchemaFormat { .. }
            | Self::DuplicateStateTarget { .. }
            | Self::NoActiveProfiles
            | Self::ProfileDiscoveryFailed { .. }
            | Self::FileReadFailed { .. }
            | Self::FileWriteFailed { .. }
            | Self::ApplyConflict { .. }
            | Self::CopyDrift { .. }
            | Self::InvalidProfileName { .. }
            | Self::UnknownProfile { .. }
            | Self::DuplicateProfileSelection { .. }
            | Self::ManifestFileMissing { .. }
            | Self::ManifestScriptMissing { .. }
            | Self::DuplicateTargetClaim { .. }
            | Self::InvalidOverride { .. }
            | Self::InvalidScriptPlatform { .. }
            | Self::ScriptMissing { .. }
            | Self::ScriptExecFailed { .. }
            | Self::ScriptFailed { .. }
            | Self::MigrationConfirmationRequired
            | Self::MigrationDirtyTree
            | Self::MigrationInvalid { .. }
            | Self::RepoCreateFailed { .. }
            | Self::UnrelatedGitRepo { .. }
            | Self::GitUnavailable
            | Self::GitCommandFailed { .. } => None,
        }
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(self.message())
    }
}

impl std::error::Error for Error {}
