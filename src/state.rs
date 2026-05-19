#![allow(dead_code)]

use std::{collections::BTreeSet, fs, io, path::Path};

use serde::{Deserialize, Serialize};

use crate::{errors::Error, manifest::FileMode};

const CURRENT_FORMAT: u32 = 1;

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub(crate) struct MachineState {
    pub(crate) active_profiles: Vec<String>,
    pub(crate) targets: Vec<TargetState>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct TargetState {
    pub(crate) target: String,
    pub(crate) profile: String,
    pub(crate) source: String,
    pub(crate) mode: FileMode,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) source_hash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) applied_hash: Option<String>,
}

pub(crate) fn read_state(path: &Path) -> Result<MachineState, Error> {
    let contents = fs::read_to_string(path).map_err(|error| read_error(path, error))?;

    parse_state(&contents, path)
}

pub(crate) fn read_state_if_exists(path: &Path) -> Result<Option<MachineState>, Error> {
    match fs::read_to_string(path) {
        Ok(contents) => parse_state(&contents, path).map(Some),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
        Err(error) => Err(read_error(path, error)),
    }
}

pub(crate) fn write_state(path: &Path, state: &MachineState) -> Result<(), Error> {
    let contents = serialize_state(state, path)?;

    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|error| write_error(path, error))?;
    }

    fs::write(path, contents).map_err(|error| write_error(path, error))
}

pub(crate) fn parse_state(contents: &str, path: &Path) -> Result<MachineState, Error> {
    let parsed: MachineStateToml =
        toml::from_str(contents).map_err(|error| Error::SchemaInvalid {
            path: path_to_string(path),
            message: error.to_string(),
        })?;

    if parsed.format != CURRENT_FORMAT {
        return Err(Error::UnsupportedSchemaFormat {
            path: path_to_string(path),
            format: parsed.format,
        });
    }

    reject_duplicate_targets(&parsed.targets)?;

    Ok(MachineState {
        active_profiles: parsed.active_profiles,
        targets: parsed.targets,
    })
}

