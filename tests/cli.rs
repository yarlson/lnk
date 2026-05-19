use assert_cmd::Command as AssertCommand;
use sha2::{Digest, Sha256};
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::{
    fs,
    path::{Path, PathBuf},
    process::Command as ProcessCommand,
    time::{SystemTime, UNIX_EPOCH},
};

fn lnk() -> AssertCommand {
    AssertCommand::cargo_bin("lnk").expect("lnk binary should build")
}

fn lnk_with_git_identity() -> AssertCommand {
    let mut command = lnk();
    command
        .env("GIT_AUTHOR_NAME", "lnk test")
        .env("GIT_AUTHOR_EMAIL", "lnk@example.invalid")
        .env("GIT_COMMITTER_NAME", "lnk test")
        .env("GIT_COMMITTER_EMAIL", "lnk@example.invalid");
    command
}

fn git_init(repo: &Path) {
    let output = ProcessCommand::new("git")
        .arg("init")
        .arg(repo)
        .output()
        .expect("git init should run");

    assert!(
        output.status.success(),
        "git init failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );
}

fn git_in<const N: usize>(repo: &Path, args: [&str; N]) -> String {
    let output = ProcessCommand::new("git")
        .arg("-C")
        .arg(repo)
        .args(args)
        .env("GIT_AUTHOR_NAME", "lnk test")
        .env("GIT_AUTHOR_EMAIL", "lnk@example.invalid")
        .env("GIT_COMMITTER_NAME", "lnk test")
        .env("GIT_COMMITTER_EMAIL", "lnk@example.invalid")
        .output()
        .expect("git command should run");

    assert!(
        output.status.success(),
        "git command failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );

    String::from_utf8_lossy(&output.stdout).into_owned()
}

fn commit_all(repo: &Path, message: &str) {
    git_in(repo, ["add", "."]);
    git_in(repo, ["commit", "-m", message]);
}

fn write_root_manifest(repo: &Path) {
    fs::create_dir_all(repo).expect("repo dir should be created");
    fs::write(repo.join("lnk.toml"), "format = 1\nname = \"test-lnk\"\n")
        .expect("root manifest should write");
}

fn create_profile(repo: &Path, profile: &str) {
    fs::create_dir_all(repo.join("profiles").join(profile).join("files"))
        .expect("profile files dir should be created");
}

fn write_profile_file(repo: &Path, profile: &str, relative: &str, contents: &str) {
    let path = repo
        .join("profiles")
        .join(profile)
        .join("files")
        .join(relative);

    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).expect("profile file parent should be created");
    }

    fs::write(path, contents).expect("profile file should write");
}

fn write_profile_manifest(repo: &Path, profile: &str, contents: &str) {
    fs::write(
        repo.join("profiles").join(profile).join("profile.toml"),
        contents,
    )
    .expect("profile manifest should write");
}

fn write_profile_script(repo: &Path, profile: &str, script: &str, contents: &str) {
    let path = repo
        .join("profiles")
        .join(profile)
        .join("scripts")
        .join(script);

    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).expect("script parent should be created");
    }

    fs::write(&path, contents).expect("profile script should write");

    #[cfg(unix)]
    {
        let mut permissions = fs::metadata(&path)
            .expect("profile script metadata should read")
            .permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&path, permissions).expect("profile script should be executable");
    }
}

#[cfg(unix)]
fn create_symlink(source: &Path, target: &Path) {
    std::os::unix::fs::symlink(source, target).expect("symlink should be created");
}

#[cfg(windows)]
fn create_symlink(source: &Path, target: &Path) {
    std::os::windows::fs::symlink_file(source, target).expect("symlink should be created");
}

fn write_state(state_home: &Path, active_profiles: &[&str]) {
    let profiles = active_profiles
        .iter()
        .map(|profile| format!("\"{profile}\""))
        .collect::<Vec<_>>()
        .join(", ");
    let path = state_home.join("lnk").join("machine.toml");

    fs::create_dir_all(path.parent().expect("state path should have parent"))
        .expect("state dir should be created");
    fs::write(
        path,
        format!("format = 1\nactive_profiles = [{profiles}]\ntargets = []\n"),
    )
    .expect("state file should write");
}

fn state_contents(state_home: &Path) -> String {
    fs::read_to_string(state_home.join("lnk").join("machine.toml"))
        .expect("state file should exist")
}

fn write_state_with_copy_target(
    state_home: &Path,
    active_profiles: &[&str],
    target: &str,
    profile: &str,
    source: &str,
    source_hash: &str,
    applied_hash: &str,
) {
    let profiles = active_profiles
        .iter()
        .map(|profile| format!("\"{profile}\""))
        .collect::<Vec<_>>()
        .join(", ");
    let path = state_home.join("lnk").join("machine.toml");

    fs::create_dir_all(path.parent().expect("state path should have parent"))
        .expect("state dir should be created");
    fs::write(
        path,
        format!(
            "format = 1\nactive_profiles = [{profiles}]\n\n[[targets]]\ntarget = \"{target}\"\nprofile = \"{profile}\"\nsource = \"{source}\"\nmode = \"copy\"\nsource_hash = \"{source_hash}\"\napplied_hash = \"{applied_hash}\"\n"
        ),
    )
    .expect("state file should write");
}

fn write_state_with_symlink_target(
    state_home: &Path,
    active_profiles: &[&str],
    target: &str,
    profile: &str,
    source: &str,
) {
    let profiles = active_profiles
        .iter()
        .map(|profile| format!("\"{profile}\""))
        .collect::<Vec<_>>()
        .join(", ");
    let path = state_home.join("lnk").join("machine.toml");

    fs::create_dir_all(path.parent().expect("state path should have parent"))
        .expect("state dir should be created");
    fs::write(
        path,
        format!(
            "format = 1\nactive_profiles = [{profiles}]\n\n[[targets]]\ntarget = \"{target}\"\nprofile = \"{profile}\"\nsource = \"{source}\"\nmode = \"symlink\"\n"
        ),
    )
    .expect("state file should write");
}

