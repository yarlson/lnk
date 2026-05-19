use std::{
    collections::BTreeMap,
    fs, io,
    path::{Path, PathBuf},
};

mod mutation;

use crate::{
    app::{ApplyPlanAction, ApplyPlanOutput, Output},
    cli::ApplyCommand,
    copy::{CopyDrift, classify_copy_drift, hash_file},
    errors::Error,
    manifest,
    manifest::FileMode,
    ownership::{self, FileClaim},
    paths, state,
    state::{MachineState, TargetState},
};

pub fn apply(command: ApplyCommand) -> Result<Output, Error> {
    let repo = paths::repo_dir()?;
    let home = paths::home_dir()?;
    let state_file = paths::state_file()?;

    let _root_manifest = manifest::read_root_manifest(&repo)?;
    let machine_state = state::read_state_if_exists(&state_file)?;
    let profiles = selected_profiles(&command, machine_state.as_ref())?;
    let ownership = ownership::plan(&repo, &profiles)?;
    let targets = targets_by_path(machine_state.as_ref());
    let actions = plan_actions(&repo, &home, &ownership.files, &targets, command.backup)?;

    if !command.dry_run {
        reject_blocking_actions(&actions)?;
        mutation::apply_mutations(&repo, &home, &ownership.files, &targets, command.backup)?;
        let state =
            mutation::machine_state_for_claims(&repo, &home, profiles.clone(), &ownership.files)?;
        state::write_state(&state_file, &state)?;
    }

    Ok(Output::ApplyPlan(ApplyPlanOutput {
        profiles,
        dry_run: command.dry_run,
        backup: command.backup,
        actions,
    }))
}

fn reject_blocking_actions(actions: &[ApplyPlanAction]) -> Result<(), Error> {
    for action in actions {
        match action {
            ApplyPlanAction::Conflict { target, reason } => {
                return Err(Error::ApplyConflict {
                    target: target.clone(),
                    reason: reason.clone(),
                });
            }
            ApplyPlanAction::CopyDrift { target, drift, .. } => {
                return Err(Error::CopyDrift {
                    target: target.clone(),
                    drift: drift.clone(),
                });
            }
            ApplyPlanAction::Override { .. }
            | ApplyPlanAction::Backup { .. }
            | ApplyPlanAction::Symlink { .. }
            | ApplyPlanAction::Copy { .. } => {}
        }
    }

    Ok(())
}

fn selected_profiles(
    command: &ApplyCommand,
    machine_state: Option<&MachineState>,
) -> Result<Vec<String>, Error> {
    if !command.profiles.is_empty() {
        return Ok(command.profiles.clone());
    }

    let profiles = machine_state
        .map(|state| state.active_profiles.clone())
        .unwrap_or_default();

    if profiles.is_empty() {
        Err(Error::NoActiveProfiles)
    } else {
        Ok(profiles)
    }
}

fn targets_by_path(machine_state: Option<&MachineState>) -> BTreeMap<String, TargetState> {
    machine_state
        .map(|state| {
            state
                .targets
                .iter()
                .map(|target| (target.target.clone(), target.clone()))
                .collect()
        })
        .unwrap_or_default()
}

fn plan_actions(
    repo: &Path,
    home: &Path,
    claims: &[FileClaim],
    targets: &BTreeMap<String, TargetState>,
    backup: bool,
) -> Result<Vec<ApplyPlanAction>, Error> {
    let mut actions = Vec::new();

    for claim in claims {
        if let Some(overrides) = &claim.overrides {
            actions.push(ApplyPlanAction::Override {
                profile: claim.profile.clone(),
                target: claim.target.clone(),
                overrides: overrides.clone(),
            });
        }

        actions.extend(plan_claim(
            repo,
            home,
            claim,
            targets.get(&claim.target),
            backup,
        )?);
    }

    Ok(actions)
}

fn plan_claim(
    repo: &Path,
    home: &Path,
    claim: &FileClaim,
    previous: Option<&TargetState>,
    backup: bool,
) -> Result<Vec<ApplyPlanAction>, Error> {
    let target = paths::normalize_target_path(Path::new(&claim.target))?;
    let live_path = paths::target_path(home, &target)?;
    let live_target = live_target(&live_path)?;
    let target_state = classify_target(previous.is_some(), live_target.is_some(), backup);
    let mut actions = Vec::new();

    match target_state {
        TargetClassification::Missing | TargetClassification::Managed => {}
        TargetClassification::UnmanagedConflict => {
            actions.push(conflict(&claim.target, "unmanaged_target"));
            return Ok(actions);
        }
        TargetClassification::UnmanagedBackup => {
            actions.push(backup_action(&claim.target, "unmanaged_target"));
        }
    }

    match claim.mode {
        FileMode::Symlink => {
            actions.push(symlink_action(claim));
        }
        FileMode::Copy => {
            actions.extend(plan_copy(
                repo,
                claim,
                previous,
                live_target.as_ref(),
                backup,
            )?);
        }
    }

    Ok(actions)
}

