#![allow(dead_code)]

use std::{
    collections::{BTreeMap, BTreeSet},
    io,
    path::{Path, PathBuf},
};

use crate::{
    errors::Error,
    manifest::FileMode,
    paths,
    profile::{self, Profile},
};

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct OwnershipPlan {
    pub(crate) files: Vec<FileClaim>,
    pub(crate) scripts: Vec<ScriptMetadata>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct FileClaim {
    pub(crate) profile: String,
    pub(crate) source: String,
    pub(crate) target: String,
    pub(crate) mode: FileMode,
    pub(crate) overrides: Option<String>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub(crate) struct ScriptMetadata {
    pub(crate) profile: String,
    pub(crate) script: String,
    pub(crate) platforms: Vec<String>,
}

pub(crate) fn plan(repo: &Path, selected_profiles: &[String]) -> Result<OwnershipPlan, Error> {
    let profiles = profile::discover(repo)?;
    let selected = selected_profiles_in_order(&profiles, selected_profiles)?;
    let mut accepted_by_target: BTreeMap<String, FileClaim> = BTreeMap::new();
    let mut files: Vec<FileClaim> = Vec::new();
    let mut scripts = Vec::new();

    for profile in selected {
        scripts.extend(script_metadata(profile)?);

        for claim in file_claims(repo, profile)? {
            let replaces_existing = validate_override(&accepted_by_target, &claim)?;

            if replaces_existing {
                files.retain(|existing| existing.target != claim.target);
            }

            accepted_by_target.insert(claim.target.clone(), claim.clone());
            files.push(claim);
        }
    }

    Ok(OwnershipPlan { files, scripts })
}

fn selected_profiles_in_order<'a>(
    profiles: &'a BTreeMap<String, Profile>,
    selected_profiles: &[String],
) -> Result<Vec<&'a Profile>, Error> {
    let mut seen = BTreeSet::new();
    let mut selected = Vec::new();

    for name in selected_profiles {
        if !seen.insert(name.as_str()) {
            return Err(Error::DuplicateProfileSelection {
                profile: name.clone(),
            });
        }

        selected.push(profiles.get(name).ok_or_else(|| Error::UnknownProfile {
            profile: name.clone(),
        })?);
    }

    Ok(selected)
}

pub(crate) fn file_claims(repo: &Path, profile: &Profile) -> Result<Vec<FileClaim>, Error> {
    validate_manifest_file_sources(profile)?;

    let files_dir = profile.dir.join("files");
    let discovered = regular_files_under(&files_dir)?;
    let mut discovered_sources = BTreeSet::new();
    let mut claims = Vec::new();

    for source_relative in discovered {
        let source = path_to_string(&source_relative);
        discovered_sources.insert(source.clone());

        let manifest_entry = profile.manifest.files.get(&source);
        let target = manifest_entry
            .and_then(|entry| entry.target.as_deref())
            .unwrap_or(&source);
        let target = paths::normalize_target_path(Path::new(target))?;
        let mode = manifest_entry.map_or(FileMode::Symlink, |entry| entry.mode);
        let overrides = manifest_entry.and_then(|entry| entry.overrides.clone());
        let repo_source = repo_relative_source(repo, &profile.dir, &source_relative);

        claims.push(FileClaim {
            profile: profile.name.clone(),
            source: path_to_string(&repo_source),
            target: path_to_string(target.as_path()),
            mode,
            overrides,
        });
    }

    for source in profile.manifest.files.keys() {
        if !discovered_sources.contains(source) {
            return Err(Error::ManifestFileMissing {
                profile: profile.name.clone(),
                source: source.clone(),
            });
        }
    }

    Ok(claims)
}

fn validate_manifest_file_sources(profile: &Profile) -> Result<(), Error> {
    for source in profile.manifest.files.keys() {
        paths::normalize_target_path(Path::new(source))?;
    }

    Ok(())
}

fn regular_files_under(files_dir: &Path) -> Result<Vec<PathBuf>, Error> {
    let mut files = Vec::new();
    collect_regular_files(files_dir, files_dir, &mut files)?;
    files.sort();

    Ok(files)
}

