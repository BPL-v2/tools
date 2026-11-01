use std::time::{Duration, Instant};

#[derive(Debug, Clone, PartialEq)]
struct Policy {
    max_hits: usize,
    period: Duration,
}

impl Policy {
    fn current_hits(&self, request_times: &[Instant], now: Instant) -> usize {
        request_times
            .iter()
            .filter(|time| now.duration_since(**time) < self.period)
            .count()
    }
}

pub struct RateLimiter {
    policies: Vec<Policy>,
    request_times: Vec<Instant>,
}

impl RateLimiter {
    pub fn new() -> Self {
        Self {
            // Default fallback limits if headers are not available
            policies: vec![
                Policy { max_hits: 60, period: Duration::from_secs(60) },
                Policy { max_hits: 120, period: Duration::from_secs(300) },
                Policy { max_hits: 300, period: Duration::from_secs(3600) },
            ],
            request_times: Vec::new(),
        }
    }

    /// Takes over the limits from the X-Rate-Limit-Account header
    /// ("60:60:10,120:300:300") and the usage from X-Rate-Limit-Account-State
    /// ("1:60:0,1:300:0").
    pub fn update(&mut self, limits: &str, state: &str, now: Instant) -> Result<(), String> {
        let limits: Vec<&str> = limits.split(',').collect();
        let states: Vec<&str> = state.split(',').collect();
        if limits.len() != states.len() {
            return Err("mismatch between rate limit and state headers".to_string());
        }

        let mut policies = Vec::with_capacity(limits.len());
        for (limit, state) in limits.iter().zip(&states) {
            let limit: Vec<&str> = limit.split(':').collect();
            let state: Vec<&str> = state.split(':').collect();
            if limit.len() != 3 || state.len() != 3 {
                continue;
            }
            let policy = Policy {
                max_hits: limit[0].parse().unwrap_or(0),
                period: Duration::from_secs(limit[1].parse().unwrap_or(0)),
            };
            // Add missing timestamps to match the server's reported usage
            let usage: usize = state[0].parse().unwrap_or(0);
            let tracked = policy.current_hits(&self.request_times, now);
            self.request_times
                .extend(std::iter::repeat_n(now, usage.saturating_sub(tracked)));
            policies.push(policy);
        }
        if !policies.is_empty() {
            self.policies = policies;
        }
        self.clean_expired(now);
        Ok(())
    }

    fn clean_expired(&mut self, now: Instant) {
        let longest = self.policies.iter().map(|policy| policy.period).max();
        let longest = longest.unwrap_or_default();
        self.request_times
            .retain(|time| now.duration_since(*time) < longest);
    }

    /// Records a request if it would not violate any policy.
    pub fn try_acquire(&mut self, now: Instant) -> bool {
        let allowed = self
            .policies
            .iter()
            .all(|policy| policy.current_hits(&self.request_times, now) < policy.max_hits);
        if allowed {
            self.request_times.push(now);
            self.clean_expired(now);
        }
        allowed
    }

    pub fn state(&self, now: Instant) -> String {
        self.policies
            .iter()
            .map(|policy| {
                format!(
                    "{}/{} per {}s",
                    policy.current_hits(&self.request_times, now),
                    policy.max_hits,
                    policy.period.as_secs()
                )
            })
            .collect::<Vec<_>>()
            .join(", ")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn blocks_at_the_limit_until_the_window_passes() {
        let start = Instant::now();
        let mut limiter = RateLimiter::new();
        limiter.update("2:10:60", "0:10:0", start).unwrap();
        assert!(limiter.try_acquire(start));
        assert!(limiter.try_acquire(start));
        assert!(!limiter.try_acquire(start));
        assert!(limiter.try_acquire(start + Duration::from_secs(10)));
    }

    #[test]
    fn adopts_server_reported_usage() {
        let now = Instant::now();
        let mut limiter = RateLimiter::new();
        limiter.update("5:60:10,100:3600:60", "5:60:0,7:3600:0", now).unwrap();
        assert!(!limiter.try_acquire(now));
        assert_eq!(limiter.state(now), "7/5 per 60s, 7/100 per 3600s");
    }

    #[test]
    fn remembers_requests_for_the_longest_window() {
        let start = Instant::now();
        let mut limiter = RateLimiter::new();
        limiter.update("1:3600:60", "0:3600:0", start).unwrap();
        assert!(limiter.try_acquire(start));
        assert!(!limiter.try_acquire(start + Duration::from_secs(1800)));
    }

    #[test]
    fn rejects_mismatched_headers_and_keeps_policies() {
        let now = Instant::now();
        let mut limiter = RateLimiter::new();
        assert!(limiter.update("1:60:0,2:60:0", "1:60:0", now).is_err());
        assert_eq!(limiter.policies.len(), 3);
    }
}
