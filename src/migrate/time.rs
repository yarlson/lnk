use std::time::{SystemTime, UNIX_EPOCH};

use crate::errors::Error;

pub(super) fn timestamp(now: SystemTime) -> Result<String, Error> {
    let seconds = now
        .duration_since(UNIX_EPOCH)
        .map_err(|_| Error::MigrationInvalid {
            message: "system clock is before the unix epoch",
        })?
        .as_secs() as i64;
    let (year, month, day, hour, minute, second) = unix_utc(seconds);

    Ok(format!(
        "{year:04}{month:02}{day:02}-{hour:02}{minute:02}{second:02}"
    ))
}

fn unix_utc(seconds: i64) -> (i64, i64, i64, i64, i64, i64) {
    let days = seconds.div_euclid(86_400);
    let seconds_of_day = seconds.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    let hour = seconds_of_day / 3_600;
    let minute = (seconds_of_day % 3_600) / 60;
    let second = seconds_of_day % 60;

    (year, month, day, hour, minute, second)
}

fn civil_from_days(days: i64) -> (i64, i64, i64) {
    let days = days + 719_468;
    let era = days.div_euclid(146_097);
    let day_of_era = days - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1_460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let year = year_of_era + era * 400;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_prime = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_prime + 2) / 5 + 1;
    let month = month_prime + if month_prime < 10 { 3 } else { -9 };
    let year = year + if month <= 2 { 1 } else { 0 };

    (year, month, day)
}

#[cfg(test)]
mod tests {
    use std::time::{Duration, UNIX_EPOCH};

    #[test]
    fn unix_timestamp_uses_required_backup_tag_format() {
        assert_eq!(
            super::timestamp(UNIX_EPOCH + Duration::from_secs(1_704_067_199))
                .expect("timestamp should format"),
            "20231231-235959"
        );
    }
}
