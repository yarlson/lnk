use std::{fs, io, path::Path};

use crate::{
    errors::Error,
    git,
    manifest::{self, FileMode, ProfileManifest},
    paths, state,
};

use super::{MIGRATION_COMMIT_MESSAGE, plan::MigrationPlan};

pub(super) fn apply(repo: &Path, plan: &MigrationPlan) -> Result<(), Error> {
    let state_path = paths::state_file()?;

    git::create_tag(repo, &plan.output.backup_tag)?;
    manifest::write_root_manifest(repo, &plan.root_manifest)?;

    for profile in &plan.profile_dirs {
        create_profile_dirs(repo, profile)?;
    }

    for file_move in &plan.file_moves {
        move_path(&file_move.source, &file_move.target)?;
        remove_empty_parents(repo, file_move.source.parent());
    }

    for (profile, manifest) in &plan.profile_manifests {
        write_profile_manifest(repo, profile, manifest)?;
    }

    for path in &plan.tracking_files {
        remove_file_if_exists(path)?;
    }

    for path in &plan.host_dirs {
        remove_dir_if_empty(path)?;
    }

    state::write_state(
        &state_path,
        &state::MachineState {
            active_profiles: plan.active_profiles.clone(),
            targets: Vec::new(),
        },
    )?;

    git::add_all(repo)?;
    git::commit(repo, MIGRATION_COMMIT_MESSAGE)?;

    Ok(())
}

fn create_profile_dirs(repo: &Path, profile: &str) -> Result<(), Error> {
    for child in ["files", "scripts"] {
        let path = repo.join("profiles").join(profile).join(child);
        fs::create_dir_all(&path).map_err(|error| Error::FileWriteFailed {
            path: path_to_string(&path),
            message: error.to_string(),
        })?;
    }

    Ok(())
}

fn move_path(source: &Path, target: &Path) -> Result<(), Error> {
    if let Some(parent) = target.parent() {
        fs::create_dir_all(parent).map_err(|error| Error::FileWriteFailed {
            path: path_to_string(parent),
            message: error.to_string(),
        })?;
    }

    fs::rename(source, target).map_err(|error| Error::FileWriteFailed {
        path: path_to_string(target),
        message: error.to_string(),
    })
}

fn write_profile_manifest(
    repo: &Path,
    profile: &str,
    manifest: &ProfileManifest,
) -> Result<(), Error> {
    let profile_dir = repo.join("profiles").join(profile);
    let path = profile_dir.join("profile.toml");

    fs::create_dir_all(&profile_dir).map_err(|error| Error::SchemaWriteFailed {
        path: path_to_string(&profile_dir),
        message: error.to_string(),
    })?;

    fs::write(&path, serialize_profile_manifest(manifest)).map_err(|error| {
        Error::SchemaWriteFailed {
            path: path_to_string(&path),
            message: error.to_string(),
        }
    })
}

fn serialize_profile_manifest(manifest: &ProfileManifest) -> String {
    let mut output = String::new();

    for (source, entry) in &manifest.files {
        output.push_str(&format!("[files.\"{}\"]\n", escape_toml_key(source)));

        if let Some(target) = &entry.target {
            output.push_str(&format!("target = \"{}\"\n", escape_toml_value(target)));
        }

        if entry.mode != FileMode::Symlink {
            output.push_str("mode = \"copy\"\n");
        }

        if let Some(overrides) = &entry.overrides {
            output.push_str(&format!(
                "overrides = \"{}\"\n",
                escape_toml_value(overrides)
            ));
        }

        output.push('\n');
    }

    output
}

fn remove_file_if_exists(path: &Path) -> Result<(), Error> {
    match fs::remove_file(path) {
        Ok(()) => Ok(()),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(()),
        Err(error) => Err(Error::FileWriteFailed {
            path: path_to_string(path),
            message: error.to_string(),
        }),
    }
}

fn remove_dir_if_empty(path: &Path) -> Result<(), Error> {
    match fs::remove_dir(path) {
        Ok(()) => Ok(()),
        Err(error)
            if matches!(
                error.kind(),
                io::ErrorKind::NotFound | io::ErrorKind::DirectoryNotEmpty
            ) =>
        {
            Ok(())
        }
        Err(error) => Err(Error::FileWriteFailed {
            path: path_to_string(path),
            message: error.to_string(),
        }),
    }
}

fn remove_empty_parents(repo: &Path, parent: Option<&Path>) {
    let Some(parent) = parent else {
        return;
    };

    if parent == repo || !parent.starts_with(repo) {
        return;
    }

    if fs::remove_dir(parent).is_ok() {
        remove_empty_parents(repo, parent.parent());
    }
}

fn escape_toml_key(value: &str) -> String {
    escape_toml_value(value)
}

fn escape_toml_value(value: &str) -> String {
    value.replace('\\', "\\\\").replace('"', "\\\"")
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