fn sha256(contents: &str) -> String {
    let digest = Sha256::digest(contents.as_bytes());
    let hex = digest
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect::<String>();

    format!("sha256:{hex}")
}

fn backup_paths(home: &Path, target: &str) -> Vec<PathBuf> {
    let prefix = format!("{target}.lnk-backup-");
    let mut paths = fs::read_dir(home)
        .expect("home should be readable")
        .map(|entry| entry.expect("backup entry should be readable").path())
        .filter(|path| {
            path.file_name()
                .expect("backup path should have file name")
                .to_string_lossy()
                .starts_with(&prefix)
        })
        .collect::<Vec<_>>();

    paths.sort();
    paths
}

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(name: &str) -> Self {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("system clock should be after unix epoch")
            .as_nanos();
        let path = std::env::temp_dir().join(format!("lnk-{name}-{}-{nonce}", std::process::id()));

        fs::create_dir_all(&path).expect("temp dir should be created");

        Self { path }
    }

    fn path(&self) -> &Path {
        &self.path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

#[test]
fn help_lists_phase_2_commands() {
    let assert = lnk().arg("--help").assert().success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    for command in [
        "init", "profiles", "apply", "status", "run", "doctor", "migrate",
    ] {
        assert!(
            stdout.contains(command),
            "help output should list {command:?}: {stdout}"
        );
    }
}

#[test]
fn invalid_command_fails_before_dispatch() {
    let assert = lnk().arg("unknown").assert().failure();
    let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

    assert!(stderr.contains("unknown"), "{stderr}");
}

#[test]
fn unsupported_flags_are_rejected() {
    let assert = lnk().args(["apply", "--save"]).assert().failure();
    let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

    assert!(stderr.contains("--save"), "{stderr}");
}

#[test]
fn phase_2_commands_reach_domain_boundary() {
    let cases: &[&[&str]] = &[&["migrate", "v1"], &["migrate", "v1", "--write", "--yes"]];

    for args in cases {
        let assert = lnk().args(*args).assert().code(1);
        let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

        assert_eq!("error: command is not implemented yet\n", stderr);
    }
}

#[test]
fn init_creates_empty_path_repo_layout_and_initial_commit() {
    let temp = TempDir::new("init-empty");
    let repo = temp.path().join("machine-setup");

    lnk_with_git_identity()
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .success();

    assert_eq!(
        "format = 1\nname = \"machine-setup\"\n",
        fs::read_to_string(repo.join("lnk.toml")).expect("root manifest should exist")
    );
    assert!(repo.join("profiles/base/files").is_dir());
    assert!(repo.join("profiles/base/scripts").is_dir());
    assert_eq!(
        "lnk: initialize v2 setup repo\n",
        git_in(&repo, ["log", "-1", "--format=%s"])
    );
    assert_eq!("", git_in(&repo, ["status", "--porcelain"]));
}

#[test]
fn init_is_idempotent_for_existing_v2_repo() {
    let temp = TempDir::new("init-existing-v2");
    let repo = temp.path().join("lnk");

    lnk_with_git_identity()
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .success();

    fs::write(repo.join("lnk.toml"), "format = 1\nname = \"edited\"\n")
        .expect("edited manifest should write");
    fs::remove_dir_all(repo.join("profiles/base/scripts"))
        .expect("scripts dir should be removable");

    lnk_with_git_identity()
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .success();

    assert_eq!(
        "format = 1\nname = \"edited\"\n",
        fs::read_to_string(repo.join("lnk.toml")).expect("root manifest should still exist")
    );
    assert!(repo.join("profiles/base/files").is_dir());
    assert!(repo.join("profiles/base/scripts").is_dir());
    assert_eq!("1\n", git_in(&repo, ["rev-list", "--count", "HEAD"]));
}

#[test]
fn init_refuses_existing_unrelated_git_repo_with_commits() {
    let temp = TempDir::new("init-unrelated");
    let repo = temp.path().join("repo");

    git_init(&repo);
    fs::write(repo.join("README.md"), "not an lnk repo\n").expect("fixture should write");
    git_in(&repo, ["add", "README.md"]);
    git_in(&repo, ["commit", "-m", "initial unrelated commit"]);

    let assert = lnk_with_git_identity()
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .failure();
    let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

    assert_eq!(
        "error: existing Git repo has commits but no lnk.toml; use 'lnk migrate v1' or an empty directory\n",
        stderr
    );
    assert!(!repo.join("lnk.toml").exists());
}

#[test]
fn init_initial_commit_excludes_unrelated_staged_files() {
    let temp = TempDir::new("init-existing-empty-git");
    let repo = temp.path().join("repo");

    git_init(&repo);
    fs::write(repo.join("README.md"), "not an lnk file\n").expect("fixture should write");
    git_in(&repo, ["add", "README.md"]);

    lnk_with_git_identity()
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .success();

    assert_eq!(
        "lnk.toml\n",
        git_in(&repo, ["log", "-1", "--name-only", "--format="])
    );
    assert_eq!("A  README.md\n", git_in(&repo, ["status", "--porcelain"]));
}

#[test]
fn init_reports_missing_git_before_creating_repo() {
    let temp = TempDir::new("init-missing-git");
    let repo = temp.path().join("repo");
    let path = temp.path().join("empty-path");
    fs::create_dir(&path).expect("empty PATH dir should be created");

    let assert = lnk()
        .env("PATH", &path)
        .env("LNK_HOME", &repo)
        .arg("init")
        .assert()
        .failure();
    let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

    assert_eq!("error: git is required but was not found\n", stderr);
    assert!(!repo.exists());
}

#[test]
fn help_and_version_do_not_preflight_git() {
    let path = TempDir::new("empty-path");

    lnk()
        .env("PATH", path.path())
        .arg("--help")
        .assert()
        .success();
    lnk()
        .env("PATH", path.path())
        .arg("--version")
        .assert()
        .success();
}

#[test]
fn repo_commands_preflight_git() {
    let cases: &[&[&str]] = &[
        &["init"],
        &["profiles"],
        &["apply"],
        &["status"],
        &["run", "setup"],
        &["doctor"],
        &["migrate", "v1"],
    ];
    let path = TempDir::new("empty-path");

    for args in cases {
        let assert = lnk()
            .env("PATH", path.path())
            .args(*args)
            .assert()
            .failure();
        let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

        assert_eq!("error: git is required but was not found\n", stderr);
    }
}

#[test]
fn structured_repo_commands_render_missing_git_as_json() {
    let cases: &[&[&str]] = &[
        &["profiles", "--json"],
        &["apply", "base", "--dry-run", "--json"],
        &["status", "--json"],
        &["doctor", "--json"],
    ];
    let path = TempDir::new("empty-path");

    for args in cases {
        let assert = lnk()
            .env("PATH", path.path())
            .args(*args)
            .assert()
            .failure();
        let output = assert.get_output();
        let stderr = String::from_utf8_lossy(&output.stderr);

        assert_eq!("", String::from_utf8_lossy(&output.stdout));
        assert_eq!(
            "{\"error\":{\"code\":\"git.unavailable\",\"message\":\"git is required but was not found\"}}\n",
            stderr
        );
    }
}

#[test]
fn apply_dry_run_with_profiles_plans_symlink_without_mutating_home_or_state() {
    let temp = TempDir::new("apply-dry-run-profiles");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "base"])
        .assert()
        .success();
    let output = assert.get_output();

    assert_eq!(
        "apply plan:\nsymlink .zshrc <- profiles/base/files/.zshrc (base)\n",
        String::from_utf8_lossy(&output.stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&output.stderr));
    assert!(!home.join(".zshrc").exists());
    assert!(!state_home.join("lnk/machine.toml").exists());
}

