use std::{fs, io, path::Path};

use crate::{app::Output, errors::Error, git, manifest, manifest::RootManifest, paths};

const INITIAL_COMMIT_MESSAGE: &str = "lnk: initialize v2 setup repo";
const INITIAL_PATHS: &[&str] = &["lnk.toml"];

pub fn init() -> Result<Output, Error> {
    let repo = paths::repo_dir()?;

    create_dir_all(&repo)?;

    let is_git_repo = git::is_repo(&repo)?;
    let has_commits = is_git_repo && git::has_commits(&repo)?;
    let manifest_exists = repo.join("lnk.toml").exists();

    if is_git_repo && has_commits && !manifest_exists {
        return Err(Error::UnrelatedGitRepo {
            path: path_to_string(&repo),
        });
    }

    ensure_root_manifest(&repo, manifest_exists)?;
    create_dir_all(&repo.join("profiles").join("base").join("files"))?;
    create_dir_all(&repo.join("profiles").join("base").join("scripts"))?;

    if !is_git_repo {
        git::init(&repo)?;
    }

    if !has_commits {
        git::add_paths(&repo, INITIAL_PATHS)?;
        git::commit_paths(&repo, INITIAL_COMMIT_MESSAGE, INITIAL_PATHS)?;
    }

    Ok(Output::Empty)
}

fn ensure_root_manifest(repo: &Path, exists: bool) -> Result<(), Error> {
    if exists {
        manifest::read_root_manifest(repo)?;
        return Ok(());
    }

    manifest::write_root_manifest(
        repo,
        &RootManifest {
            name: Some(default_repo_name(repo)),
        },
    )
}

fn default_repo_name(repo: &Path) -> String {
    repo.file_name()
        .filter(|name| !name.is_empty())
        .map(|name| name.to_string_lossy().into_owned())
        .unwrap_or_else(|| "lnk".to_string())
}

fn create_dir_all(path: &Path) -> Result<(), Error> {
    fs::create_dir_all(path).map_err(|error| create_error(path, error))
}

fn create_error(path: &Path, error: io::Error) -> Error {
    Error::RepoCreateFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
