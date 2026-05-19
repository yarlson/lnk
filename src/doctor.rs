use std::{collections::BTreeMap, io, path::Path};

use crate::{
    app::{DoctorIssueOutput, DoctorOutput, Output, StatusTargetStatus},
    errors::Error,
    manifest::{self, FileMode},
    ownership, paths,
    profile::{self, Profile, Profiles},
    state,
};

pub fn doctor() -> Result<Output, Error> {
    let repo = paths::repo_dir()?;
    let home = paths::home_dir()?;
    let state_file = paths::state_file()?;
    let mut issues = Vec::new();

    if let Err(error) = manifest::read_root_manifest(&repo) {
        issues.push(issue_from_error("root_schema_invalid", error));
        return Ok(Output::Doctor(DoctorOutput { issues }));
    }

    let profiles = discover_profiles(&repo, &mut issues)?;
    validate_profiles(&repo, &profiles, &mut issues);

    match state::read_state_if_exists(&state_file) {
        Ok(Some(machine_state)) => {
            if let Err(error) = ownership::plan(&repo, &machine_state.active_profiles) {
                issues.push(issue_from_error("profile_selection_invalid", error));
            }

            for target in &machine_state.targets {
                if let Err(error) = audit_state_target(&repo, &home, target, &mut issues) {
                    issues.push(issue_from_error("local_state_target_invalid", error));
                }
            }
        }
        Ok(None) => {}
        Err(error) => issues.push(issue_from_error("local_state_invalid", error)),
    }

    Ok(Output::Doctor(DoctorOutput { issues }))
}

fn discover_profiles(repo: &Path, issues: &mut Vec<DoctorIssueOutput>) -> Result<Profiles, Error> {
    let profiles_dir = repo.join("profiles");
    let entries = match profile::sorted_dir_entries(&profiles_dir) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(BTreeMap::new()),
        Err(error) => {
            return Err(Error::ProfileDiscoveryFailed {
                path: path_to_string(&profiles_dir),
                message: error.to_string(),
            });
        }
    };
    let mut profiles = BTreeMap::new();

    for entry in entries {
        let path = entry.path();
        let file_type = entry
            .file_type()
            .map_err(|error| Error::ProfileDiscoveryFailed {
                path: path_to_string(&path),
                message: error.to_string(),
            })?;

        if !file_type.is_dir() {
            continue;
        }

        let name = match entry.file_name().into_string() {
            Ok(name) => name,
            Err(_) => {
                issues.push(DoctorIssueOutput {
                    code: "profile_name_invalid".to_string(),
                    message: "profile name is invalid".to_string(),
                    path: Some(path_to_string(&path)),
                    profile: None,
                    target: None,
                    source: None,
                    status: None,
                });
                continue;
            }
        };

        match manifest::read_profile_manifest(&path) {
            Ok(manifest) => {
                profiles.insert(
                    name.clone(),
                    Profile {
                        name,
                        dir: path,
                        manifest,
                    },
                );
            }
            Err(error) => issues.push(DoctorIssueOutput {
                code: "profile_manifest_invalid".to_string(),
                message: error.message().to_string(),
                path: Some(path_to_string(&path.join("profile.toml"))),
                profile: Some(name),
                target: None,
                source: None,
                status: None,
            }),
        }
    }

    Ok(profiles)
}

fn validate_profiles(repo: &Path, profiles: &Profiles, issues: &mut Vec<DoctorIssueOutput>) {
    for profile in profiles.values() {
        if let Err(error) = ownership::file_claims(repo, profile) {
            issues.push(issue_from_error("profile_file_invalid", error));
        }

        if let Err(error) = ownership::script_metadata(profile) {
            issues.push(issue_from_error("profile_script_invalid", error));
        }
    }
}