pub(crate) fn serialize_state(state: &MachineState, path: &Path) -> Result<String, Error> {
    let mut targets = state.targets.clone();
    targets.sort_by(|left, right| left.target.cmp(&right.target));

    let file = MachineStateToml {
        format: CURRENT_FORMAT,
        active_profiles: state.active_profiles.clone(),
        targets,
    };

    toml::to_string_pretty(&file).map_err(|error| Error::SchemaInvalid {
        path: path_to_string(path),
        message: error.to_string(),
    })
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct MachineStateToml {
    format: u32,
    active_profiles: Vec<String>,
    targets: Vec<TargetState>,
}

fn reject_duplicate_targets(targets: &[TargetState]) -> Result<(), Error> {
    let mut seen = BTreeSet::new();

    for target in targets {
        if !seen.insert(&target.target) {
            return Err(Error::DuplicateStateTarget {
                target: target.target.clone(),
            });
        }
    }

    Ok(())
}

fn read_error(path: &Path, error: io::Error) -> Error {
    Error::SchemaReadFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}

fn write_error(path: &Path, error: io::Error) -> Error {
    Error::SchemaWriteFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}

#[cfg(test)]
mod tests {
    use std::{fs, path::Path};

    use super::{MachineState, TargetState, parse_state, serialize_state, write_state};
    use crate::{errors::Error, manifest::FileMode};

    #[test]
    fn parses_valid_state() {
        let state = parse_state(
            include_str!("../tests/fixtures/schema/state-valid.toml"),
            path(),
        )
        .expect("valid state should parse");

        assert_eq!(state.active_profiles, ["base", "mac", "work"]);
        assert_eq!(state.targets.len(), 2);
        assert_eq!(state.targets[0].mode, FileMode::Symlink);
        assert_eq!(state.targets[1].mode, FileMode::Copy);
        assert_eq!(
            state.targets[1].source_hash.as_deref(),
            Some("sha256:source")
        );
    }

    #[test]
    fn state_rejects_missing_malformed_unsupported_unknown_and_duplicate_targets() {
        for fixture in [
            "state-missing-format.toml",
            "state-malformed.toml",
            "state-unknown-field.toml",
        ] {
            let contents = fixture_contents(fixture);
            assert!(
                matches!(
                    parse_state(contents, path()),
                    Err(Error::SchemaInvalid { .. })
                ),
                "{fixture} should be a schema error"
            );
        }

        assert_eq!(
            parse_state(
                include_str!("../tests/fixtures/schema/state-unsupported-format.toml"),
                path()
            ),
            Err(Error::UnsupportedSchemaFormat {
                path: "fixture.toml".to_string(),
                format: 2,
            })
        );
        assert_eq!(
            parse_state(
                include_str!("../tests/fixtures/schema/state-duplicate-target.toml"),
                path()
            ),
            Err(Error::DuplicateStateTarget {
                target: ".zshrc".to_string(),
            })
        );
    }

    #[test]
    fn serializes_targets_in_deterministic_target_order() {
        let state = MachineState {
            active_profiles: vec!["work".to_string(), "base".to_string()],
            targets: vec![copy_target(".gitconfig"), symlink_target(".zshrc")],
        };

        let contents = serialize_state(&state, path()).expect("state should serialize");
        let gitconfig = contents
            .find("target = \".gitconfig\"")
            .expect("gitconfig target should be present");
        let zshrc = contents
            .find("target = \".zshrc\"")
            .expect("zshrc target should be present");

        assert!(
            gitconfig < zshrc,
            "targets should be sorted by target path:\n{contents}"
        );
    }

    #[test]
    fn write_state_creates_parent_directories() {
        let temp = tempfile_dir("state-write");
        let path = temp.path().join("state/lnk/machine.toml");

        write_state(
            &path,
            &MachineState {
                active_profiles: vec!["base".to_string()],
                targets: vec![symlink_target(".zshrc")],
            },
        )
        .expect("state should write");

        let contents = fs::read_to_string(path).expect("state file should exist");
        assert!(contents.contains("format = 1"));
        assert!(contents.contains("active_profiles = [\"base\"]"));
    }

    fn fixture_contents(name: &str) -> &'static str {
        match name {
            "state-missing-format.toml" => {
                include_str!("../tests/fixtures/schema/state-missing-format.toml")
            }
            "state-malformed.toml" => {
                include_str!("../tests/fixtures/schema/state-malformed.toml")
            }
            "state-unknown-field.toml" => {
                include_str!("../tests/fixtures/schema/state-unknown-field.toml")
            }
            _ => unreachable!("unknown fixture"),
        }
    }

    fn symlink_target(target: &str) -> TargetState {
        TargetState {
            target: target.to_string(),
            profile: "base".to_string(),
            source: format!("profiles/base/files/{target}"),
            mode: FileMode::Symlink,
            source_hash: None,
            applied_hash: None,
        }
    }

    fn copy_target(target: &str) -> TargetState {
        TargetState {
            target: target.to_string(),
            profile: "work".to_string(),
            source: format!("profiles/work/files/{target}"),
            mode: FileMode::Copy,
            source_hash: Some("sha256:source".to_string()),
            applied_hash: Some("sha256:applied".to_string()),
        }
    }

    fn path() -> &'static Path {
        Path::new("fixture.toml")
    }

    struct TempDir {
        path: std::path::PathBuf,
    }

    impl TempDir {
        fn path(&self) -> &Path {
            &self.path
        }
    }

    impl Drop for TempDir {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.path);
        }
    }

    fn tempfile_dir(name: &str) -> TempDir {
        let path = std::env::temp_dir().join(format!(
            "lnk-{name}-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .expect("clock should be after unix epoch")
                .as_nanos()
        ));
        fs::create_dir_all(&path).expect("temp fixture should be created");

        TempDir { path }
    }
}
