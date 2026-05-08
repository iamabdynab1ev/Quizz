package postgres

// Quiz operations are now handled by CourseRepository.
// Courses embed quiz settings (quiz_pass_percent, quiz_minutes, max_attempts,
// retake_cooldown_days) and own their questions directly.
