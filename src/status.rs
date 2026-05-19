use std::{fs, io, path::Path};

use crate::{
    app::{Output, StatusOutput, StatusTargetOutput, StatusTargetStatus},
    copy::{CopyDrift, classify_copy_drift, hash_file, path_from_home},
    errors::Error,
    git,
    manifest::{self, FileMode},
    paths, state,
    state::TargetState,
};

pub fn status() -> Result<Output, Error> {
    let repo = paths::repo_dir()?;
    let home = paths::home_dir()?;
    let state_file = paths::state_file()?;

    let _root_manifest = manifest::read_root_manifest(&repo)?;
    let machine_state = state::read_state_if_exists(&state_file)?.unwrap_or_default();
    let repo_changes = git::status_porcelain(&repo)?;
    let targets = machine_state
        .targets
        .iter()
        .map(|target| status_target(&repo, &home, target))
        .collect::<Result<Vec<_>, _>>()?;

    Ok(Output::Status(StatusOutput {
        active_profiles: machine_state.active_profiles,
        targets,
        repo_dirty: !repo_changes.is_empty(),
        repo_changes,
    }))
}

fn status_target(
    repo: &Path,
    home: &Path,
    target: &TargetState,
) -> Result<StatusTargetOutput, Error> {
    Ok(StatusTargetOutput {
        target: target.target.clone(),
        profile: target.profile.clone(),
        source: target.source.clone(),
        mode: mode_name(target.mode).to_string(),
        status: target_status(repo, home, target)?,
    })
}

pub(crate) fn target_status(
    repo: &Path,
    home: &Path,
    target: &TargetState,
) -> Result<StatusTargetStatus, Error> {
    match target.mode {
        FileMode::Symlink => symlink_status(repo, home, target),
        FileMode::Copy => copy_status(repo, home, target),
    }
}

fn symlink_status(
    repo: &Path,
    home: &Path,
    target: &TargetState,
) -> Result<StatusTargetStatus, Error> {
    let live_path = path_from_home(home, &target.target)?;
    let expected_source = repo.join(&target.source);

    match fs::symlink_metadata(&live_path) {
        Ok(metadata) if !metadata.file_type().is_symlink() => Ok(StatusTargetStatus::WrongSymlink),
        Ok(_) => {
            let link_target = fs::read_link(&live_path).map_err(|error| Error::FileReadFailed {
                path: path_to_string(&live_path),
                message: error.to_string(),
            })?;

            if link_target != expected_source {
                return Ok(StatusTargetStatus::WrongSymlink);
            }

            if !expected_source.exists() {
                return Ok(StatusTargetStatus::StaleSymlink);
            }

            Ok(StatusTargetStatus::Clean)
        }
        Err(error) if error.kind() == io::ErrorKind::NotFound => {
            Ok(StatusTargetStatus::MissingSymlink)
        }
        Err(error) => Err(Error::FileReadFailed {
            path: path_to_string(&live_path),
            message: error.to_string(),
        }),
    }
}

fn copy_status(
    repo: &Path,
    home: &Path,
    target: &TargetState,
) -> Result<StatusTargetStatus, Error> {
    let source_path = repo.join(&target.source);
    let live_path = path_from_home(home, &target.target)?;
    let source_hash = hash_file(&source_path)?;
    let live_hash = match fs::symlink_metadata(&live_path) {
        Ok(metadata) if metadata.file_type().is_file() => Some(hash_file(&live_path)?),
        Ok(_) => return Ok(StatusTargetStatus::WrongCopyType),
        Err(error) if error.kind() == io::ErrorKind::NotFound => {
            return Ok(StatusTargetStatus::MissingCopy);
        }
        Err(error) => {
            return Err(Error::FileReadFailed {
                path: path_to_string(&live_path),
                message: error.to_string(),
            });
        }
    };

    Ok(
        match classify_copy_drift(Some(target), &source_hash, live_hash.as_deref()) {
            CopyDrift::Clean => StatusTargetStatus::Clean,
            CopyDrift::SourceChanged => StatusTargetStatus::SourceChanged,
            CopyDrift::LocalDrift => StatusTargetStatus::LocalCopyDrift,
            CopyDrift::Divergent => StatusTargetStatus::DivergentCopy,
            CopyDrift::Unknown => StatusTargetStatus::UnknownCopyState,
        },
    )
}

fn mode_name(mode: FileMode) -> &'static str {
    match mode {
        FileMode::Symlink => "symlink",
        FileMode::Copy => "copy",
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
