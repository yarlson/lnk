use std::{collections::BTreeMap, path::Path, process::Command as ProcessCommand};

use crate::{
    app::Output,
    cli::RunCommand,
    errors::Error,
    manifest,
    ownership::{self, ScriptMetadata},
    paths,
    profile::{self, Profile},
    state,
};

pub fn run(command: RunCommand) -> Result<Output, Error> {
    let repo = paths::repo_dir()?;
    let _root_manifest = manifest::read_root_manifest(&repo)?;
    let profiles = profile::discover(&repo)?;

    match command {
        RunCommand::Script { script } => run_across_active_profiles(&repo, &profiles, &script)?,
        RunCommand::ProfileScript { profile, script } => {
            run_profile_script(&repo, &profiles, &profile, &script)?
        }
    }

    Ok(Output::Empty)
}

fn run_across_active_profiles(
    repo: &Path,
    profiles: &BTreeMap<String, Profile>,
    script: &str,
) -> Result<(), Error> {
    let state = state::read_state_if_exists(&paths::state_file()?)?.unwrap_or_default();

    if state.active_profiles.is_empty() {
        return Err(Error::NoActiveProfiles);
    }

    for profile_name in state.active_profiles {
        let profile = profiles
            .get(&profile_name)
            .ok_or_else(|| Error::UnknownProfile {
                profile: profile_name.clone(),
            })?;
        let Some(metadata) = script_metadata(profile, script)? else {
            continue;
        };

        run_script_if_platform_matches(repo, &metadata)?;
    }

    Ok(())
}

fn run_profile_script(
    repo: &Path,
    profiles: &BTreeMap<String, Profile>,
    profile_name: &str,
    script: &str,
) -> Result<(), Error> {
    let profile = profiles
        .get(profile_name)
        .ok_or_else(|| Error::UnknownProfile {
            profile: profile_name.to_string(),
        })?;
    let metadata = script_metadata(profile, script)?.ok_or_else(|| Error::ScriptMissing {
        profile: profile_name.to_string(),
        script: script.to_string(),
    })?;

    run_script_if_platform_matches(repo, &metadata)
}

fn script_metadata(profile: &Profile, script: &str) -> Result<Option<ScriptMetadata>, Error> {
    Ok(ownership::script_metadata(profile)?
        .into_iter()
        .find(|metadata| metadata.script == script))
}

fn run_script_if_platform_matches(repo: &Path, metadata: &ScriptMetadata) -> Result<(), Error> {
    let current_platform = std::env::consts::OS;

    if !metadata.platforms.is_empty()
        && !metadata
            .platforms
            .iter()
            .any(|platform| platform == current_platform)
    {
        println!(
            "skip {}:{}: platform {}",
            metadata.profile, metadata.script, current_platform
        );
        return Ok(());
    }

    run_script(repo, metadata)
}

fn run_script(repo: &Path, metadata: &ScriptMetadata) -> Result<(), Error> {
    let script_path = repo
        .join("profiles")
        .join(&metadata.profile)
        .join("scripts")
        .join(&metadata.script);
    let status = ProcessCommand::new(&script_path)
        .status()
        .map_err(|error| Error::ScriptExecFailed {
            profile: metadata.profile.clone(),
            script: metadata.script.clone(),
            path: path_to_string(&script_path),
            message: error.to_string(),
        })?;

    if status.success() {
        Ok(())
    } else {
        Err(Error::ScriptFailed {
            profile: metadata.profile.clone(),
            script: metadata.script.clone(),
            code: status.code(),
        })
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}