#[test]
fn apply_dry_run_without_profiles_uses_local_active_profiles() {
    let temp = TempDir::new("apply-dry-run-active");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    write_state(&state_home, &["base"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run"])
        .assert()
        .success();
    let output = assert.get_output();

    assert_eq!(
        "apply plan:\nsymlink .zshrc <- profiles/base/files/.zshrc (base)\n",
        String::from_utf8_lossy(&output.stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&output.stderr));
    assert!(!home.join(".zshrc").exists());
}

#[test]
fn apply_dry_run_without_profiles_requires_existing_active_profiles() {
    let temp = TempDir::new("apply-dry-run-no-active");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run"])
        .assert()
        .failure();
    let output = assert.get_output();

    assert_eq!("", String::from_utf8_lossy(&output.stdout));
    assert_eq!(
        "error: run 'lnk apply <profile...>' first\n",
        String::from_utf8_lossy(&output.stderr)
    );
}

#[test]
fn apply_dry_run_json_classifies_unmanaged_conflicts_and_backup_plan() {
    let temp = TempDir::new("apply-dry-run-conflict");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".zshrc"), "local zshrc\n").expect("live file should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--json", "base"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"base\"],\"dry_run\":true,\"backup\":false,\"actions\":[{\"action\":\"conflict\",\"target\":\".zshrc\",\"reason\":\"unmanaged_target\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--backup", "--json", "base"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"base\"],\"dry_run\":true,\"backup\":true,\"actions\":[{\"action\":\"backup\",\"target\":\".zshrc\",\"reason\":\"unmanaged_target\"},{\"action\":\"symlink\",\"profile\":\"base\",\"source\":\"profiles/base/files/.zshrc\",\"target\":\".zshrc\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        "local zshrc\n",
        fs::read_to_string(home.join(".zshrc")).expect("dry-run should preserve live file")
    );
    assert!(!state_home.join("lnk/machine.toml").exists());
}

#[test]
fn apply_dry_run_json_classifies_copy_source_drift_from_state_hashes() {
    let temp = TempDir::new("apply-dry-run-copy-drift");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = "profiles/work/files/.gitconfig";
    let old_contents = "[user]\nname = Old\n";
    let new_contents = "[user]\nname = New\n";

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), old_contents).expect("live copy should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", new_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    write_state_with_copy_target(
        &state_home,
        &["work"],
        ".gitconfig",
        "work",
        source,
        &sha256(old_contents),
        &sha256(old_contents),
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"work\"],\"dry_run\":true,\"backup\":false,\"actions\":[{\"action\":\"copy\",\"profile\":\"work\",\"source\":\"profiles/work/files/.gitconfig\",\"target\":\".gitconfig\",\"drift\":\"source_changed\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        old_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("dry-run should preserve live copy")
    );
}

#[test]
fn apply_dry_run_json_includes_explicit_override_plan() {
    let temp = TempDir::new("apply-dry-run-override");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".gitconfig", "base config\n");
    write_profile_file(&repo, "work", ".gitconfig", "work config\n");
    write_profile_manifest(
        &repo,
        "work",
        "[files.\".gitconfig\"]\noverrides = \"base\"\n",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--json", "base", "work"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"base\",\"work\"],\"dry_run\":true,\"backup\":false,\"actions\":[{\"action\":\"override\",\"profile\":\"work\",\"target\":\".gitconfig\",\"overrides\":\"base\"},{\"action\":\"symlink\",\"profile\":\"work\",\"source\":\"profiles/work/files/.gitconfig\",\"target\":\".gitconfig\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert!(!home.join(".gitconfig").exists());
    assert!(!state_home.join("lnk/machine.toml").exists());
}

