package domain

type StudentDashboardStats struct {
	ActiveEnrollments    int     `json:"active_enrollments"`
	CompletedEnrollments int     `json:"completed_enrollments"`
	AttemptsTotal        int     `json:"attempts_total"`
	PassedAttempts       int     `json:"passed_attempts"`
	CertificatesTotal    int     `json:"certificates_total"`
	UnreadNotifications  int     `json:"unread_notifications"`
	AverageScorePercent  float64 `json:"average_score_percent"`
}

type StudentDashboard struct {
	User              User                  `json:"user"`
	Stats             StudentDashboardStats `json:"stats"`
	RecentEnrollments []Enrollment          `json:"recent_enrollments"`
	RecentAttempts    []Attempt             `json:"recent_attempts"`
	Certificates      []Certificate         `json:"certificates"`
}

type AdminDashboardStats struct {
	UsersTotal           int `json:"users_total"`
	ActiveUsers          int `json:"active_users"`
	StudentsTotal        int `json:"students_total"`
	CoursesTotal         int `json:"courses_total"`
	PublishedCourses     int `json:"published_courses"`
	QuizzesTotal         int `json:"quizzes_total"`
	PublishedQuizzes     int `json:"published_quizzes"`
	EnrollmentsTotal     int `json:"enrollments_total"`
	CompletedEnrollments int `json:"completed_enrollments"`
	AttemptsTotal        int `json:"attempts_total"`
	PassedAttempts       int `json:"passed_attempts"`
	AttemptsNeedReview   int `json:"attempts_need_review"`
	CertificatesTotal    int `json:"certificates_total"`
	PendingReviews       int `json:"pending_reviews"`
}

type AdminDashboard struct {
	Stats          AdminDashboardStats `json:"stats"`
	RecentUsers    []User              `json:"recent_users"`
	RecentAttempts []Attempt           `json:"recent_attempts"`
}
