#![allow(dead_code)]

use std::{
    ffi::{OsStr, OsString},
    io,
    path::{Path, PathBuf},
    process::{Command, Output},
};

use crate::errors::Error;

const REPO_SELECTION_ENV: &[&str] = &["GIT_DIR", "GIT_WORK_TREE"];

pub(crate) fn preflight() -> Result<(), Error> {
    run_anywhere(["--version"])?.require_success()
}

pub(crate) fn init(repo: &Path) -> Result<(), Error> {
    run_anywhere([
        GitArg::Borrowed("init"),
        GitArg::Owned(repo.as_os_str().to_os_string()),
    ])?
    .require_success()
}

pub(crate) fn is_repo(repo: &Path) -> Result<bool, Error> {
    if !repo.exists() {
        return Ok(false);
    }

    let output = run_in_repo(repo, ["rev-parse", "--show-toplevel"])?;

    Ok(output.success && same_path(repo, Path::new(output.stdout.trim())))
}

pub(crate) fn has_commits(repo: &Path) -> Result<bool, Error> {
    Ok(run_in_repo(repo, ["rev-parse", "--verify", "HEAD"])?.success)
}

pub(crate) fn is_dirty(repo: &Path) -> Result<bool, Error> {
    Ok(!status_porcelain(repo)?.is_empty())
}

pub(crate) fn status_porcelain(repo: &Path) -> Result<Vec<String>, Error> {
    let output = run_in_repo(repo, ["--no-optional-locks", "status", "--porcelain=v1"])?;
    output.require_success()?;

    Ok(output.stdout.lines().map(str::to_string).collect())
}

pub(crate) fn add_all(repo: &Path) -> Result<(), Error> {
    run_in_repo(repo, ["add", "--all"])?.require_success()
}

pub(crate) fn add_paths(repo: &Path, pathspecs: &[&str]) -> Result<(), Error> {
    let mut args = Vec::with_capacity(pathspecs.len() + 1);
    args.push(GitArg::Borrowed("add"));
    args.extend(pathspecs.iter().copied().map(GitArg::Borrowed));

    run_in_repo(repo, args)?.require_success()
}

pub(crate) fn commit(repo: &Path, message: &str) -> Result<(), Error> {
    run_in_repo(repo, ["commit", "-m", message])?.require_success()
}

pub(crate) fn commit_paths(repo: &Path, message: &str, pathspecs: &[&str]) -> Result<(), Error> {
    let mut args = Vec::with_capacity(pathspecs.len() + 3);
    args.extend([
        GitArg::Borrowed("commit"),
        GitArg::Borrowed("-m"),
        GitArg::Borrowed(message),
        GitArg::Borrowed("--"),
    ]);
    args.extend(pathspecs.iter().copied().map(GitArg::Borrowed));

    run_in_repo(repo, args)?.require_success()
}

pub(crate) fn add_and_commit(repo: &Path, message: &str) -> Result<(), Error> {
    add_all(repo)?;
    commit(repo, message)
}

pub(crate) fn create_tag(repo: &Path, tag: &str) -> Result<(), Error> {
    run_in_repo(repo, ["tag", tag])?.require_success()
}

#[derive(Debug)]
struct GitOutput {
    command: String,
    success: bool,
    stdout: String,
    stderr: String,
}

impl GitOutput {
    fn require_success(&self) -> Result<(), Error> {
        if self.success {
            Ok(())
        } else {
            Err(Error::GitCommandFailed {
                command: self.command.clone(),
                stderr: self.stderr.clone(),
            })
        }
    }
}

fn run_in_repo<'a, I, S>(repo: &Path, args: I) -> Result<GitOutput, Error>
where
    I: IntoIterator<Item = S>,
    S: Into<GitArg<'a>>,
{
    let mut git_args = vec![
        GitArg::Borrowed("-C"),
        GitArg::Owned(repo.as_os_str().to_os_string()),
    ];
    git_args.extend(args.into_iter().map(Into::into));
    run_anywhere(git_args)
}

fn run_anywhere<'a, I, S>(args: I) -> Result<GitOutput, Error>
where
    I: IntoIterator<Item = S>,
    S: Into<GitArg<'a>>,
{
    let args: Vec<GitArg<'_>> = args.into_iter().map(Into::into).collect();
    let mut command = git_command();

    for arg in &args {
        command.arg(arg.as_os_str());
    }

    output_to_result(command.output(), display_command(&args))
}

fn git_command() -> Command {
    let mut command = Command::new("git");

    for variable in REPO_SELECTION_ENV {
        command.env_remove(variable);
    }

    command
}

fn output_to_result(output: io::Result<Output>, command: String) -> Result<GitOutput, Error> {
    match output {
        Ok(output) => Ok(GitOutput {
            command,
            success: output.status.success(),
            stdout: String::from_utf8_lossy(&output.stdout).into_owned(),
            stderr: String::from_utf8_lossy(&output.stderr).trim().to_string(),
        }),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Err(Error::GitUnavailable),
        Err(error) => Err(Error::GitCommandFailed {
            command,
            stderr: error.to_string(),
        }),
    }
}

#[derive(Clone, Debug)]
enum GitArg<'a> {
    Borrowed(&'a str),
    Owned(OsString),
}

