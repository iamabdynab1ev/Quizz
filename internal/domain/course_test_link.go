package domain

type CourseTest struct {
	ID       string  `json:"id,omitempty"`
	CourseID *string `json:"course_id,omitempty"`
	ModuleID *string `json:"module_id,omitempty"`
	QuizID   string  `json:"quiz_id"`
	Position int     `json:"position"`
}

type CreateCourseTestParams struct {
	CourseID *string `json:"course_id,omitempty"`
	ModuleID *string `json:"module_id,omitempty"`
	QuizID   string  `json:"quiz_id"`
	Position int     `json:"position"`
}

type CourseTestListFilter struct {
	CourseID *string
	ModuleID *string
}
