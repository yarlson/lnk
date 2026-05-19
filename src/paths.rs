#![allow(dead_code)]

use std::{
    env,
    ffi::OsStr,
    path::{Component, Path, PathBuf},
};

use crate::errors::Error;

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct HomeRelativePath(PathBuf);

impl HomeRelativePath {
    pub(crate) fn as_path(&self) -> &Path {
        &self.0
    }
}

pub(crate) fn home_dir() -> Result<PathBuf, Error> {
    let home = env::var_os("HOME");
    home_dir_from(home.as_deref())
}

pub(crate) fn repo_dir() -> Result<PathBuf, Error> {
    let lnk_home = env::var_os("LNK_HOME");
    let xdg_config_home = env::var_os("XDG_CONFIG_HOME");
    let home = env::var_os("HOME");

    repo_dir_from_env(
        home.as_deref(),
        lnk_home.as_deref(),
        xdg_config_home.as_deref(),
    )
}

pub(crate) fn state_file() -> Result<PathBuf, Error> {
    let xdg_state_home = env::var_os("XDG_STATE_HOME");
    let home = env::var_os("HOME");

    state_file_from_env(home.as_deref(), xdg_state_home.as_deref())
}

pub(crate) fn home_dir_from(home: Option<&OsStr>) -> Result<PathBuf, Error> {
    let Some(home) = non_empty_env_value(home) else {
        return Err(Error::MissingHome);
    };

    let path = PathBuf::from(home);
    if !path.is_absolute() {
        return Err(Error::InvalidHome {
            path: path_to_string(&path),
        });
    }

    Ok(path)
}

pub(crate) fn repo_dir_from(
    home: &Path,
    lnk_home: Option<&OsStr>,
    xdg_config_home: Option<&OsStr>,
) -> PathBuf {
    if let Some(lnk_home) = non_empty_env_value(lnk_home) {
        return PathBuf::from(lnk_home);
    }

    xdg_dir_from(xdg_config_home)
        .unwrap_or_else(|| home.join(".config"))
        .join("lnk")
}

pub(crate) fn repo_dir_from_env(
    home: Option<&OsStr>,
    lnk_home: Option<&OsStr>,
    xdg_config_home: Option<&OsStr>,
) -> Result<PathBuf, Error> {
    if let Some(lnk_home) = non_empty_env_value(lnk_home) {
        return Ok(PathBuf::from(lnk_home));
    }

    if let Some(xdg_config_home) = xdg_dir_from(xdg_config_home) {
        return Ok(xdg_config_home.join("lnk"));
    }

    Ok(home_dir_from(home)?.join(".config").join("lnk"))
}

pub(crate) fn state_file_from(home: &Path, xdg_state_home: Option<&OsStr>) -> PathBuf {
    xdg_dir_from(xdg_state_home)
        .unwrap_or_else(|| home.join(".local").join("state"))
        .join("lnk")
        .join("machine.toml")
}

pub(crate) fn state_file_from_env(
    home: Option<&OsStr>,
    xdg_state_home: Option<&OsStr>,
) -> Result<PathBuf, Error> {
    if let Some(xdg_state_home) = xdg_dir_from(xdg_state_home) {
        return Ok(xdg_state_home.join("lnk").join("machine.toml"));
    }

    Ok(home_dir_from(home)?
        .join(".local")
        .join("state")
        .join("lnk")
        .join("machine.toml"))
}

pub(crate) fn xdg_dir_from(value: Option<&OsStr>) -> Option<PathBuf> {
    let value = non_empty_env_value(value)?;
    let path = PathBuf::from(value);

    path.is_absolute().then_some(path)
}

pub(crate) fn normalize_target_path(path: &Path) -> Result<HomeRelativePath, Error> {
    let original = path_to_string(path);

    if path.as_os_str().is_empty() {
        return Err(invalid_target(original, "empty"));
    }

    if path.is_absolute() {
        return Err(invalid_target(original, "absolute"));
    }

    let mut normalized = PathBuf::new();
    let mut saw_component = false;

    for component in path.components() {
        match component {
            Component::Normal(part) => {
                normalized.push(part);
                saw_component = true;
            }
            Component::ParentDir => return Err(invalid_target(original, "parent-escaping")),
            Component::CurDir => return Err(invalid_target(original, "not-normalized")),
            Component::RootDir | Component::Prefix(_) => {
                return Err(invalid_target(original, "absolute"));
            }
        }
    }

    if !saw_component {
        return Err(invalid_target(original, "empty"));
    }

    if normalized.as_os_str() != path.as_os_str() {
        return Err(invalid_target(original, "not-normalized"));
    }

    Ok(HomeRelativePath(normalized))
}

pub(crate) fn target_path(home: &Path, target: &HomeRelativePath) -> Result<PathBuf, Error> {
    let path = home.join(target.as_path());
    ensure_inside_home(home, &path)?;
    Ok(path)
}

pub(crate) fn ensure_inside_home(home: &Path, path: &Path) -> Result<(), Error> {
    if !home.is_absolute() {
        return Err(Error::InvalidHome {
            path: path_to_string(home),
        });
    }

    if path.is_absolute() && path.starts_with(home) && path != home {
        return Ok(());
    }

    Err(Error::InvalidTargetPath {
        path: path_to_string(path),
        reason: "outside-home",
    })
}

fn non_empty_env_value(value: Option<&OsStr>) -> Option<&OsStr> {
    value.filter(|value| !value.is_empty())
}

fn invalid_target(path: String, reason: &'static str) -> Error {
    Error::InvalidTargetPath { path, reason }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}

