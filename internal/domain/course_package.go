package domain

type CoursePackage struct {
	Course     Course     `json:"course"`
	Quiz       Quiz       `json:"quiz"`
	CourseTest CourseTest `json:"course_test"`
}

type CreateCoursePackageParams struct {
	Course       CreateCourseParams `json:"course"`
	Quiz         CreateQuizParams   `json:"quiz"`
	LinkPosition int                `json:"link_position"`
}
