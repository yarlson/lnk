use std::{
    collections::BTreeMap,
    fs, io,
    path::{Path, PathBuf},
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use crate::{
    apply::{LiveTarget, live_target, path_to_string},
    copy::{CopyDrift, classify_copy_drift, hash_file, path_from_home},
    errors::Error,
    manifest::FileMode,
    ownership::FileClaim,
    state::{MachineState, TargetState},
};

pub(super) fn apply_mutations(
    repo: &Path,
    home: &Path,
    claims: &[FileClaim],
    targets: &BTreeMap<String, TargetState>,
    backup: bool,
) -> Result<(), Error> {
    for claim in claims {
        match claim.mode {
            FileMode::Symlink => {
                apply_symlink(repo, home, claim, targets.get(&claim.target), backup)?
            }
            FileMode::Copy => apply_copy(repo, home, claim, targets.get(&claim.target), backup)?,
        }
    }

    Ok(())
}

pub(super) fn machine_state_for_claims(
    repo: &Path,
    home: &Path,
    profiles: Vec<String>,
    claims: &[FileClaim],
) -> Result<MachineState, Error> {
    let mut targets = Vec::new();

    for claim in claims {
        let (source_hash, applied_hash) = match claim.mode {
            FileMode::Symlink => (None, None),
            FileMode::Copy => {
                let source_hash = hash_file(&repo.join(&claim.source))?;
                let applied_hash = hash_file(&path_from_home(home, &claim.target)?)?;

                (Some(source_hash), Some(applied_hash))
            }
        };

        targets.push(TargetState {
            target: claim.target.clone(),
            profile: claim.profile.clone(),
            source: claim.source.clone(),
            mode: claim.mode,
            source_hash,
            applied_hash,
        });
    }

    Ok(MachineState {
        active_profiles: profiles,
        targets,
    })
}

fn apply_symlink(
    repo: &Path,
    home: &Path,
    claim: &FileClaim,
    previous: Option<&TargetState>,
    backup: bool,
) -> Result<(), Error> {
    let live_path = path_from_home(home, &claim.target)?;

    prepare_target(
        repo,
        &live_path,
        claim,
        previous,
        backup,
        "managed_target_changed",
    )?;
    create_parent_dir(&live_path)?;
    create_file_symlink(&repo.join(&claim.source), &live_path)
        .map_err(|error| write_error(&live_path, error))
}

fn apply_copy(
    repo: &Path,
    home: &Path,
    claim: &FileClaim,
    previous: Option<&TargetState>,
    backup: bool,
) -> Result<(), Error> {
    let live_path = path_from_home(home, &claim.target)?;
    let live_target = live_target(&live_path)?;
    let source_path = repo.join(&claim.source);
    let source_hash = hash_file(&source_path)?;
    let live_hash = match &live_target {
        Some(LiveTarget::File(path)) => Some(hash_file(path)?),
        Some(LiveTarget::Other) | None => None,
    };
    let drift = classify_copy_drift(previous, &source_hash, live_hash.as_deref());

    match drift {
        CopyDrift::Clean | CopyDrift::SourceChanged | CopyDrift::Unknown => {}
        CopyDrift::LocalDrift | CopyDrift::Divergent if backup => {
            backup_live_target(&live_path)?;
        }
        CopyDrift::LocalDrift => {
            return Err(Error::CopyDrift {
                target: claim.target.clone(),
                drift: "local_copy_drift".to_string(),
            });
        }
        CopyDrift::Divergent => {
            return Err(Error::CopyDrift {
                target: claim.target.clone(),
                drift: "divergent_copy".to_string(),
            });
        }
    }

    if !matches!(drift, CopyDrift::LocalDrift | CopyDrift::Divergent) {
        prepare_target(
            repo,
            &live_path,
            claim,
            previous,
            backup,
            "managed_target_changed",
        )?;
    }

    create_parent_dir(&live_path)?;
    copy_via_temp(&source_path, &live_path)
}

fn prepare_target(
    repo: &Path,
    live_path: &Path,
    claim: &FileClaim,
    previous: Option<&TargetState>,
    backup: bool,
    changed_reason: &str,
) -> Result<(), Error> {
    let live_target = live_target(live_path)?;

    match (previous, live_target) {
        (_, None) => Ok(()),
        (None, Some(_)) if backup => backup_live_target(live_path),
        (None, Some(_)) => Err(Error::ApplyConflict {
            target: claim.target.clone(),
            reason: "unmanaged_target".to_string(),
        }),
        (Some(previous), Some(_)) if target_is_still_managed(repo, live_path, previous)? => {
            remove_live_target(live_path)
        }
        (Some(_), Some(_)) if backup => backup_live_target(live_path),
        (Some(_), Some(_)) => Err(Error::ApplyConflict {
            target: claim.target.clone(),
            reason: changed_reason.to_string(),
        }),
    }
}

fn target_is_still_managed(
    repo: &Path,
    live_path: &Path,
    previous: &TargetState,
) -> Result<bool, Error> {
    match previous.mode {
        FileMode::Symlink => match fs::read_link(live_path) {
            Ok(target) => Ok(target == repo.join(&previous.source)),
            Err(error) if error.kind() == io::ErrorKind::InvalidInput => Ok(false),
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(false),
            Err(error) => Err(Error::FileReadFailed {
                path: path_to_string(live_path),
                message: error.to_string(),
            }),
        },
        FileMode::Copy => {
            let Some(applied_hash) = previous.applied_hash.as_deref() else {
                return Ok(false);
            };

            match live_target(live_path)? {
                Some(LiveTarget::File(path)) => Ok(hash_file(&path)? == applied_hash),
                Some(LiveTarget::Other) | None => Ok(false),
            }
        }
    }
}

fn create_parent_dir(path: &Path) -> Result<(), Error> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|error| write_error(parent, error))?;
    }

    Ok(())
}