fn audit_state_target(
    repo: &Path,
    home: &Path,
    target: &state::TargetState,
    issues: &mut Vec<DoctorIssueOutput>,
) -> Result<(), Error> {
    let source = repo.join(&target.source);

    if !source.exists() {
        issues.push(DoctorIssueOutput {
            code: "state_source_missing".to_string(),
            message: "local state source no longer exists".to_string(),
            path: Some(path_to_string(&source)),
            profile: Some(target.profile.clone()),
            target: Some(target.target.clone()),
            source: Some(target.source.clone()),
            status: None,
        });

        if target.mode == FileMode::Copy {
            return Ok(());
        }
    }

    let status = crate::status::target_status(repo, home, target)?;

    match (target.mode, status) {
        (FileMode::Symlink, StatusTargetStatus::MissingSymlink) => {
            issues.push(status_issue("symlink_missing", target, status));
        }
        (FileMode::Symlink, StatusTargetStatus::WrongSymlink) => {
            issues.push(status_issue("symlink_wrong", target, status));
        }
        (FileMode::Symlink, StatusTargetStatus::StaleSymlink) => {
            issues.push(status_issue("symlink_stale", target, status));
        }
        (
            FileMode::Copy,
            StatusTargetStatus::SourceChanged
            | StatusTargetStatus::LocalCopyDrift
            | StatusTargetStatus::DivergentCopy
            | StatusTargetStatus::UnknownCopyState,
        ) => issues.push(status_issue("copy_drift", target, status)),
        _ => {}
    }

    Ok(())
}

fn status_issue(
    code: &str,
    target: &state::TargetState,
    status: StatusTargetStatus,
) -> DoctorIssueOutput {
    DoctorIssueOutput {
        code: code.to_string(),
        message: "live target is unhealthy".to_string(),
        path: None,
        profile: Some(target.profile.clone()),
        target: Some(target.target.clone()),
        source: Some(target.source.clone()),
        status: Some(status),
    }
}

fn issue_from_error(code: &str, error: Error) -> DoctorIssueOutput {
    let message = error.message().to_string();

    match error {
        Error::InvalidTargetPath { path, reason } => DoctorIssueOutput {
            code: "target_path_invalid".to_string(),
            message: format!("target path is invalid: {reason}"),
            path: Some(path),
            profile: None,
            target: None,
            source: None,
            status: None,
        },
        Error::DuplicateTargetClaim {
            target,
            first_profile,
            second_profile,
        } => DoctorIssueOutput {
            code: "target_duplicate".to_string(),
            message: format!(
                "target is claimed by multiple profiles: {first_profile}, {second_profile}"
            ),
            path: None,
            profile: Some(second_profile),
            target: Some(target),
            source: None,
            status: None,
        },
        Error::InvalidOverride {
            profile,
            target,
            overrides,
        } => DoctorIssueOutput {
            code: "override_invalid".to_string(),
            message: format!("override does not match earlier target claim: {overrides}"),
            path: None,
            profile: Some(profile),
            target: Some(target),
            source: None,
            status: None,
        },
        Error::ManifestFileMissing { profile, source } => DoctorIssueOutput {
            code: "manifest_file_missing".to_string(),
            message: "profile manifest references a missing file".to_string(),
            path: None,
            profile: Some(profile),
            target: None,
            source: Some(source),
            status: None,
        },
        Error::ManifestScriptMissing { profile, script } => DoctorIssueOutput {
            code: "manifest_script_missing".to_string(),
            message: "profile manifest references a missing script".to_string(),
            path: None,
            profile: Some(profile),
            target: None,
            source: Some(script),
            status: None,
        },
        Error::InvalidScriptPlatform {
            profile,
            script,
            platform,
        } => DoctorIssueOutput {
            code: "script_platform_invalid".to_string(),
            message: format!("script platform is unsupported: {platform}"),
            path: None,
            profile: Some(profile),
            target: None,
            source: Some(script),
            status: None,
        },
        Error::UnknownProfile { profile } => DoctorIssueOutput {
            code: "active_profile_unknown".to_string(),
            message: "active profile does not exist".to_string(),
            path: None,
            profile: Some(profile),
            target: None,
            source: None,
            status: None,
        },
        Error::DuplicateProfileSelection { profile } => DoctorIssueOutput {
            code: "active_profile_duplicate".to_string(),
            message: "active profile is selected more than once".to_string(),
            path: None,
            profile: Some(profile),
            target: None,
            source: None,
            status: None,
        },
        Error::SchemaReadFailed { path, .. }
        | Error::SchemaInvalid { path, .. }
        | Error::UnsupportedSchemaFormat { path, .. }
        | Error::DuplicateStateTarget { target: path } => DoctorIssueOutput {
            code: code.to_string(),
            message,
            path: Some(path),
            profile: None,
            target: None,
            source: None,
            status: None,
        },
        _ => DoctorIssueOutput {
            code: code.to_string(),
            message,
            path: None,
            profile: None,
            target: None,
            source: None,
            status: None,
        },
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