impl GitArg<'_> {
    fn as_os_str(&self) -> &OsStr {
        match self {
            Self::Borrowed(value) => OsStr::new(value),
            Self::Owned(value) => value,
        }
    }
}

impl<'a> From<&'a str> for GitArg<'a> {
    fn from(value: &'a str) -> Self {
        Self::Borrowed(value)
    }
}

impl From<String> for GitArg<'_> {
    fn from(value: String) -> Self {
        Self::Owned(value.into())
    }
}

impl From<&std::ffi::OsStr> for GitArg<'_> {
    fn from(value: &std::ffi::OsStr) -> Self {
        Self::Owned(value.to_os_string())
    }
}

fn display_command(args: &[GitArg<'_>]) -> String {
    let mut command = String::from("git");

    for arg in args {
        command.push(' ');
        command.push_str(&arg.as_os_str().to_string_lossy());
    }

    command
}

fn same_path(left: &Path, right: &Path) -> bool {
    comparable_path(left) == comparable_path(right)
}

fn comparable_path(path: &Path) -> PathBuf {
    path.canonicalize().unwrap_or_else(|_| path.to_path_buf())
}

#[cfg(test)]
mod tests {
    use std::{
        ffi::OsStr,
        fs,
        path::{Path, PathBuf},
        time::{SystemTime, UNIX_EPOCH},
    };

    use super::{add_and_commit, create_tag, has_commits, init, is_dirty, is_repo, preflight};

    struct TempDir {
        path: PathBuf,
    }

    impl TempDir {
        fn new(name: &str) -> Self {
            let nonce = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .expect("system clock should be after unix epoch")
                .as_nanos();
            let path =
                std::env::temp_dir().join(format!("lnk-{name}-{}-{nonce}", std::process::id()));

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
    fn preflight_accepts_available_git() {
        preflight().expect("git should be available in the test environment");
    }

    #[test]
    fn git_commands_ignore_inherited_repo_selection_environment() {
        let command = super::git_command();
        let envs: Vec<_> = command.get_envs().collect();

        for variable in super::REPO_SELECTION_ENV {
            assert!(
                envs.iter()
                    .any(|(key, value)| key == &OsStr::new(variable) && value.is_none()),
                "{variable} should be removed from git subprocess environment"
            );
        }
    }

    #[test]
    fn init_creates_git_repo_without_commits() {
        let temp = TempDir::new("git-init");
        let repo = temp.path().join("repo");
        let plain_dir = temp.path().join("plain");
        fs::create_dir(&plain_dir).expect("plain fixture dir should be created");

        assert!(!is_repo(&repo).expect("missing repo check should succeed"));
        assert!(!is_repo(&plain_dir).expect("plain dir repo check should succeed"));

        init(&repo).expect("git init should create repo");

        assert!(is_repo(&repo).expect("repo check should succeed"));
        assert!(!has_commits(&repo).expect("new repo should not have commits"));
        assert!(!is_dirty(&repo).expect("empty repo should not be dirty"));
    }

    #[test]
    fn is_repo_requires_the_path_to_be_the_worktree_root() {
        let temp = TempDir::new("git-root");
        let repo = temp.path().join("repo");
        let nested = repo.join("nested");

        init(&repo).expect("git init should create repo");
        fs::create_dir(&nested).expect("nested dir should be created");

        assert!(is_repo(&repo).expect("repo root check should succeed"));
        assert!(!is_repo(&nested).expect("nested dir check should succeed"));
    }

    #[test]
    fn add_commit_dirty_state_and_tag_helpers_use_git_repo() {
        let temp = TempDir::new("git-helpers");
        let repo = temp.path().join("repo");

        init(&repo).expect("git init should create repo");
        super::run_in_repo(&repo, ["config", "user.name", "lnk test"])
            .expect("git config should run")
            .require_success()
            .expect("git user.name config should succeed");
        super::run_in_repo(&repo, ["config", "user.email", "lnk@example.invalid"])
            .expect("git config should run")
            .require_success()
            .expect("git user.email config should succeed");
        fs::write(repo.join("lnk.toml"), "format = 1\n").expect("fixture should write");

        assert!(is_dirty(&repo).expect("untracked file should make repo dirty"));

        add_and_commit(&repo, "lnk: initialize v2 setup repo").expect("commit should succeed");

        assert!(has_commits(&repo).expect("repo should have a commit"));
        assert!(!is_dirty(&repo).expect("committed repo should be clean"));

        fs::write(repo.join("lnk.toml"), "format = 1\nname = \"changed\"\n")
            .expect("fixture should modify tracked file");
        assert!(is_dirty(&repo).expect("tracked modification should make repo dirty"));

        add_and_commit(&repo, "lnk: update v2 setup repo").expect("second commit should succeed");
        assert!(!is_dirty(&repo).expect("second commit should clean repo"));

        create_tag(&repo, "lnk-migration-backup").expect("tag should be created");
        super::run_in_repo(
            &repo,
            ["rev-parse", "--verify", "refs/tags/lnk-migration-backup"],
        )
        .expect("git tag verification should run")
        .require_success()
        .expect("tag should exist");
    }
}