fn remove_live_target(path: &Path) -> Result<(), Error> {
    fs::remove_file(path).map_err(|error| write_error(path, error))
}

fn backup_live_target(path: &Path) -> Result<(), Error> {
    let backup = next_backup_path(path)?;

    fs::rename(path, &backup).map_err(|error| write_error(path, error))
}

fn next_backup_path(path: &Path) -> Result<PathBuf, Error> {
    loop {
        let backup = backup_path(path, SystemTime::now())?;

        if !backup.exists() {
            return Ok(backup);
        }

        std::thread::sleep(Duration::from_secs(1));
    }
}

fn backup_path(path: &Path, now: SystemTime) -> Result<PathBuf, Error> {
    let timestamp = backup_timestamp(now)?;
    let file_name = path
        .file_name()
        .ok_or_else(|| Error::FileWriteFailed {
            path: path_to_string(path),
            message: "target path has no file name".to_string(),
        })?
        .to_string_lossy();

    Ok(path.with_file_name(format!("{file_name}.lnk-backup-{timestamp}")))
}

fn backup_timestamp(now: SystemTime) -> Result<String, Error> {
    let seconds = now
        .duration_since(UNIX_EPOCH)
        .map_err(|error| Error::FileWriteFailed {
            path: "<time>".to_string(),
            message: error.to_string(),
        })?
        .as_secs() as i64;
    let days = seconds.div_euclid(86_400);
    let seconds_of_day = seconds.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    let hour = seconds_of_day / 3_600;
    let minute = (seconds_of_day % 3_600) / 60;
    let second = seconds_of_day % 60;

    Ok(format!(
        "{year:04}{month:02}{day:02}-{hour:02}{minute:02}{second:02}"
    ))
}

fn civil_from_days(days_since_unix_epoch: i64) -> (i64, i64, i64) {
    let days = days_since_unix_epoch + 719_468;
    let era = days.div_euclid(146_097);
    let day_of_era = days - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1_460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let mut year = year_of_era + era * 400;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_prime = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_prime + 2) / 5 + 1;
    let month = month_prime + if month_prime < 10 { 3 } else { -9 };

    year += if month <= 2 { 1 } else { 0 };

    (year, month, day)
}

fn copy_via_temp(source: &Path, target: &Path) -> Result<(), Error> {
    let parent = target.parent().ok_or_else(|| Error::FileWriteFailed {
        path: path_to_string(target),
        message: "target path has no parent".to_string(),
    })?;
    let file_name = target
        .file_name()
        .ok_or_else(|| Error::FileWriteFailed {
            path: path_to_string(target),
            message: "target path has no file name".to_string(),
        })?
        .to_string_lossy();
    let temp = parent.join(format!(".{file_name}.lnk-tmp-{}", std::process::id()));

    fs::copy(source, &temp).map_err(|error| write_error(&temp, error))?;
    fs::rename(&temp, target).map_err(|error| {
        let _ = fs::remove_file(&temp);
        write_error(target, error)
    })?;

    Ok(())
}

#[cfg(unix)]
fn create_file_symlink(source: &Path, target: &Path) -> io::Result<()> {
    std::os::unix::fs::symlink(source, target)
}

#[cfg(windows)]
fn create_file_symlink(source: &Path, target: &Path) -> io::Result<()> {
    std::os::windows::fs::symlink_file(source, target)
}

fn write_error(path: &Path, error: io::Error) -> Error {
    Error::FileWriteFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}
