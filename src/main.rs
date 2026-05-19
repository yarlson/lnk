fn main() -> std::process::ExitCode {
    use clap::Parser;

    match lnk::cli::Cli::try_parse() {
        Ok(cli) => {
            let command = cli.into_command();
            let output_format = command.output_format();
            render(output_format, lnk::app::dispatch(command))
        }
        Err(error) => {
            let exit_code = error.exit_code();
            let _ = error.print();
            exit_code_from_i32(exit_code)
        }
    }
}

fn render(
    output_format: lnk::cli::OutputFormat,
    result: lnk::app::Result<lnk::app::Output>,
) -> std::process::ExitCode {
    match result {
        Ok(lnk::app::Output::Empty) => std::process::ExitCode::SUCCESS,
        Ok(lnk::app::Output::ApplyPlan(output)) => {
            render_apply_plan(output_format, &output);
            std::process::ExitCode::SUCCESS
        }
        Ok(lnk::app::Output::Doctor(output)) => {
            render_doctor(output_format, &output);
            std::process::ExitCode::SUCCESS
        }
        Ok(lnk::app::Output::MigrationPlan(output)) => {
            render_migration_plan(&output);
            std::process::ExitCode::SUCCESS
        }
        Ok(lnk::app::Output::Profiles(output)) => {
            render_profiles(output_format, &output);
            std::process::ExitCode::SUCCESS
        }
        Ok(lnk::app::Output::Status(output)) => {
            render_status(output_format, &output);
            std::process::ExitCode::SUCCESS
        }
        Err(error) => {
            render_error(output_format, &error);
            std::process::ExitCode::FAILURE
        }
    }
}

fn render_migration_plan(output: &lnk::app::MigrationPlanOutput) {
    println!("migration plan:");

    for action in &output.actions {
        match action {
            lnk::app::MigrationActionOutput::WriteRootManifest => {
                println!("write lnk.toml");
            }
            lnk::app::MigrationActionOutput::MoveFile { source, target } => {
                println!("move {source} -> {target}");
            }
            lnk::app::MigrationActionOutput::MoveScript { source, target } => {
                println!("move {source} -> {target}");
            }
            lnk::app::MigrationActionOutput::WriteOverride {
                profile,
                target,
                overrides,
            } => {
                println!("override {target}: {profile} overrides {overrides}");
            }
            lnk::app::MigrationActionOutput::RemoveTrackingFile { path } => {
                println!("remove {path}");
            }
            lnk::app::MigrationActionOutput::RemoveEmptyHostDir { path } => {
                println!("remove empty {path}");
            }
        }
    }

    println!("backup tag: {}", output.backup_tag);
    println!("commit: {}", output.commit_message);
    println!("local profiles: {}", output.local_profiles.join(" "));

    if output.wrote {
        println!("local state written");
    }
}

fn render_doctor(output_format: lnk::cli::OutputFormat, output: &lnk::app::DoctorOutput) {
    match output_format {
        lnk::cli::OutputFormat::Human => render_doctor_human(output),
        lnk::cli::OutputFormat::Json => {
            println!(
                "{}",
                serde_json::to_string(output).expect("doctor output should serialize")
            );
        }
    }
}

fn render_doctor_human(output: &lnk::app::DoctorOutput) {
    println!("doctor:");

    if output.issues.is_empty() {
        println!("ok");
        return;
    }

    for issue in &output.issues {
        print!("{}: {}", issue.code, issue.message);

        if let Some(path) = &issue.path {
            print!(" path={path}");
        }
        if let Some(profile) = &issue.profile {
            print!(" profile={profile}");
        }
        if let Some(target) = &issue.target {
            print!(" target={target}");
        }
        if let Some(source) = &issue.source {
            print!(" source={source}");
        }
        if let Some(status) = issue.status {
            print!(" status={}", status_label(status));
        }

        println!();
    }
}

fn render_status(output_format: lnk::cli::OutputFormat, output: &lnk::app::StatusOutput) {
    match output_format {
        lnk::cli::OutputFormat::Human => render_status_human(output),
        lnk::cli::OutputFormat::Json => {
            println!(
                "{}",
                serde_json::to_string(output).expect("status output should serialize")
            );
        }
    }
}

fn render_status_human(output: &lnk::app::StatusOutput) {
    println!("status:");
    if output.repo_dirty {
        println!("repo: dirty");
        for change in &output.repo_changes {
            println!("{change}");
        }
    } else {
        println!("repo: clean");
    }

    println!();
    println!("active profiles:");
    for profile in &output.active_profiles {
        println!("{profile}");
    }

    println!();
    println!("applied targets:");
    for target in &output.targets {
        println!(
            "{} {} <- {} ({}, {})",
            status_label(target.status),
            target.target,
            target.source,
            target.profile,
            target.mode
        );
    }
}

