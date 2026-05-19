use std::{
    collections::BTreeMap,
    fs, io,
    path::{Path, PathBuf},
};

use crate::{
    app::{Output, ProfilesOutput},
    errors::Error,
    manifest,
    manifest::ProfileManifest,
    paths, state,
};

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct Profile {
    pub(crate) name: String,
    pub(crate) dir: PathBuf,
    pub(crate) manifest: ProfileManifest,
}

pub(crate) type Profiles = BTreeMap<String, Profile>;

pub fn profiles() -> Result<Output, Error> {
    let repo = paths::repo_dir()?;
    let _root_manifest = manifest::read_root_manifest(&repo)?;
    let profiles = discover(&repo)?;
    let machine_state = state::read_state_if_exists(&paths::state_file()?)?;

    Ok(Output::Profiles(ProfilesOutput {
        profiles: profiles.keys().cloned().collect(),
        active_profiles: machine_state
            .as_ref()
            .map(|state| state.active_profiles.clone())
            .unwrap_or_default(),
        local_state_exists: machine_state.is_some(),
    }))
}

pub(crate) fn discover(repo: &Path) -> Result<Profiles, Error> {
    let profiles_dir = repo.join("profiles");
    let entries = match sorted_dir_entries(&profiles_dir) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(BTreeMap::new()),
        Err(error) => return Err(discovery_error(&profiles_dir, error)),
    };

    let mut profiles = BTreeMap::new();

    for entry in entries {
        let file_type = entry
            .file_type()
            .map_err(|error| discovery_error(&entry.path(), error))?;

        if !file_type.is_dir() {
            continue;
        }

        let path = entry.path();
        let name = entry
            .file_name()
            .into_string()
            .map_err(|_| Error::InvalidProfileName {
                path: path_to_string(&path),
            })?;
        let manifest = manifest::read_profile_manifest(&path)?;

        profiles.insert(
            name.clone(),
            Profile {
                name,
                dir: path,
                manifest,
            },
        );
    }

    Ok(profiles)
}

pub(crate) fn sorted_dir_entries(path: &Path) -> Result<Vec<fs::DirEntry>, io::Error> {
    let mut entries = fs::read_dir(path)?.collect::<Result<Vec<_>, io::Error>>()?;
    entries.sort_by_key(|entry| entry.path());

    Ok(entries)
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