#[test]
fn apply_dry_run_json_classifies_local_copy_drift_and_backup_plan() {
    let temp = TempDir::new("apply-dry-run-local-copy-drift");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = "profiles/work/files/.gitconfig";
    let applied_contents = "[user]\nname = Applied\n";
    let local_contents = "[user]\nname = Local\n";

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), local_contents).expect("live copy should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", applied_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    write_state_with_copy_target(
        &state_home,
        &["work"],
        ".gitconfig",
        "work",
        source,
        &sha256(applied_contents),
        &sha256(applied_contents),
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"work\"],\"dry_run\":true,\"backup\":false,\"actions\":[{\"action\":\"copy_drift\",\"profile\":\"work\",\"source\":\"profiles/work/files/.gitconfig\",\"target\":\".gitconfig\",\"drift\":\"local_copy_drift\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--backup", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"work\"],\"dry_run\":true,\"backup\":true,\"actions\":[{\"action\":\"backup\",\"target\":\".gitconfig\",\"reason\":\"local_copy_drift\"},{\"action\":\"copy\",\"profile\":\"work\",\"source\":\"profiles/work/files/.gitconfig\",\"target\":\".gitconfig\",\"drift\":\"local_copy_drift\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        local_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("dry-run should preserve live copy")
    );
}

#[test]
fn apply_dry_run_json_classifies_divergent_copy_drift() {
    let temp = TempDir::new("apply-dry-run-divergent-copy-drift");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = "profiles/work/files/.gitconfig";
    let applied_contents = "[user]\nname = Applied\n";
    let source_contents = "[user]\nname = Source\n";
    let local_contents = "[user]\nname = Local\n";

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), local_contents).expect("live copy should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", source_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    write_state_with_copy_target(
        &state_home,
        &["work"],
        ".gitconfig",
        "work",
        source,
        &sha256(applied_contents),
        &sha256(applied_contents),
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--dry-run", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"profiles\":[\"work\"],\"dry_run\":true,\"backup\":false,\"actions\":[{\"action\":\"copy_drift\",\"profile\":\"work\",\"source\":\"profiles/work/files/.gitconfig\",\"target\":\".gitconfig\",\"drift\":\"divergent_copy\"}]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        local_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("dry-run should preserve live copy")
    );
}

#[test]
fn apply_creates_symlink_and_stores_active_profiles_after_success() {
    let temp = TempDir::new("apply-symlink");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    assert_eq!(
        repo.join("profiles/base/files/.zshrc"),
        fs::read_link(home.join(".zshrc")).expect("target should be a symlink")
    );

    let state = state_contents(&state_home);
    assert!(state.contains("active_profiles = [\"base\"]"), "{state}");
    assert!(state.contains("target = \".zshrc\""), "{state}");
    assert!(state.contains("mode = \"symlink\""), "{state}");
}

#[test]
fn apply_creates_parent_directories_for_mapped_symlink_targets() {
    let temp = TempDir::new("apply-nested-symlink");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", "init.lua", "vim.opt.number = true\n");
    write_profile_manifest(
        &repo,
        "base",
        "[files.\"init.lua\"]\ntarget = \".config/nvim/init.lua\"\n",
    );

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    assert_eq!(
        repo.join("profiles/base/files/init.lua"),
        fs::read_link(home.join(".config/nvim/init.lua")).expect("target should be a symlink")
    );
    assert!(home.join(".config/nvim").is_dir());

    let state = state_contents(&state_home);
    assert!(
        state.contains("target = \".config/nvim/init.lua\""),
        "{state}"
    );
}

#[test]
fn apply_replaces_existing_managed_symlink_on_reapply() {
    let temp = TempDir::new("apply-managed-symlink-reapply");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply"])
        .assert()
        .success();

    assert_eq!(
        repo.join("profiles/base/files/.zshrc"),
        fs::read_link(home.join(".zshrc")).expect("target should still be a managed symlink")
    );
}

#[test]
fn apply_refuses_to_replace_changed_managed_symlink_without_backup() {
    let temp = TempDir::new("apply-managed-symlink-changed");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    fs::remove_file(home.join(".zshrc")).expect("managed symlink should be removable");
    fs::write(home.join(".zshrc"), "local replacement\n").expect("local replacement should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--json"])
        .assert()
        .failure();

    assert_eq!(
        "{\"error\":{\"code\":\"apply.conflict\",\"message\":\"apply conflict\",\"target\":\".zshrc\",\"reason\":\"managed_target_changed\"}}\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
    assert_eq!(
        "local replacement\n",
        fs::read_to_string(home.join(".zshrc")).expect("changed live file should be preserved")
    );

    let state = state_contents(&state_home);
    assert!(state.contains("active_profiles = [\"base\"]"), "{state}");
    assert!(state.contains("target = \".zshrc\""), "{state}");
}

#[test]
fn apply_copies_file_and_records_copy_hashes() {
    let temp = TempDir::new("apply-copy");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let contents = "[user]\nname = Work\n";

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "work"])
        .assert()
        .success();

    assert_eq!(
        contents,
        fs::read_to_string(home.join(".gitconfig")).expect("copy target should exist")
    );

    let state = state_contents(&state_home);
    assert!(state.contains("active_profiles = [\"work\"]"), "{state}");
    assert!(state.contains("mode = \"copy\""), "{state}");
    assert!(
        state.contains(&format!("source_hash = \"{}\"", sha256(contents))),
        "{state}"
    );
    assert!(
        state.contains(&format!("applied_hash = \"{}\"", sha256(contents))),
        "{state}"
    );
}