fn collect_regular_files(root: &Path, dir: &Path, files: &mut Vec<PathBuf>) -> Result<(), Error> {
    let entries = match profile::sorted_dir_entries(dir) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(()),
        Err(error) => return Err(discovery_error(dir, error)),
    };

    for entry in entries {
        let path = entry.path();
        let file_type = entry
            .file_type()
            .map_err(|error| discovery_error(&path, error))?;

        if file_type.is_dir() {
            collect_regular_files(root, &path, files)?;
        } else if file_type.is_file() {
            files.push(
                path.strip_prefix(root)
                    .expect("discovered file should be under profile files root")
                    .to_path_buf(),
            );
        }
    }

    Ok(())
}

fn repo_relative_source(repo: &Path, profile_dir: &Path, source_relative: &Path) -> PathBuf {
    profile_dir
        .join("files")
        .join(source_relative)
        .strip_prefix(repo)
        .expect("profile source should be under repo")
        .to_path_buf()
}

fn validate_override(
    accepted_by_target: &BTreeMap<String, FileClaim>,
    claim: &FileClaim,
) -> Result<bool, Error> {
    match (
        accepted_by_target.get(&claim.target),
        claim.overrides.as_deref(),
    ) {
        (None, None) => Ok(false),
        (None, Some(overrides)) => Err(Error::InvalidOverride {
            profile: claim.profile.clone(),
            target: claim.target.clone(),
            overrides: overrides.to_string(),
        }),
        (Some(previous), Some(overrides))
            if previous.profile == overrides && previous.profile != claim.profile =>
        {
            Ok(true)
        }
        (Some(_), Some(overrides)) => Err(Error::InvalidOverride {
            profile: claim.profile.clone(),
            target: claim.target.clone(),
            overrides: overrides.to_string(),
        }),
        (Some(previous), None) => Err(Error::DuplicateTargetClaim {
            target: claim.target.clone(),
            first_profile: previous.profile.clone(),
            second_profile: claim.profile.clone(),
        }),
    }
}

pub(crate) fn script_metadata(profile: &Profile) -> Result<Vec<ScriptMetadata>, Error> {
    let scripts_dir = profile.dir.join("scripts");
    let discovered = regular_files_in_dir(&scripts_dir)?;
    let mut discovered_scripts = BTreeSet::new();
    let mut scripts = Vec::new();

    for script in discovered {
        let script = path_to_string(&script);
        discovered_scripts.insert(script.clone());

        let platforms = profile
            .manifest
            .scripts
            .get(&script)
            .and_then(|entry| entry.platforms.clone())
            .unwrap_or_default();

        validate_platforms(profile, &script, &platforms)?;

        scripts.push(ScriptMetadata {
            profile: profile.name.clone(),
            script,
            platforms,
        });
    }

    for script in profile.manifest.scripts.keys() {
        if !discovered_scripts.contains(script) {
            return Err(Error::ManifestScriptMissing {
                profile: profile.name.clone(),
                script: script.clone(),
            });
        }
    }

    Ok(scripts)
}

fn regular_files_in_dir(dir: &Path) -> Result<Vec<PathBuf>, Error> {
    let entries = match profile::sorted_dir_entries(dir) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(error) => return Err(discovery_error(dir, error)),
    };
    let mut files = Vec::new();

    for entry in entries {
        let path = entry.path();
        let file_type = entry
            .file_type()
            .map_err(|error| discovery_error(&path, error))?;

        if file_type.is_file() {
            files.push(PathBuf::from(entry.file_name()));
        }
    }

    Ok(files)
}

fn validate_platforms(profile: &Profile, script: &str, platforms: &[String]) -> Result<(), Error> {
    for platform in platforms {
        if !is_supported_platform(platform) {
            return Err(Error::InvalidScriptPlatform {
                profile: profile.name.clone(),
                script: script.to_string(),
                platform: platform.clone(),
            });
        }
    }

    Ok(())
}