#[cfg(test)]
mod tests {
    use std::{ffi::OsStr, path::Path};

    use super::{
        ensure_inside_home, home_dir_from, normalize_target_path, repo_dir_from, repo_dir_from_env,
        state_file_from, state_file_from_env, target_path, xdg_dir_from,
    };
    use crate::errors::Error;

    fn os(value: &str) -> &OsStr {
        OsStr::new(value)
    }

    #[test]
    fn home_dir_requires_present_absolute_home() {
        assert_eq!(home_dir_from(None), Err(Error::MissingHome));
        assert_eq!(home_dir_from(Some(os(""))), Err(Error::MissingHome));
        assert_eq!(
            home_dir_from(Some(os("relative-home"))),
            Err(Error::InvalidHome {
                path: "relative-home".to_string()
            })
        );
        assert_eq!(
            home_dir_from(Some(os("/tmp/home"))),
            Ok(Path::new("/tmp/home").to_path_buf())
        );
    }

    #[test]
    fn repo_dir_uses_lnk_home_before_xdg_and_default() {
        assert_eq!(
            repo_dir_from(
                Path::new("/tmp/home"),
                Some(os("/tmp/lnk")),
                Some(os("/tmp/config"))
            ),
            Path::new("/tmp/lnk")
        );
        assert_eq!(
            repo_dir_from(Path::new("/tmp/home"), None, Some(os("/tmp/config"))),
            Path::new("/tmp/config/lnk")
        );
        assert_eq!(
            repo_dir_from(Path::new("/tmp/home"), None, None),
            Path::new("/tmp/home/.config/lnk")
        );
    }

    #[test]
    fn repo_dir_does_not_require_home_until_default_is_needed() {
        assert_eq!(
            repo_dir_from_env(None, Some(os("/tmp/lnk")), Some(os("/tmp/config"))),
            Ok(Path::new("/tmp/lnk").to_path_buf())
        );
        assert_eq!(
            repo_dir_from_env(None, None, Some(os("/tmp/config"))),
            Ok(Path::new("/tmp/config/lnk").to_path_buf())
        );
        assert_eq!(repo_dir_from_env(None, None, None), Err(Error::MissingHome));
        assert_eq!(
            repo_dir_from_env(None, None, Some(os("relative-config"))),
            Err(Error::MissingHome)
        );
    }

    #[test]
    fn state_file_uses_xdg_state_then_default() {
        assert_eq!(
            state_file_from(Path::new("/tmp/home"), Some(os("/tmp/state"))),
            Path::new("/tmp/state/lnk/machine.toml")
        );
        assert_eq!(
            state_file_from(Path::new("/tmp/home"), None),
            Path::new("/tmp/home/.local/state/lnk/machine.toml")
        );
    }

    #[test]
    fn state_file_does_not_require_home_until_default_is_needed() {
        assert_eq!(
            state_file_from_env(None, Some(os("/tmp/state"))),
            Ok(Path::new("/tmp/state/lnk/machine.toml").to_path_buf())
        );
        assert_eq!(state_file_from_env(None, None), Err(Error::MissingHome));
        assert_eq!(
            state_file_from_env(None, Some(os("relative-state"))),
            Err(Error::MissingHome)
        );
    }

    #[test]
    fn xdg_dir_ignores_empty_and_relative_values() {
        assert_eq!(xdg_dir_from(None), None);
        assert_eq!(xdg_dir_from(Some(os(""))), None);
        assert_eq!(xdg_dir_from(Some(os("relative"))), None);
        assert_eq!(
            xdg_dir_from(Some(os("/tmp/config"))),
            Some(Path::new("/tmp/config").to_path_buf())
        );
    }

    #[test]
    fn normalized_targets_accept_home_relative_paths() {
        let target = normalize_target_path(Path::new(".config/nvim/init.lua"))
            .expect("target should normalize");

        assert_eq!(target.as_path(), Path::new(".config/nvim/init.lua"));
        assert_eq!(
            target_path(Path::new("/tmp/home"), &target),
            Ok(Path::new("/tmp/home/.config/nvim/init.lua").to_path_buf())
        );
    }

    #[test]
    fn normalized_targets_reject_unsafe_or_non_normal_paths() {
        for (path, reason) in [
            ("", "empty"),
            ("/tmp/file", "absolute"),
            ("..", "parent-escaping"),
            ("../file", "parent-escaping"),
            ("file/../other", "parent-escaping"),
            (".", "not-normalized"),
            ("file/./other", "not-normalized"),
            ("file//other", "not-normalized"),
            ("file/", "not-normalized"),
        ] {
            assert_eq!(
                normalize_target_path(Path::new(path)),
                Err(Error::InvalidTargetPath {
                    path: path.to_string(),
                    reason,
                }),
                "{path:?} should be rejected as {reason}"
            );
        }
    }

    #[test]
    fn home_boundary_rejects_paths_outside_home() {
        assert_eq!(
            ensure_inside_home(Path::new("/tmp/home"), Path::new("/tmp/home/.zshrc")),
            Ok(())
        );
        assert_eq!(
            ensure_inside_home(Path::new("/tmp/home"), Path::new("/tmp/home2/.zshrc")),
            Err(Error::InvalidTargetPath {
                path: "/tmp/home2/.zshrc".to_string(),
                reason: "outside-home",
            })
        );
        assert_eq!(
            ensure_inside_home(Path::new("/tmp/home"), Path::new("/tmp/home")),
            Err(Error::InvalidTargetPath {
                path: "/tmp/home".to_string(),
                reason: "outside-home",
            })
        );
    }
}
