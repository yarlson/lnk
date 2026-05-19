#![allow(dead_code)]

use std::{collections::BTreeMap, fs, io, path::Path};

use serde::{Deserialize, Serialize};

use crate::errors::Error;

const CURRENT_FORMAT: u32 = 1;

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct RootManifest {
    pub(crate) name: Option<String>,
}

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub(crate) struct ProfileManifest {
    pub(crate) files: BTreeMap<String, FileEntry>,
    pub(crate) scripts: BTreeMap<String, ScriptEntry>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct FileEntry {
    pub(crate) target: Option<String>,
    pub(crate) mode: FileMode,
    pub(crate) overrides: Option<String>,
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum FileMode {
    #[default]
    Symlink,
    Copy,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct ScriptEntry {
    pub(crate) platforms: Option<Vec<String>>,
}

pub(crate) fn read_root_manifest(repo: &Path) -> Result<RootManifest, Error> {
    let path = repo.join("lnk.toml");
    let contents = read_schema(&path)?;

    parse_root_manifest(&contents, &path)
}

pub(crate) fn write_root_manifest(repo: &Path, manifest: &RootManifest) -> Result<(), Error> {
    let path = repo.join("lnk.toml");
    let contents = serialize_root_manifest(manifest, &path)?;

    fs::write(&path, contents).map_err(|error| Error::SchemaWriteFailed {
        path: path_to_string(&path),
        message: error.to_string(),
    })
}

pub(crate) fn parse_root_manifest(contents: &str, path: &Path) -> Result<RootManifest, Error> {
    let parsed: RootManifestToml = parse_toml(contents, path)?;
    require_current_format(parsed.format, path)?;

    Ok(RootManifest { name: parsed.name })
}

pub(crate) fn serialize_root_manifest(
    manifest: &RootManifest,
    path: &Path,
) -> Result<String, Error> {
    let file = RootManifestToml {
        format: CURRENT_FORMAT,
        name: manifest.name.clone(),
    };

    toml::to_string_pretty(&file).map_err(|error| Error::SchemaInvalid {
        path: path_to_string(path),
        message: error.to_string(),
    })
}

pub(crate) fn read_profile_manifest(profile_dir: &Path) -> Result<ProfileManifest, Error> {
    let path = profile_dir.join("profile.toml");

    match fs::read_to_string(&path) {
        Ok(contents) => parse_profile_manifest(&contents, &path),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(ProfileManifest::default()),
        Err(error) => Err(read_error(&path, error)),
    }
}

pub(crate) fn parse_profile_manifest(
    contents: &str,
    path: &Path,
) -> Result<ProfileManifest, Error> {
    let parsed: ProfileManifestToml = parse_toml(contents, path)?;

    Ok(ProfileManifest {
        files: parsed
            .files
            .into_iter()
            .map(|(source, entry)| {
                (
                    source,
                    FileEntry {
                        target: entry.target,
                        mode: entry.mode,
                        overrides: entry.overrides,
                    },
                )
            })
            .collect(),
        scripts: parsed
            .scripts
            .into_iter()
            .map(|(name, entry)| {
                (
                    name,
                    ScriptEntry {
                        platforms: entry.platforms,
                    },
                )
            })
            .collect(),
    })
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct RootManifestToml {
    format: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
struct ProfileManifestToml {
    #[serde(default)]
    files: BTreeMap<String, RawFileEntry>,
    #[serde(default)]
    scripts: BTreeMap<String, RawScriptEntry>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct RawFileEntry {
    target: Option<String>,
    #[serde(default)]
    mode: FileMode,
    overrides: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct RawScriptEntry {
    platforms: Option<Vec<String>>,
}

fn read_schema(path: &Path) -> Result<String, Error> {
    fs::read_to_string(path).map_err(|error| read_error(path, error))
}

fn read_error(path: &Path, error: io::Error) -> Error {
    Error::SchemaReadFailed {
        path: path_to_string(path),
        message: error.to_string(),
    }
}

fn parse_toml<T>(contents: &str, path: &Path) -> Result<T, Error>
where
    T: for<'de> Deserialize<'de>,
{
    toml::from_str(contents).map_err(|error| Error::SchemaInvalid {
        path: path_to_string(path),
        message: error.to_string(),
    })
}

fn require_current_format(format: u32, path: &Path) -> Result<(), Error> {
    if format == CURRENT_FORMAT {
        Ok(())
    } else {
        Err(Error::UnsupportedSchemaFormat {
            path: path_to_string(path),
            format,
        })
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}

#[cfg(test)]
mod tests {
    use std::{fs, path::Path};

    use super::{
        FileMode, RootManifest, parse_profile_manifest, parse_root_manifest, read_profile_manifest,
        serialize_root_manifest, write_root_manifest,
    };
    use crate::errors::Error;

    #[test]
    fn parses_valid_root_manifest() {
        let manifest = parse_root_manifest(
            include_str!("../tests/fixtures/schema/root-valid.toml"),
            path(),
        )
        .expect("valid root manifest should parse");

        assert_eq!(manifest.name.as_deref(), Some("machine-setup"));
    }

    #[test]
    fn serializes_root_manifest_with_required_format_and_optional_name() {
        let named = serialize_root_manifest(
            &RootManifest {
                name: Some("machine-setup".to_string()),
            },
            path(),
        )
        .expect("root manifest should serialize");
        assert_eq!("format = 1\nname = \"machine-setup\"\n", named);

        let unnamed = serialize_root_manifest(&RootManifest { name: None }, path())
            .expect("root manifest without name should serialize");
        assert_eq!("format = 1\n", unnamed);
    }

    #[test]
    fn write_root_manifest_writes_lnk_toml() {
        let temp = tempfile_dir("root-write");

        write_root_manifest(
            temp.path(),
            &RootManifest {
                name: Some("machine-setup".to_string()),
            },
        )
        .expect("root manifest should write");

        let contents =
            fs::read_to_string(temp.path().join("lnk.toml")).expect("root manifest should exist");
        assert_eq!("format = 1\nname = \"machine-setup\"\n", contents);
    }

    #[test]
    fn root_manifest_rejects_missing_malformed_unsupported_and_unknown_fields() {
        assert!(matches!(
            parse_root_manifest(
                include_str!("../tests/fixtures/schema/root-missing-format.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
        assert!(matches!(
            parse_root_manifest(
                include_str!("../tests/fixtures/schema/root-malformed.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
        assert_eq!(
            parse_root_manifest(
                include_str!("../tests/fixtures/schema/root-unsupported-format.toml"),
                path()
            ),
            Err(Error::UnsupportedSchemaFormat {
                path: "fixture.toml".to_string(),
                format: 2,
            })
        );
        assert!(matches!(
            parse_root_manifest(
                include_str!("../tests/fixtures/schema/root-unknown-field.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
    }

    #[test]
    fn parses_valid_profile_manifest() {
        let manifest = parse_profile_manifest(
            include_str!("../tests/fixtures/schema/profile-valid.toml"),
            path(),
        )
        .expect("valid profile manifest should parse");

        let gitconfig = manifest
            .files
            .get(".gitconfig")
            .expect("file entry should exist");
        assert_eq!(gitconfig.mode, FileMode::Copy);
        assert_eq!(gitconfig.target.as_deref(), Some(".config/git/config"));
        assert_eq!(gitconfig.overrides.as_deref(), Some("base"));

        let setup = manifest
            .scripts
            .get("setup")
            .expect("script entry should exist");
        assert_eq!(
            setup.platforms.as_ref().expect("platforms should exist"),
            &["macos".to_string(), "linux".to_string()]
        );
    }

    #[test]
    fn profile_manifest_is_optional() {
        let temp = tempfile_dir("profile-optional");

        assert_eq!(
            read_profile_manifest(temp.path()).expect("missing profile.toml should be empty"),
            super::ProfileManifest::default()
        );
    }

    #[test]
    fn profile_manifest_rejects_malformed_and_unknown_fields() {
        assert!(matches!(
            parse_profile_manifest(
                include_str!("../tests/fixtures/schema/profile-malformed.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
        assert!(matches!(
            parse_profile_manifest(
                include_str!("../tests/fixtures/schema/profile-top-level-unknown-field.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
        assert!(matches!(
            parse_profile_manifest(
                include_str!("../tests/fixtures/schema/profile-unknown-field.toml"),
                path()
            ),
            Err(Error::SchemaInvalid { .. })
        ));
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