fn is_supported_platform(platform: &str) -> bool {
    matches!(platform, "macos" | "linux" | "windows")
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

#[cfg(test)]
mod tests {
    use std::{
        fs,
        path::{Path, PathBuf},
        time::{SystemTime, UNIX_EPOCH},
    };

    use super::plan;
    use crate::{errors::Error, manifest::FileMode};

    #[test]
    fn keeps_selected_profile_order() {
        let repo = TempRepo::new("profile-order");
        repo.file("profiles/base/files/.zshrc");
        repo.file("profiles/work/files/.gitconfig");

        let plan =
            plan(repo.path(), &profiles(["work", "base"])).expect("ownership plan should build");

        assert_eq!(plan.files[0].profile, "work");
        assert_eq!(plan.files[0].target, ".gitconfig");
        assert_eq!(plan.files[1].profile, "base");
        assert_eq!(plan.files[1].target, ".zshrc");
    }

    #[test]
    fn applies_target_mapping_and_copy_mode_from_manifest() {
        let repo = TempRepo::new("target-mapping");
        repo.file("profiles/work/files/.gitconfig");
        repo.write(
            "profiles/work/profile.toml",
            r#"
[files.".gitconfig"]
target = ".config/git/config"
mode = "copy"
"#,
        );

        let plan = plan(repo.path(), &profiles(["work"])).expect("ownership plan should build");

        assert_eq!(plan.files.len(), 1);
        assert_eq!(plan.files[0].source, "profiles/work/files/.gitconfig");
        assert_eq!(plan.files[0].target, ".config/git/config");
        assert_eq!(plan.files[0].mode, FileMode::Copy);
    }

    #[test]
    fn rejects_missing_and_duplicate_selected_profiles() {
        let repo = TempRepo::new("selected-profiles");
        repo.file("profiles/base/files/.zshrc");

        assert_eq!(
            plan(repo.path(), &profiles(["base", "base"])),
            Err(Error::DuplicateProfileSelection {
                profile: "base".to_string(),
            })
        );
        assert_eq!(
            plan(repo.path(), &profiles(["work"])),
            Err(Error::UnknownProfile {
                profile: "work".to_string(),
            })
        );
    }

    #[test]
    fn rejects_duplicate_targets_without_explicit_later_override() {
        let repo = TempRepo::new("duplicate-target");
        repo.file("profiles/base/files/.gitconfig");
        repo.file("profiles/work/files/.gitconfig");

        assert_eq!(
            plan(repo.path(), &profiles(["base", "work"])),
            Err(Error::DuplicateTargetClaim {
                target: ".gitconfig".to_string(),
                first_profile: "base".to_string(),
                second_profile: "work".to_string(),
            })
        );
    }

    #[test]
    fn accepts_duplicate_target_when_later_profile_overrides_current_owner() {
        let repo = TempRepo::new("explicit-override");
        repo.file("profiles/base/files/.gitconfig");
        repo.file("profiles/work/files/.gitconfig");
        repo.write(
            "profiles/work/profile.toml",
            r#"
[files.".gitconfig"]
overrides = "base"
"#,
        );

        let plan =
            plan(repo.path(), &profiles(["base", "work"])).expect("override should be valid");

        assert_eq!(plan.files.len(), 1);
        assert_eq!(plan.files[0].profile, "work");
        assert_eq!(plan.files[0].target, ".gitconfig");
        assert_eq!(plan.files[0].overrides.as_deref(), Some("base"));
    }

    #[test]
    fn rejects_override_without_matching_earlier_target_owner() {
        let repo = TempRepo::new("invalid-override");
        repo.file("profiles/base/files/.gitconfig");
        repo.file("profiles/work/files/.gitconfig");
        repo.write(
            "profiles/work/profile.toml",
            r#"
[files.".gitconfig"]
overrides = "mac"
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base", "work"])),
            Err(Error::InvalidOverride {
                profile: "work".to_string(),
                target: ".gitconfig".to_string(),
                overrides: "mac".to_string(),
            })
        );
    }

    #[test]
    fn rejects_same_profile_override() {
        let repo = TempRepo::new("same-profile-override");
        repo.file("profiles/base/files/.gitconfig");
        repo.file("profiles/base/files/.config/git/config");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[files.".config/git/config"]
target = ".gitconfig"
overrides = "base"
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base"])),
            Err(Error::InvalidOverride {
                profile: "base".to_string(),
                target: ".gitconfig".to_string(),
                overrides: "base".to_string(),
            })
        );
    }

    #[test]
    fn validates_manifest_file_sources_and_target_paths() {
        let repo = TempRepo::new("manifest-sources");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[files.".missing"]
mode = "copy"
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base"])),
            Err(Error::ManifestFileMissing {
                profile: "base".to_string(),
                source: ".missing".to_string(),
            })
        );

        let repo = TempRepo::new("manifest-targets");
        repo.file("profiles/base/files/.zshrc");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[files.".zshrc"]
target = "../.zshrc"
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base"])),
            Err(Error::InvalidTargetPath {
                path: "../.zshrc".to_string(),
                reason: "parent-escaping",
            })
        );
    }

    #[test]
    fn validates_script_platform_names() {
        let repo = TempRepo::new("script-platforms");
        repo.file("profiles/base/scripts/setup");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[scripts.setup]
platforms = ["macos", "plan9"]
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base"])),
            Err(Error::InvalidScriptPlatform {
                profile: "base".to_string(),
                script: "setup".to_string(),
                platform: "plan9".to_string(),
            })
        );
    }

    #[test]
    fn accepts_supported_script_platform_names() {
        let repo = TempRepo::new("supported-script-platforms");
        repo.file("profiles/base/scripts/setup");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[scripts.setup]
platforms = ["macos", "linux", "windows"]
"#,
        );

        let plan = plan(repo.path(), &profiles(["base"])).expect("platforms should be valid");

        assert_eq!(plan.scripts.len(), 1);
        assert_eq!(plan.scripts[0].platforms, ["macos", "linux", "windows"]);
    }

    #[test]
    fn discovers_scripts_by_convention_and_rejects_metadata_for_missing_scripts() {
        let repo = TempRepo::new("script-discovery");
        repo.file("profiles/base/scripts/setup");

        let ownership =
            plan(repo.path(), &profiles(["base"])).expect("script should be discovered");

        assert_eq!(ownership.scripts.len(), 1);
        assert_eq!(ownership.scripts[0].script, "setup");
        assert!(ownership.scripts[0].platforms.is_empty());

        let repo = TempRepo::new("missing-script-metadata");
        repo.write(
            "profiles/base/profile.toml",
            r#"
[scripts.setup]
platforms = ["macos"]
"#,
        );

        assert_eq!(
            plan(repo.path(), &profiles(["base"])),
            Err(Error::ManifestScriptMissing {
                profile: "base".to_string(),
                script: "setup".to_string(),
            })
        );
    }

    fn profiles<const N: usize>(names: [&str; N]) -> Vec<String> {
        names.into_iter().map(str::to_string).collect()
    }

    struct TempRepo {
        path: PathBuf,
    }

    impl TempRepo {
        fn new(name: &str) -> Self {
            let nonce = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .expect("system clock should be after unix epoch")
                .as_nanos();
            let path =
                std::env::temp_dir().join(format!("lnk-{name}-{}-{nonce}", std::process::id()));

            fs::create_dir_all(&path).expect("temp repo should be created");

            Self { path }
        }

        fn path(&self) -> &Path {
            &self.path
        }

        fn file(&self, relative: &str) {
            self.write(relative, "");
        }

        fn write(&self, relative: &str, contents: &str) {
            let path = self.path.join(relative);

            if let Some(parent) = path.parent() {
                fs::create_dir_all(parent).expect("fixture parent should be created");
            }

            fs::write(path, contents).expect("fixture file should be written");
        }
    }

    impl Drop for TempRepo {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.path);
        }
    }
}