#[test]
fn apply_with_backup_backs_up_drifted_copy_before_replacing_it() {
    let temp = TempDir::new("apply-copy-drift-backup");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = "profiles/work/files/.gitconfig";
    let applied_contents = "[user]\nname = Applied\n";
    let local_contents = "[user]\nname = Local\n";

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), local_contents).expect("live copy should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", applied_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    write_state_with_copy_target(
        &state_home,
        &["work"],
        ".gitconfig",
        "work",
        source,
        &sha256(applied_contents),
        &sha256(applied_contents),
    );

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--backup"])
        .assert()
        .success();

    assert_eq!(
        applied_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("copy target should be replaced")
    );
    let backups = backup_paths(&home, ".gitconfig");
    assert_eq!(1, backups.len(), "{backups:?}");
    assert_eq!(
        local_contents,
        fs::read_to_string(&backups[0]).expect("backup should preserve local drift")
    );

    let state = state_contents(&state_home);
    assert!(state.contains("active_profiles = [\"work\"]"), "{state}");
    assert!(
        state.contains(&format!("applied_hash = \"{}\"", sha256(applied_contents))),
        "{state}"
    );
}

#[test]
fn apply_never_runs_profile_scripts() {
    let temp = TempDir::new("apply-no-script-run");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let marker = temp.path().join("script-ran");
    let scripts_dir = repo.join("profiles/base/scripts");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    fs::create_dir_all(&scripts_dir).expect("scripts dir should be created");
    fs::write(
        scripts_dir.join("setup"),
        format!("#!/bin/sh\ntouch {}\n", marker.display()),
    )
    .expect("script should write");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    assert!(!marker.exists(), "apply must not run profile scripts");
    assert_eq!(
        repo.join("profiles/base/files/.zshrc"),
        fs::read_link(home.join(".zshrc")).expect("target should be a symlink")
    );
}

