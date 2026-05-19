use std::{
    collections::{BTreeMap, BTreeSet},
    fs, io,
    path::{Path, PathBuf},
    process::Command,
};

use crate::{
    app::{MigrationActionOutput, MigrationPlanOutput},
    errors::Error,
    manifest::{FileEntry, FileMode, ProfileManifest, RootManifest},
    paths,
};

use super::{BACKUP_TAG_PREFIX, MIGRATION_COMMIT_MESSAGE};

pub(super) struct MigrationPlan {
    pub(super) output: MigrationPlanOutput,
    pub(super) root_manifest: RootManifest,
    pub(super) profile_dirs: BTreeSet<String>,
    pub(super) file_moves: Vec<FileMove>,
    pub(super) profile_manifests: BTreeMap<String, ProfileManifest>,
    pub(super) tracking_files: Vec<PathBuf>,
    pub(super) host_dirs: Vec<PathBuf>,
    pub(super) active_profiles: Vec<String>,
}

impl MigrationPlan {
    pub(super) fn build(repo: &Path, timestamp: &str) -> Result<Self, Error> {
        let common = read_tracking_file(repo, ".lnk")?.unwrap_or_default();
        let hosts = read_host_tracking(repo)?;
        let host_dirs = read_host_dirs(repo)?;
        let bootstrap = repo.join("bootstrap.sh").exists();
        let active_profiles = active_profiles(&hosts)?;
        let profile_dirs = profile_dirs(&hosts);
        let mut actions = Vec::new();
        let mut file_moves = Vec::new();
        let mut profile_manifests = BTreeMap::new();

        if common.is_empty() && hosts.is_empty() && !bootstrap {
            return Err(Error::MigrationInvalid {
                message: "no v1 migration inputs found",
            });
        }

        actions.push(MigrationActionOutput::WriteRootManifest);
        collect_base_moves(repo, &common, &mut actions, &mut file_moves)?;
        collect_host_moves(
            repo,
            &common,
            &hosts,
            &mut actions,
            &mut file_moves,
            &mut profile_manifests,
        )?;
        collect_bootstrap_move(repo, bootstrap, &mut actions, &mut file_moves)?;
        ensure_empty_profiles_are_persisted(
            repo,
            &profile_dirs,
            &file_moves,
            &mut profile_manifests,
        );
        collect_cleanup_actions(repo, &hosts, &host_dirs, &mut actions);

        let output = MigrationPlanOutput {
            backup_tag: format!("{BACKUP_TAG_PREFIX}-{timestamp}"),
            commit_message: MIGRATION_COMMIT_MESSAGE.to_string(),
            local_profiles: active_profiles.clone(),
            actions,
            wrote: false,
        };

        Ok(Self {
            output,
            root_manifest: RootManifest {
                name: Some(default_repo_name(repo)),
            },
            profile_dirs,
            file_moves,
            profile_manifests,
            tracking_files: tracking_files(repo, &hosts),
            host_dirs,
            active_profiles,
        })
    }
}

fn profile_dirs(hosts: &BTreeMap<String, HostTracking>) -> BTreeSet<String> {
    let mut profiles = BTreeSet::from(["base".to_string()]);
    profiles.extend(hosts.keys().cloned());
    profiles
}