fn plan_copy(
    repo: &Path,
    claim: &FileClaim,
    previous: Option<&TargetState>,
    live_target: Option<&LiveTarget>,
    backup: bool,
) -> Result<Vec<ApplyPlanAction>, Error> {
    let source_hash = hash_file(&repo.join(&claim.source))?;
    let live_hash = match live_target {
        Some(LiveTarget::File(path)) => Some(hash_file(path)?),
        Some(LiveTarget::Other) | None => None,
    };
    let drift = classify_copy_drift(previous, &source_hash, live_hash.as_deref());

    Ok(match drift {
        CopyDrift::Clean | CopyDrift::Unknown => vec![copy_action(claim, None)],
        CopyDrift::SourceChanged => vec![copy_action(claim, Some("source_changed"))],
        CopyDrift::LocalDrift if backup => vec![
            backup_action(&claim.target, "local_copy_drift"),
            copy_action(claim, Some("local_copy_drift")),
        ],
        CopyDrift::Divergent if backup => vec![
            backup_action(&claim.target, "divergent_copy"),
            copy_action(claim, Some("divergent_copy")),
        ],
        CopyDrift::LocalDrift => vec![copy_drift_action(claim, "local_copy_drift")],
        CopyDrift::Divergent => vec![copy_drift_action(claim, "divergent_copy")],
    })
}

pub(super) fn live_target(path: &Path) -> Result<Option<LiveTarget>, Error> {
    match fs::symlink_metadata(path) {
        Ok(metadata) if metadata.file_type().is_file() => {
            Ok(Some(LiveTarget::File(path.to_path_buf())))
        }
        Ok(_) => Ok(Some(LiveTarget::Other)),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(error) => Err(Error::FileReadFailed {
            path: path_to_string(path),
            message: error.to_string(),
        }),
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(super) enum LiveTarget {
    File(PathBuf),
    Other,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum TargetClassification {
    Missing,
    Managed,
    UnmanagedConflict,
    UnmanagedBackup,
}

fn classify_target(managed: bool, live_exists: bool, backup: bool) -> TargetClassification {
    match (managed, live_exists, backup) {
        (_, false, _) => TargetClassification::Missing,
        (true, true, _) => TargetClassification::Managed,
        (false, true, true) => TargetClassification::UnmanagedBackup,
        (false, true, false) => TargetClassification::UnmanagedConflict,
    }
}

fn symlink_action(claim: &FileClaim) -> ApplyPlanAction {
    ApplyPlanAction::Symlink {
        profile: claim.profile.clone(),
        source: claim.source.clone(),
        target: claim.target.clone(),
    }
}

fn copy_action(claim: &FileClaim, drift: Option<&str>) -> ApplyPlanAction {
    ApplyPlanAction::Copy {
        profile: claim.profile.clone(),
        source: claim.source.clone(),
        target: claim.target.clone(),
        drift: drift.map(str::to_string),
    }
}

fn copy_drift_action(claim: &FileClaim, drift: &str) -> ApplyPlanAction {
    ApplyPlanAction::CopyDrift {
        profile: claim.profile.clone(),
        source: claim.source.clone(),
        target: claim.target.clone(),
        drift: drift.to_string(),
    }
}

fn backup_action(target: &str, reason: &str) -> ApplyPlanAction {
    ApplyPlanAction::Backup {
        target: target.to_string(),
        reason: reason.to_string(),
    }
}

fn conflict(target: &str, reason: &str) -> ApplyPlanAction {
    ApplyPlanAction::Conflict {
        target: target.to_string(),
        reason: reason.to_string(),
    }
}

pub(super) fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}

#[cfg(test)]
mod tests {
    use super::{TargetClassification, classify_target};

    #[test]
    fn classifies_unmanaged_live_targets_as_conflict_or_backup() {
        assert_eq!(
            classify_target(false, false, false),
            TargetClassification::Missing
        );
        assert_eq!(
            classify_target(true, true, false),
            TargetClassification::Managed
        );
        assert_eq!(
            classify_target(false, true, false),
            TargetClassification::UnmanagedConflict
        );
        assert_eq!(
            classify_target(false, true, true),
            TargetClassification::UnmanagedBackup
        );
    }
}