#[test]
fn run_executes_named_script_across_active_profiles_in_order() {
    let temp = TempDir::new("run-order");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    write_profile_script(&repo, "base", "setup", "#!/bin/sh\nprintf 'base\\n'\n");
    write_profile_script(&repo, "work", "setup", "#!/bin/sh\nprintf 'work\\n'\n");
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("setup")
        .assert()
        .success();

    assert_eq!(
        "base\nwork\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn run_profile_qualified_script_ignores_active_profile_order() {
    let temp = TempDir::new("run-profile-qualified");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    write_profile_script(&repo, "base", "setup", "#!/bin/sh\nprintf 'base\\n'\n");
    write_profile_script(&repo, "work", "setup", "#!/bin/sh\nprintf 'work\\n'\n");
    write_state(&state_home, &["base"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("work:setup")
        .assert()
        .success();

    assert_eq!(
        "work\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn run_profile_qualified_script_does_not_require_active_profiles() {
    let temp = TempDir::new("run-qualified-without-active-profiles");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    write_profile_script(&repo, "work", "setup", "#!/bin/sh\nprintf 'work\\n'\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("work:setup")
        .assert()
        .success();

    assert_eq!(
        "work\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn run_skips_missing_scripts_in_multi_profile_mode() {
    let temp = TempDir::new("run-missing-script");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    write_profile_script(&repo, "work", "setup", "#!/bin/sh\nprintf 'work\\n'\n");
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("setup")
        .assert()
        .success();

    assert_eq!(
        "work\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn run_profile_qualified_missing_script_fails() {
    let temp = TempDir::new("run-qualified-missing-script");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("base:setup")
        .assert()
        .failure();

    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stdout));
    assert_eq!(
        "error: script does not exist\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
}

#[test]
fn run_skips_scripts_gated_to_another_platform() {
    let temp = TempDir::new("run-platform-skip");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");
    let marker = temp.path().join("script-ran");
    let other_platform = match std::env::consts::OS {
        "macos" => "linux",
        "linux" => "macos",
        "windows" => "linux",
        _ => "macos",
    };

    write_root_manifest(&repo);
    write_profile_script(
        &repo,
        "base",
        "setup",
        &format!("#!/bin/sh\ntouch {}\n", marker.display()),
    );
    write_profile_manifest(
        &repo,
        "base",
        &format!("[scripts.setup]\nplatforms = [\"{other_platform}\"]\n"),
    );
    write_state(&state_home, &["base"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("setup")
        .assert()
        .success();

    assert_eq!(
        format!("skip base:setup: platform {}\n", std::env::consts::OS),
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
    assert!(!marker.exists(), "platform-gated script must not run");
}

#[test]
fn run_streams_script_stdout_and_stderr() {
    let temp = TempDir::new("run-stdio");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    write_profile_script(
        &repo,
        "base",
        "setup",
        "#!/bin/sh\nprintf 'out\\n'\nprintf 'err\\n' >&2\n",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("base:setup")
        .assert()
        .success();

    assert_eq!(
        "out\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        "err\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
}

#[test]
fn run_returns_non_zero_when_executed_script_fails() {
    let temp = TempDir::new("run-failure-code");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    write_profile_script(
        &repo,
        "base",
        "setup",
        "#!/bin/sh\nprintf 'before failure\\n'\nexit 7\n",
    );
    write_profile_script(&repo, "work", "setup", "#!/bin/sh\nprintf 'work\\n'\n");
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("run")
        .arg("setup")
        .assert()
        .failure();

    assert_eq!(
        "before failure\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!(
        "error: script failed\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
}

#[test]
fn apply_with_backup_renames_unmanaged_conflict_before_symlinking() {
    let temp = TempDir::new("apply-backup");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".zshrc"), "local zshrc\n").expect("live file should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--backup", "base"])
        .assert()
        .success();

    assert_eq!(
        repo.join("profiles/base/files/.zshrc"),
        fs::read_link(home.join(".zshrc")).expect("target should be a symlink")
    );
    let backups = backup_paths(&home, ".zshrc");
    assert_eq!(1, backups.len(), "{backups:?}");
    assert_eq!(
        "local zshrc\n",
        fs::read_to_string(&backups[0]).expect("backup should preserve original contents")
    );
}

#[test]
fn apply_stops_on_unmanaged_conflict_without_writing_state() {
    let temp = TempDir::new("apply-unmanaged-conflict");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".zshrc"), "local zshrc\n").expect("live file should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .failure();

    assert_eq!(
        "error: apply conflict\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
    assert_eq!(
        "local zshrc\n",
        fs::read_to_string(home.join(".zshrc")).expect("live file should be preserved")
    );
    assert!(!state_home.join("lnk/machine.toml").exists());

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "--json", "base"])
        .assert()
        .failure();

    assert_eq!(
        "{\"error\":{\"code\":\"apply.conflict\",\"message\":\"apply conflict\",\"target\":\".zshrc\",\"reason\":\"unmanaged_target\"}}\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
}

#[test]
fn failed_apply_preserves_previous_active_profiles() {
    let temp = TempDir::new("apply-preserve-active");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), "local config\n").expect("live file should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    write_profile_file(&repo, "work", ".gitconfig", "repo config\n");
    write_state(&state_home, &["base"]);

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "work"])
        .assert()
        .failure();

    assert_eq!(
        "format = 1\nactive_profiles = [\"base\"]\ntargets = []\n",
        state_contents(&state_home)
    );
}

#[test]
fn apply_stops_on_copy_drift_without_backup() {
    let temp = TempDir::new("apply-copy-drift-stop");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = "profiles/work/files/.gitconfig";
    let applied_contents = "[user]\nname = Applied\n";
    let local_contents = "[user]\nname = Local\n";

    fs::create_dir_all(&home).expect("home should be created");
    fs::write(home.join(".gitconfig"), local_contents).expect("live copy should write");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", applied_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    write_state_with_copy_target(
        &state_home,
        &["work"],
        ".gitconfig",
        "work",
        source,
        &sha256(applied_contents),
        &sha256(applied_contents),
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply"])
        .assert()
        .failure();

    assert_eq!(
        "error: copy drift blocks apply\n",
        String::from_utf8_lossy(&assert.get_output().stderr)
    );
    assert_eq!(
        local_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("live copy should be preserved")
    );
}

#[test]
fn status_json_reports_clean_state() {
    let temp = TempDir::new("status-clean");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["status", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"active_profiles\":[\"base\"],\"targets\":[{\"target\":\".zshrc\",\"profile\":\"base\",\"source\":\"profiles/base/files/.zshrc\",\"mode\":\"symlink\",\"status\":\"clean\"}],\"repo_dirty\":false,\"repo_changes\":[]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn status_reports_dirty_repo_from_git_porcelain() {
    let temp = TempDir::new("status-dirty-repo");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    write_profile_file(&repo, "base", ".zshrc", "changed zshrc\n");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .arg("status")
        .assert()
        .success();

    assert_eq!(
        "status:\nrepo: dirty\n M profiles/base/files/.zshrc\n\nactive profiles:\nbase\n\napplied targets:\nclean .zshrc <- profiles/base/files/.zshrc (base, symlink)\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
}

#[test]
fn status_detects_missing_symlink() {
    let temp = TempDir::new("status-missing-symlink");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();
    fs::remove_file(home.join(".zshrc")).expect("managed symlink should be removable");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["status", "--json"])
        .assert()
        .success();

    assert!(
        String::from_utf8_lossy(&assert.get_output().stdout)
            .contains("\"status\":\"missing_symlink\"")
    );
}

#[test]
fn status_detects_wrong_symlink() {
    let temp = TempDir::new("status-wrong-symlink");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();
    fs::remove_file(home.join(".zshrc")).expect("managed symlink should be removable");
    create_symlink(&repo.join("lnk.toml"), &home.join(".zshrc"));

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["status", "--json"])
        .assert()
        .success();

    assert!(
        String::from_utf8_lossy(&assert.get_output().stdout)
            .contains("\"status\":\"wrong_symlink\"")
    );
}

#[test]
fn status_detects_stale_symlink_without_mutating_it() {
    let temp = TempDir::new("status-stale-symlink");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let source = repo.join("profiles/base/files/.zshrc");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();
    fs::remove_file(&source).expect("source file should be removable");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["status", "--json"])
        .assert()
        .success();

    assert!(
        String::from_utf8_lossy(&assert.get_output().stdout)
            .contains("\"status\":\"stale_symlink\"")
    );
    assert_eq!(
        source,
        fs::read_link(home.join(".zshrc")).expect("status must not replace stale symlink")
    );
    assert!(
        !source.exists(),
        "status must not recreate missing repo source files"
    );
}

#[test]
fn status_detects_copy_drift() {
    let temp = TempDir::new("status-copy-drift");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let applied_contents = "[user]\nname = Applied\n";

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", applied_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");
    git_init(&repo);
    commit_all(&repo, "fixtures");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "work"])
        .assert()
        .success();
    fs::write(home.join(".gitconfig"), "[user]\nname = Local\n")
        .expect("live copy should be editable");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["status", "--json"])
        .assert()
        .success();

    assert!(
        String::from_utf8_lossy(&assert.get_output().stdout)
            .contains("\"status\":\"local_copy_drift\"")
    );
}

