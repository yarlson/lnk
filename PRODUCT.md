# Product

## Register

product

## Users

Confident, Git-fluent developers managing dotfiles across multiple machines. They already know `git status`, `git push`, and `git pull` by muscle memory and have either bounced off heavier tools (chezmoi, yadm, stow) or never wanted that ceremony in the first place. They reach for lnk in the terminal, often inside scripts and provisioning flows, and judge it by whether it stays out of the way and never surprises them about the state of their filesystem.

The job to be done: track a set of dotfiles in a normal Git repo, sync them between machines, and trust that nothing will silently overwrite or lose work along the way.

## Product Purpose

lnk is a single-binary Go CLI that turns dotfiles management into normal Git usage. It moves files into a managed Git repo, replaces them with symlinks, and exposes a small surface of commands (`add`, `rm`, `push`, `pull`, `status`, `diff`, `list`, `doctor`, `bootstrap`) that map directly to the user's mental model.

Success means the user adopts lnk in minutes, runs commands in scripts and at the prompt without re-reading help, and never has to debug what the tool did to their files.

The designed surface is the terminal: command output, error messages, status lines, doctor reports, dry-run previews, and the cues that signal what changed on disk.

## Brand Personality

Quiet, precise, trustworthy.

Voice is calm and declarative. Output reports what happened, in the user's terms, in as few words as the situation needs. No celebration, no preamble, no internal monologue. Errors are direct and actionable; they name the file, the cause, and the way out.

The emotional goal is trust. The user should finish every command feeling they know exactly what is on disk and that nothing happened that they did not ask for.

## Anti-references

lnk is not verbose enterprise tooling. No `Initializing context...`, no `Loading providers...`, no INFO/DEBUG noise on success, no multi-line preamble before the actual work. Terraform-style "what I am about to do" narration and AWS-CLI-style ceremony are the wrong shape. If a command has nothing to report, it reports nothing.

## Design Principles

- **Show, don't narrate.** Output reflects what changed on the filesystem and in Git, not the tool's internal steps. No status updates the user did not ask for. Quiet on success; specific on failure.
- **Trust through reversibility.** Every destructive operation is recoverable. Adds roll back on failure. Pulls and doctor back up real files to `.lnk-backup` instead of overwriting. Dry-run is a first-class affordance, not an afterthought. The user feels safe because they are safe.
- **Plain over clever.** State is inspectable on disk: real files, real symlinks, plain-text `.lnk` indexes. Output is predictable and grep-friendly. No hidden metadata, no magic resolution, no auto-host inference. Boring, legible behavior beats impressive behavior.

## Accessibility & Inclusion

Standard CLI accessibility:

- Honor `NO_COLOR` and `--colors never`. Color is decoration; meaning is carried by words.
- Honor `--no-emoji` and `--quiet`/`-q`. The CLI must work fully without color or emoji for users who pipe to scripts, log files, or non-TTY environments.
- Detect non-TTY output and degrade gracefully: no ANSI escapes, no progress animations, no interactive prompts in pipes.
- Exit codes are reliable. Scripts can rely on them without parsing output.
- Error messages stand on their own as readable English; they do not require visual styling to be understood.
