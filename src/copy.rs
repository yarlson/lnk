use std::{
    fmt::Write as _,
    fs,
    path::{Path, PathBuf},
};

use sha2::{Digest, Sha256};

use crate::{errors::Error, state::TargetState};

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum CopyDrift {
    Clean,
    SourceChanged,
    LocalDrift,
    Divergent,
    Unknown,
}

pub(crate) fn classify_copy_drift(
    previous: Option<&TargetState>,
    current_source_hash: &str,
    current_live_hash: Option<&str>,
) -> CopyDrift {
    let Some(previous) = previous else {
        return CopyDrift::Unknown;
    };
    let (Some(stored_source_hash), Some(stored_live_hash), Some(current_live_hash)) = (
        previous.source_hash.as_deref(),
        previous.applied_hash.as_deref(),
        current_live_hash,
    ) else {
        return CopyDrift::Unknown;
    };

    drift_from_hashes(
        stored_source_hash,
        stored_live_hash,
        current_source_hash,
        current_live_hash,
    )
}

pub(crate) fn hash_file(path: &Path) -> Result<String, Error> {
    let contents = fs::read(path).map_err(|error| Error::FileReadFailed {
        path: path_to_string(path),
        message: error.to_string(),
    })?;
    let digest = Sha256::digest(&contents);
    let mut hex = String::with_capacity(64);

    for byte in digest {
        write!(hex, "{byte:02x}").expect("writing to string should not fail");
    }

    Ok(format!("sha256:{hex}"))
}

fn drift_from_hashes(
    stored_source_hash: &str,
    stored_live_hash: &str,
    current_source_hash: &str,
    current_live_hash: &str,
) -> CopyDrift {
    match (
        stored_source_hash != current_source_hash,
        stored_live_hash != current_live_hash,
    ) {
        (false, false) => CopyDrift::Clean,
        (true, false) => CopyDrift::SourceChanged,
        (false, true) => CopyDrift::LocalDrift,
        (true, true) => CopyDrift::Divergent,
    }
}

fn path_to_string(path: &Path) -> String {
    path.as_os_str().to_string_lossy().into_owned()
}

pub(crate) fn path_from_home(home: &Path, target: &str) -> Result<PathBuf, Error> {
    let target = crate::paths::normalize_target_path(Path::new(target))?;
    crate::paths::target_path(home, &target)
}

#[cfg(test)]
mod tests {
    use super::{CopyDrift, drift_from_hashes};

    #[test]
    fn classifies_copy_drift_matrix() {
        assert_eq!(
            drift_from_hashes("source-a", "live-a", "source-a", "live-a"),
            CopyDrift::Clean
        );
        assert_eq!(
            drift_from_hashes("source-a", "live-a", "source-b", "live-a"),
            CopyDrift::SourceChanged
        );
        assert_eq!(
            drift_from_hashes("source-a", "live-a", "source-a", "live-b"),
            CopyDrift::LocalDrift
        );
        assert_eq!(
            drift_from_hashes("source-a", "live-a", "source-b", "live-b"),
            CopyDrift::Divergent
        );
    }
}