#[test]
fn doctor_json_reports_clean_health() {
    let temp = TempDir::new("doctor-clean");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();

    assert_eq!(
        "{\"issues\":[]}\n",
        String::from_utf8_lossy(&assert.get_output().stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&assert.get_output().stderr));
}

#[test]
fn doctor_reports_invalid_root_schema() {
    let temp = TempDir::new("doctor-invalid-root");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    fs::create_dir_all(&repo).expect("repo should be created");
    fs::write(repo.join("lnk.toml"), "name = \"missing-format\"\n")
        .expect("invalid root manifest should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .arg("doctor")
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(stdout.contains("root_schema_invalid"), "{stdout}");
    assert!(stdout.contains("schema file is invalid"), "{stdout}");
}

#[test]
fn doctor_reports_invalid_profile_manifest() {
    let temp = TempDir::new("doctor-invalid-profile-manifest");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    fs::write(
        repo.join("profiles/base/profile.toml"),
        "[files.\".zshrc\"]\nunsupported = true\n",
    )
    .expect("invalid profile manifest should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(
        stdout.contains("\"code\":\"profile_manifest_invalid\""),
        "{stdout}"
    );
    assert!(stdout.contains("\"profile\":\"base\""), "{stdout}");
}

#[test]
fn doctor_reports_invalid_script_platforms() {
    let temp = TempDir::new("doctor-invalid-script-platform");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    fs::create_dir_all(repo.join("profiles/base/scripts"))
        .expect("profile scripts dir should be created");
    fs::write(repo.join("profiles/base/scripts/setup"), "#!/bin/sh\n")
        .expect("script should write");
    write_profile_manifest(
        &repo,
        "base",
        "[scripts.setup]\nplatforms = [\"freebsd\"]\n",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(
        stdout.contains("\"code\":\"script_platform_invalid\""),
        "{stdout}"
    );
    assert!(stdout.contains("\"profile\":\"base\""), "{stdout}");
    assert!(stdout.contains("freebsd"), "{stdout}");
}

#[test]
fn doctor_reports_invalid_home_bound_target_paths() {
    let temp = TempDir::new("doctor-invalid-target");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    write_profile_manifest(
        &repo,
        "base",
        "[files.\".zshrc\"]\ntarget = \"../escape\"\n",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(
        stdout.contains("\"code\":\"target_path_invalid\""),
        "{stdout}"
    );
    assert!(stdout.contains("parent-escaping"), "{stdout}");
}

#[test]
fn doctor_reports_invalid_local_state_target_paths() {
    let temp = TempDir::new("doctor-invalid-state-target");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    write_state_with_symlink_target(
        &state_home,
        &["base"],
        "../escape",
        "base",
        "profiles/base/files/.zshrc",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(
        stdout.contains("\"code\":\"target_path_invalid\""),
        "{stdout}"
    );
    assert!(stdout.contains("parent-escaping"), "{stdout}");
}

#[test]
fn doctor_reports_duplicate_target_claims_from_active_profiles() {
    let temp = TempDir::new("doctor-duplicate-target");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".gitconfig", "base config\n");
    write_profile_file(&repo, "work", ".gitconfig", "work config\n");
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(stdout.contains("\"code\":\"target_duplicate\""), "{stdout}");
    assert!(stdout.contains("\"target\":\".gitconfig\""), "{stdout}");
}

#[test]
fn doctor_reports_invalid_override_declarations_from_active_profiles() {
    let temp = TempDir::new("doctor-invalid-override");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".gitconfig", "base config\n");
    write_profile_file(&repo, "work", ".gitconfig", "work config\n");
    write_profile_manifest(
        &repo,
        "work",
        "[files.\".gitconfig\"]\noverrides = \"mac\"\n",
    );
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(stdout.contains("\"code\":\"override_invalid\""), "{stdout}");
    assert!(stdout.contains("mac"), "{stdout}");
}

#[test]
fn doctor_reports_state_sources_that_no_longer_exist() {
    let temp = TempDir::new("doctor-missing-source");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    create_profile(&repo, "base");
    write_state_with_symlink_target(
        &state_home,
        &["base"],
        ".zshrc",
        "base",
        "profiles/base/files/.zshrc",
    );

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(
        stdout.contains("\"code\":\"state_source_missing\""),
        "{stdout}"
    );
    assert!(stdout.contains("\"target\":\".zshrc\""), "{stdout}");
}

#[test]
fn doctor_reports_copied_file_drift_without_mutating_it() {
    let temp = TempDir::new("doctor-copy-drift");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");
    let applied_contents = "[user]\nname = Applied\n";
    let local_contents = "[user]\nname = Local\n";

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "work", ".gitconfig", applied_contents);
    write_profile_manifest(&repo, "work", "[files.\".gitconfig\"]\nmode = \"copy\"\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "work"])
        .assert()
        .success();
    fs::write(home.join(".gitconfig"), local_contents).expect("live copy should be editable");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(stdout.contains("\"code\":\"copy_drift\""), "{stdout}");
    assert!(
        stdout.contains("\"status\":\"local_copy_drift\""),
        "{stdout}"
    );
    assert_eq!(
        local_contents,
        fs::read_to_string(home.join(".gitconfig")).expect("doctor must not rewrite drifted copy")
    );
}

#[test]
fn doctor_reports_missing_and_wrong_symlinks() {
    let temp = TempDir::new("doctor-symlinks");
    let repo = temp.path().join("repo");
    let home = temp.path().join("home");
    let state_home = temp.path().join("state");

    fs::create_dir_all(&home).expect("home should be created");
    write_root_manifest(&repo);
    write_profile_file(&repo, "base", ".zshrc", "repo zshrc\n");
    write_profile_file(&repo, "base", ".gitconfig", "repo config\n");

    lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["apply", "base"])
        .assert()
        .success();

    fs::remove_file(home.join(".zshrc")).expect("managed symlink should be removable");
    fs::remove_file(home.join(".gitconfig")).expect("managed symlink should be removable");
    create_symlink(&repo.join("lnk.toml"), &home.join(".gitconfig"));

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("HOME", &home)
        .env("XDG_STATE_HOME", &state_home)
        .args(["doctor", "--json"])
        .assert()
        .success();
    let stdout = String::from_utf8_lossy(&assert.get_output().stdout);

    assert!(stdout.contains("\"code\":\"symlink_missing\""), "{stdout}");
    assert!(
        stdout.contains("\"status\":\"missing_symlink\""),
        "{stdout}"
    );
    assert!(stdout.contains("\"code\":\"symlink_wrong\""), "{stdout}");
    assert!(stdout.contains("\"status\":\"wrong_symlink\""), "{stdout}");
}

#[test]
fn profiles_lists_repo_profiles_without_local_state() {
    let temp = TempDir::new("profiles-no-state");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "work");
    create_profile(&repo, "base");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("profiles")
        .assert()
        .success();
    let output = assert.get_output();

    assert_eq!(
        "profiles:\nbase\nwork\n",
        String::from_utf8_lossy(&output.stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&output.stderr));
    assert!(!state_home.join("lnk/machine.toml").exists());
}