fn status_label(status: lnk::app::StatusTargetStatus) -> &'static str {
    match status {
        lnk::app::StatusTargetStatus::Clean => "clean",
        lnk::app::StatusTargetStatus::MissingSymlink => "missing symlink",
        lnk::app::StatusTargetStatus::StaleSymlink => "stale symlink",
        lnk::app::StatusTargetStatus::WrongSymlink => "wrong symlink",
        lnk::app::StatusTargetStatus::MissingCopy => "missing copy",
        lnk::app::StatusTargetStatus::WrongCopyType => "wrong copy type",
        lnk::app::StatusTargetStatus::SourceChanged => "source changed",
        lnk::app::StatusTargetStatus::LocalCopyDrift => "local copy drift",
        lnk::app::StatusTargetStatus::DivergentCopy => "divergent copy",
        lnk::app::StatusTargetStatus::UnknownCopyState => "unknown copy state",
    }
}

fn render_apply_plan(output_format: lnk::cli::OutputFormat, output: &lnk::app::ApplyPlanOutput) {
    match output_format {
        lnk::cli::OutputFormat::Human => render_apply_plan_human(output),
        lnk::cli::OutputFormat::Json => {
            println!(
                "{}",
                serde_json::to_string(output).expect("apply plan output should serialize")
            );
        }
    }
}

fn render_apply_plan_human(output: &lnk::app::ApplyPlanOutput) {
    println!("apply plan:");

    for action in &output.actions {
        match action {
            lnk::app::ApplyPlanAction::Override {
                profile,
                target,
                overrides,
            } => {
                println!("override {target}: {profile} overrides {overrides}");
            }
            lnk::app::ApplyPlanAction::Backup { target, reason } => {
                println!("backup {target}: {reason}");
            }
            lnk::app::ApplyPlanAction::Conflict { target, reason } => {
                println!("conflict {target}: {reason}");
            }
            lnk::app::ApplyPlanAction::CopyDrift { target, drift, .. } => {
                println!("copy drift {target}: {drift}");
            }
            lnk::app::ApplyPlanAction::Symlink {
                profile,
                source,
                target,
            } => {
                println!("symlink {target} <- {source} ({profile})");
            }
            lnk::app::ApplyPlanAction::Copy {
                profile,
                source,
                target,
                drift,
            } => {
                if let Some(drift) = drift {
                    println!("copy {target} <- {source} ({profile}, {drift})");
                } else {
                    println!("copy {target} <- {source} ({profile})");
                }
            }
        }
    }
}

fn render_profiles(output_format: lnk::cli::OutputFormat, output: &lnk::app::ProfilesOutput) {
    match output_format {
        lnk::cli::OutputFormat::Human => render_profiles_human(output),
        lnk::cli::OutputFormat::Json => {
            println!(
                "{}",
                serde_json::to_string(output).expect("profiles output should serialize")
            );
        }
    }
}

fn render_profiles_human(output: &lnk::app::ProfilesOutput) {
    println!("profiles:");
    for profile in &output.profiles {
        println!("{profile}");
    }

    if output.local_state_exists {
        println!();
        println!("active profiles:");
        for profile in &output.active_profiles {
            println!("{profile}");
        }
    }
}

fn render_error(output_format: lnk::cli::OutputFormat, error: &lnk::errors::Error) {
    match output_format {
        lnk::cli::OutputFormat::Human => eprintln!("error: {}", error.message()),
        lnk::cli::OutputFormat::Json => eprintln!("{}", json_error(error)),
    }
}

fn json_error(error: &lnk::errors::Error) -> String {
    let mut json = format!(
        "{{\"error\":{{\"code\":\"{}\",\"message\":\"{}\"",
        json_escape(error.code()),
        json_escape(error.message())
    );

    if let Some(command) = error.command() {
        json.push_str(&format!(",\"command\":\"{}\"", json_escape(command)));
    }

    match error {
        lnk::errors::Error::ApplyConflict { target, reason } => {
            json.push_str(&format!(
                ",\"target\":\"{}\",\"reason\":\"{}\"",
                json_escape(target),
                json_escape(reason)
            ));
        }
        lnk::errors::Error::CopyDrift { target, drift } => {
            json.push_str(&format!(
                ",\"target\":\"{}\",\"drift\":\"{}\"",
                json_escape(target),
                json_escape(drift)
            ));
        }
        _ => {}
    }

    json.push_str("}}");
    json
}

fn json_escape(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len());

    for character in value.chars() {
        match character {
            '"' => escaped.push_str("\\\""),
            '\\' => escaped.push_str("\\\\"),
            '\n' => escaped.push_str("\\n"),
            '\r' => escaped.push_str("\\r"),
            '\t' => escaped.push_str("\\t"),
            character if character.is_control() => {
                escaped.push_str(&format!("\\u{:04x}", character as u32));
            }
            character => escaped.push(character),
        }
    }

    escaped
}

fn exit_code_from_i32(code: i32) -> std::process::ExitCode {
    match code {
        0 => std::process::ExitCode::SUCCESS,
        code => std::process::ExitCode::from(code as u8),
    }
}