fn ensure_empty_profiles_are_persisted(
    repo: &Path,
    profile_dirs: &BTreeSet<String>,
    file_moves: &[FileMove],
    profile_manifests: &mut BTreeMap<String, ProfileManifest>,
) {
    for profile in profile_dirs {
        if profile_manifests.contains_key(profile) {
            continue;
        }

        let profile_dir = repo.join("profiles").join(profile);
        let has_migrated_content = file_moves
            .iter()
            .any(|file_move| file_move.target.starts_with(&profile_dir));

        if !has_migrated_content {
            profile_manifests.insert(profile.clone(), ProfileManifest::default());
        }
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(super) struct FileMove {
    pub(super) source: PathBuf,
    pub(super) target: PathBuf,
}

#[derive(Clone, Debug, Eq, PartialEq)]
struct HostTracking {
    name: String,
    paths: Vec<String>,
}

fn collect_base_moves(
    repo: &Path,
    common: &[String],
    actions: &mut Vec<MigrationActionOutput>,
    file_moves: &mut Vec<FileMove>,
) -> Result<(), Error> {
    for path in common {
        let source = repo.join(path);
        let target = repo.join("profiles/base/files").join(path);
        push_file_move(repo, actions, file_moves, source, target)?;
    }

    Ok(())
}

fn collect_host_moves(
    repo: &Path,
    common: &[String],
    hosts: &BTreeMap<String, HostTracking>,
    actions: &mut Vec<MigrationActionOutput>,
    file_moves: &mut Vec<FileMove>,
    profile_manifests: &mut BTreeMap<String, ProfileManifest>,
) -> Result<(), Error> {
    let common_targets = common.iter().cloned().collect::<BTreeSet<_>>();

    for host in hosts.values() {
        let mut manifest = ProfileManifest::default();

        for path in &host.paths {
            let source = repo.join(format!("{}.lnk", host.name)).join(path);
            let target = repo
                .join("profiles")
                .join(&host.name)
                .join("files")
                .join(path);

            if common_targets.contains(path) {
                manifest.files.insert(
                    path.clone(),
                    FileEntry {
                        target: None,
                        mode: FileMode::Symlink,
                        overrides: Some("base".to_string()),
                    },
                );
                actions.push(MigrationActionOutput::WriteOverride {
                    profile: host.name.clone(),
                    target: path.clone(),
                    overrides: "base".to_string(),
                });
            }

            push_file_move(repo, actions, file_moves, source, target)?;
        }

        if !manifest.files.is_empty() {
            profile_manifests.insert(host.name.clone(), manifest);
        }
    }

    Ok(())
}

fn collect_bootstrap_move(
    repo: &Path,
    bootstrap: bool,
    actions: &mut Vec<MigrationActionOutput>,
    file_moves: &mut Vec<FileMove>,
) -> Result<(), Error> {
    if !bootstrap {
        return Ok(());
    }

    let source = repo.join("bootstrap.sh");
    let target = repo.join("profiles/base/scripts/setup");
    validate_move(file_moves, &source, &target)?;
    actions.push(MigrationActionOutput::MoveScript {
        source: "bootstrap.sh".to_string(),
        target: "profiles/base/scripts/setup".to_string(),
    });
    file_moves.push(FileMove { source, target });

    Ok(())
}

fn collect_cleanup_actions(
    repo: &Path,
    hosts: &BTreeMap<String, HostTracking>,
    host_dirs: &[PathBuf],
    actions: &mut Vec<MigrationActionOutput>,
) {
    if repo.join(".lnk").exists() {
        actions.push(MigrationActionOutput::RemoveTrackingFile {
            path: ".lnk".to_string(),
        });
    }

    for host in hosts.values() {
        actions.push(MigrationActionOutput::RemoveTrackingFile {
            path: format!(".lnk.{}", host.name),
        });
    }

    for host_dir in host_dirs {
        actions.push(MigrationActionOutput::RemoveEmptyHostDir {
            path: path_to_string(
                host_dir
                    .strip_prefix(repo)
                    .expect("host dir should be inside repo"),
            ),
        });
    }
}

fn read_host_tracking(repo: &Path) -> Result<BTreeMap<String, HostTracking>, Error> {
    let mut hosts = BTreeMap::new();

    for entry in sorted_dir_entries(repo)? {
        let path = entry.path();
        let Some(file_name) = path.file_name().and_then(|name| name.to_str()) else {
            continue;
        };
        let Some(host) = file_name.strip_prefix(".lnk.") else {
            continue;
        };

        if host.is_empty() {
            continue;
        }

        let paths = read_tracking_file(repo, file_name)?.unwrap_or_default();
        hosts.insert(
            host.to_string(),
            HostTracking {
                name: host.to_string(),
                paths,
            },
        );
    }

    Ok(hosts)
}

fn read_tracking_file(repo: &Path, name: &str) -> Result<Option<Vec<String>>, Error> {
    let path = repo.join(name);
    let contents = match fs::read_to_string(&path) {
        Ok(contents) => contents,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(None),
        Err(error) => {
            return Err(Error::FileReadFailed {
                path: path_to_string(&path),
                message: error.to_string(),
            });
        }
    };

    let mut paths = Vec::new();
    let mut seen = BTreeSet::new();

    for line in contents.lines() {
        let line = line.trim();

        if line.is_empty() {
            continue;
        }

        let normalized = paths::normalize_target_path(Path::new(line))?;
        let path = path_to_string(normalized.as_path());

        if !seen.insert(path.clone()) {
            return Err(Error::MigrationInvalid {
                message: "v1 tracking file contains duplicate paths",
            });
        }

        paths.push(path);
    }

    Ok(Some(paths))
}

fn read_host_dirs(repo: &Path) -> Result<Vec<PathBuf>, Error> {
    let mut dirs = Vec::new();

    for entry in sorted_dir_entries(repo)? {
        let path = entry.path();
        let file_type = entry
            .file_type()
            .map_err(|error| discovery_error(&path, error))?;

        if !file_type.is_dir() {
            continue;
        }

        let Some(file_name) = path.file_name().and_then(|name| name.to_str()) else {
            continue;
        };

        if file_name.ends_with(".lnk") && file_name.len() > ".lnk".len() {
            dirs.push(path);
        }
    }

    Ok(dirs)
}

fn sorted_dir_entries(path: &Path) -> Result<Vec<fs::DirEntry>, Error> {
    let mut entries = fs::read_dir(path)
        .map_err(|error| discovery_error(path, error))?
        .collect::<Result<Vec<_>, io::Error>>()
        .map_err(|error| discovery_error(path, error))?;
    entries.sort_by_key(|entry| entry.path());

    Ok(entries)
}

fn push_file_move(
    repo: &Path,
    actions: &mut Vec<MigrationActionOutput>,
    file_moves: &mut Vec<FileMove>,
    source: PathBuf,
    target: PathBuf,
) -> Result<(), Error> {
    validate_move(file_moves, &source, &target)?;
    actions.push(MigrationActionOutput::MoveFile {
        source: path_to_string(
            source
                .strip_prefix(repo)
                .expect("migration source should be inside repo"),
        ),
        target: path_to_string(
            target
                .strip_prefix(repo)
                .expect("migration target should be inside repo"),
        ),
    });
    file_moves.push(FileMove { source, target });

    Ok(())
}

fn validate_move(file_moves: &[FileMove], source: &Path, target: &Path) -> Result<(), Error> {
    if !source.is_file() {
        return Err(Error::MigrationInvalid {
            message: "v1 tracking file references a missing source file",
        });
    }

    if target.exists()
        || file_moves
            .iter()
            .any(|file_move| file_move.target == target)
    {
        return Err(Error::MigrationInvalid {
            message: "migration target already exists",
        });
    }

    Ok(())
}

fn tracking_files(repo: &Path, hosts: &BTreeMap<String, HostTracking>) -> Vec<PathBuf> {
    let mut files = Vec::new();

    if repo.join(".lnk").exists() {
        files.push(repo.join(".lnk"));
    }

    for host in hosts.keys() {
        files.push(repo.join(format!(".lnk.{host}")));
    }

    files
}

fn active_profiles(hosts: &BTreeMap<String, HostTracking>) -> Result<Vec<String>, Error> {
    let mut profiles = vec!["base".to_string()];

    if hosts.is_empty() {
        return Ok(profiles);
    }

    let hostname = current_hostname()?;

    if hosts.contains_key(&hostname) {
        profiles.push(hostname);
    }

    Ok(profiles)
}

fn current_hostname() -> Result<String, Error> {
    if let Some(hostname) = std::env::var("HOSTNAME")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
    {
        return Ok(hostname);
    }

    let output = Command::new("hostname")
        .output()
        .map_err(|error| Error::MigrationInvalid {
            message: if error.kind() == io::ErrorKind::NotFound {
                "hostname command is required for migration"
            } else {
                "hostname command failed"
            },
        })?;

    if !output.status.success() {
        return Err(Error::MigrationInvalid {
            message: "hostname command failed",
        });
    }

    let hostname = String::from_utf8_lossy(&output.stdout).trim().to_string();
    if hostname.is_empty() {
        return Err(Error::MigrationInvalid {
            message: "hostname command returned an empty name",
        });
    }

    Ok(hostname)
}

fn default_repo_name(repo: &Path) -> String {
    repo.file_name()
        .filter(|name| !name.is_empty())
        .map(|name| name.to_string_lossy().into_owned())
        .unwrap_or_else(|| "lnk".to_string())
}

fn discovery_error(path: &Path, error: io::Error) -> Error {
    Error::ProfileDiscoveryFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