#[test]
fn profiles_shows_active_profiles_when_local_state_exists() {
    let temp = TempDir::new("profiles-active-state");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    create_profile(&repo, "work");
    write_state(&state_home, &["base", "work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("profiles")
        .assert()
        .success();
    let output = assert.get_output();

    assert_eq!(
        "profiles:\nbase\nwork\n\nactive profiles:\nbase\nwork\n",
        String::from_utf8_lossy(&output.stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&output.stderr));
}

#[test]
fn profiles_json_outputs_profile_names_and_active_profiles() {
    let temp = TempDir::new("profiles-json");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    create_profile(&repo, "work");
    write_state(&state_home, &["work"]);

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .args(["profiles", "--json"])
        .assert()
        .success();
    let output = assert.get_output();

    assert_eq!(
        "{\"profiles\":[\"base\",\"work\"],\"active_profiles\":[\"work\"]}\n",
        String::from_utf8_lossy(&output.stdout)
    );
    assert_eq!("", String::from_utf8_lossy(&output.stderr));
}

#[test]
fn profiles_rejects_invalid_profile_manifest() {
    let temp = TempDir::new("profiles-invalid-manifest");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    fs::write(
        repo.join("profiles/base/profile.toml"),
        "[files.\".zshrc\"]\nunsupported = true\n",
    )
    .expect("invalid manifest should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("profiles")
        .assert()
        .failure();
    let output = assert.get_output();

    assert_eq!("", String::from_utf8_lossy(&output.stdout));
    assert_eq!(
        "error: schema file is invalid\n",
        String::from_utf8_lossy(&output.stderr)
    );
}

#[test]
fn profiles_rejects_invalid_local_state() {
    let temp = TempDir::new("profiles-invalid-state");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");
    let state_file = state_home.join("lnk").join("machine.toml");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    fs::create_dir_all(state_file.parent().expect("state path should have parent"))
        .expect("state dir should be created");
    fs::write(
        &state_file,
        "format = 1\nactive_profiles = [\"base\"]\nunknown = true\n",
    )
    .expect("invalid state should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .arg("profiles")
        .assert()
        .failure();
    let output = assert.get_output();

    assert_eq!("", String::from_utf8_lossy(&output.stdout));
    assert_eq!(
        "error: schema file is invalid\n",
        String::from_utf8_lossy(&output.stderr)
    );
}

#[test]
fn profiles_json_renders_manifest_errors_as_json() {
    let temp = TempDir::new("profiles-json-invalid-manifest");
    let repo = temp.path().join("repo");
    let state_home = temp.path().join("state");

    write_root_manifest(&repo);
    create_profile(&repo, "base");
    fs::write(
        repo.join("profiles/base/profile.toml"),
        "[files.\".zshrc\"]\nunsupported = true\n",
    )
    .expect("invalid manifest should write");

    let assert = lnk()
        .env("LNK_HOME", &repo)
        .env("XDG_STATE_HOME", &state_home)
        .args(["profiles", "--json"])
        .assert()
        .failure();
    let output = assert.get_output();

    assert_eq!("", String::from_utf8_lossy(&output.stdout));
    assert_eq!(
        "{\"error\":{\"code\":\"schema.invalid\",\"message\":\"schema file is invalid\"}}\n",
        String::from_utf8_lossy(&output.stderr)
    );
}

#[test]
fn rejects_partial_migration_confirmation() {
    for args in [
        &["migrate", "v1", "--write"][..],
        &["migrate", "v1", "--yes"],
    ] {
        let assert = lnk().args(args).assert().failure();
        let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

        assert!(stderr.contains("required"), "{stderr}");
    }
}

#[test]
fn json_flag_is_only_on_structured_commands() {
    for args in [&["init", "--json"][..], &["run", "setup", "--json"]] {
        let assert = lnk().args(args).assert().failure();
        let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

        assert!(stderr.contains("--json"), "{stderr}");
    }
}

#[test]
fn invalid_profile_script_target_is_rejected() {
    for target in ["", "work:", ":setup", "work:setup:again"] {
        let assert = lnk().args(["run", target]).assert().failure();
        let stderr = String::from_utf8_lossy(&assert.get_output().stderr);

        assert!(stderr.contains("PROFILE:SCRIPT"), "{stderr}");
    }
}
