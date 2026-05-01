package domain

type CourseModule struct {
	ID          string        `json:"id"`
	CourseID    string        `json:"course_id"`
	Position    int           `json:"position"`
	Title       MultiLangText `json:"title"`
	Description MultiLangText `json:"description"`
}

type CreateCourseModuleParams struct {
	CourseID    string        `json:"course_id"`
	Position    int           `json:"position"`
	Title       MultiLangText `json:"title"`
	Description MultiLangText `json:"description"`
}

type UpdateCourseModuleParams struct {
	ID          string        `json:"id"`
	CourseID    string        `json:"course_id"`
	Position    int           `json:"position"`
	Title       MultiLangText `json:"title"`
	Description MultiLangText `json:"description"`
}

type CourseModuleListFilter struct {
	CourseID string
}
