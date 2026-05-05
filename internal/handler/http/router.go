package http

import (
	"log/slog"
	nethttp "net/http"
	"strings"

	"lms-arvand-backend/internal/domain"
	"lms-arvand-backend/internal/handler/http/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	logger *slog.Logger,
	healthHandler *HealthHandler,
	authHandler *AuthHandler,
	authenticator middleware.Authenticator,
	usersHandler *UsersHandler,
	coursesHandler *CoursesHandler,
	quizzesHandler *QuizzesHandler,
	attemptsHandler *AttemptsHandler,
	enrollmentsHandler *EnrollmentsHandler,
	certificatesHandler *CertificatesHandler,
	courseTestsHandler *CourseTestsHandler,
	courseModulesHandler *CourseModulesHandler,
	contentBlocksHandler *ContentBlocksHandler,
	reviewsHandler *ReviewsHandler,
	notificationsHandler *NotificationsHandler,
	webhooksHandler *WebhooksHandler,
	auditLogsHandler *AuditLogsHandler,
	uploadsHandler *UploadsHandler,
	uploadsDir string,
) nethttp.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.RequestLogger(logger))

	if uploadsRoot := strings.TrimSpace(uploadsDir); uploadsRoot != "" {
		router.Handle("/uploads/*", nethttp.StripPrefix("/uploads/", nethttp.FileServer(nethttp.Dir(uploadsRoot))))
	}

	router.Get("/health", healthHandler.GetHealth)

	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthHandler.GetHealth)

		if authHandler != nil {
			api.Route("/auth", func(auth chi.Router) {
				auth.Post("/login", authHandler.Login)
				auth.Get("/google/config", authHandler.GoogleConfig)
				auth.Post("/google", authHandler.LoginWithGoogle)
			})
		}

		if certificatesHandler != nil {
			api.Get("/certificates/verify/{verifyHash}", certificatesHandler.VerifyCertificate)
		}

		if authenticator == nil {
			return
		}

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(logger, authenticator))

			if authHandler != nil {
				protected.Get("/auth/me", authHandler.Me)
				protected.Post("/auth/logout", authHandler.Logout)
			}

			if coursesHandler != nil {
				protected.Route("/courses", func(courses chi.Router) {
					courses.Get("/", coursesHandler.ListCourses)
					courses.Get("/{courseID}", coursesHandler.GetCourseByID)
					courses.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", coursesHandler.CreateCourse)
					courses.With(middleware.RequireRoles(domain.UserRoleAdmin)).Put("/{courseID}", coursesHandler.UpdateCourse)
					courses.With(middleware.RequireRoles(domain.UserRoleAdmin)).Delete("/{courseID}", coursesHandler.ArchiveCourse)
				})
			}

			if quizzesHandler != nil {
				protected.Route("/quizzes", func(quizzes chi.Router) {
					quizzes.Get("/", quizzesHandler.ListQuizzes)
					quizzes.Get("/{quizID}", quizzesHandler.GetQuizByID)
					quizzes.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", quizzesHandler.CreateQuiz)
					quizzes.With(middleware.RequireRoles(domain.UserRoleAdmin)).Put("/{quizID}", quizzesHandler.UpdateQuiz)
					quizzes.With(middleware.RequireRoles(domain.UserRoleAdmin)).Delete("/{quizID}", quizzesHandler.ArchiveQuiz)

					if attemptsHandler != nil {
						quizzes.Post("/{quizID}/attempts", attemptsHandler.SubmitAttempt)
					}
				})
			}

			if attemptsHandler != nil {
				protected.Route("/attempts", func(attempts chi.Router) {
					attempts.Get("/", attemptsHandler.ListAttempts)
					attempts.Get("/{attemptID}", attemptsHandler.GetAttemptByID)
					attempts.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/{attemptID}/review", attemptsHandler.ReviewAttempt)
				})
			}

			if enrollmentsHandler != nil {
				protected.Route("/enrollments", func(enrollments chi.Router) {
					enrollments.Get("/", enrollmentsHandler.ListEnrollments)
					enrollments.Post("/", enrollmentsHandler.CreateEnrollment)
					enrollments.Get("/{enrollmentID}", enrollmentsHandler.GetEnrollmentByID)
					enrollments.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/{enrollmentID}/complete", enrollmentsHandler.CompleteEnrollment)
				})
			}

			if certificatesHandler != nil {
				protected.Route("/certificates", func(certificates chi.Router) {
					certificates.Get("/", certificatesHandler.ListCertificates)
					certificates.Get("/{certificateID}", certificatesHandler.GetCertificateByID)
					certificates.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", certificatesHandler.CreateCertificate)
				})
			}

			if courseModulesHandler != nil {
				protected.Route("/course-modules", func(modules chi.Router) {
					modules.Get("/", courseModulesHandler.ListCourseModules)
					modules.Get("/{moduleID}", courseModulesHandler.GetCourseModuleByID)
					modules.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", courseModulesHandler.CreateCourseModule)
					modules.With(middleware.RequireRoles(domain.UserRoleAdmin)).Put("/{moduleID}", courseModulesHandler.UpdateCourseModule)
					modules.With(middleware.RequireRoles(domain.UserRoleAdmin)).Delete("/{moduleID}", courseModulesHandler.DeleteCourseModule)
				})
			}

			if contentBlocksHandler != nil {
				protected.Route("/content-blocks", func(blocks chi.Router) {
					blocks.Get("/", contentBlocksHandler.ListContentBlocks)
					blocks.Get("/{blockID}", contentBlocksHandler.GetContentBlockByID)
					blocks.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", contentBlocksHandler.CreateContentBlock)
					blocks.With(middleware.RequireRoles(domain.UserRoleAdmin)).Put("/{blockID}", contentBlocksHandler.UpdateContentBlock)
					blocks.With(middleware.RequireRoles(domain.UserRoleAdmin)).Delete("/{blockID}", contentBlocksHandler.DeleteContentBlock)
				})
			}

			if reviewsHandler != nil {
				protected.Route("/reviews", func(reviews chi.Router) {
					reviews.Get("/", reviewsHandler.ListReviews)
					reviews.Post("/", reviewsHandler.CreateReview)
					reviews.Get("/{reviewID}", reviewsHandler.GetReviewByID)
					reviews.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/{reviewID}/moderate", reviewsHandler.ModerateReview)
				})
			}

			if notificationsHandler != nil {
				protected.Route("/notifications", func(notifications chi.Router) {
					notifications.Get("/", notificationsHandler.ListNotifications)
					notifications.Get("/{notificationID}", notificationsHandler.GetNotificationByID)
					notifications.Post("/{notificationID}/read", notificationsHandler.MarkRead)
					notifications.With(middleware.RequireRoles(domain.UserRoleAdmin)).Post("/", notificationsHandler.CreateNotification)
				})
			}

			protected.Group(func(admin chi.Router) {
				admin.Use(middleware.RequireRoles(domain.UserRoleAdmin))

				if usersHandler != nil {
					admin.Route("/users", func(users chi.Router) {
						users.Get("/", usersHandler.ListUsers)
						users.Post("/", usersHandler.CreateUser)
						users.Get("/{userID}", usersHandler.GetUserByID)
						users.Put("/{userID}", usersHandler.UpdateUser)
						users.Delete("/{userID}", usersHandler.DeactivateUser)
					})
				}

				if courseTestsHandler != nil {
					admin.Route("/course-tests", func(courseTests chi.Router) {
						courseTests.Get("/", courseTestsHandler.ListCourseTests)
						courseTests.Post("/", courseTestsHandler.CreateCourseTest)
						courseTests.Delete("/", courseTestsHandler.DeleteCourseTest)
						courseTests.Delete("/{courseTestID}", courseTestsHandler.DeleteCourseTestByID)
					})
				}

				if webhooksHandler != nil {
					admin.Route("/webhooks", func(webhooks chi.Router) {
						webhooks.Get("/", webhooksHandler.ListWebhooks)
						webhooks.Post("/", webhooksHandler.CreateWebhook)
						webhooks.Get("/{webhookID}", webhooksHandler.GetWebhookByID)
						webhooks.Put("/{webhookID}", webhooksHandler.UpdateWebhook)
						webhooks.Delete("/{webhookID}", webhooksHandler.DeleteWebhook)
					})
				}

				if uploadsHandler != nil {
					admin.Post("/uploads", uploadsHandler.CreateUpload)
					admin.Post("/uploads/", uploadsHandler.CreateUpload)
				}

				if auditLogsHandler != nil {
					admin.Route("/audit-logs", func(auditLogs chi.Router) {
						auditLogs.Get("/", auditLogsHandler.ListAuditLogs)
						auditLogs.Get("/{auditLogID}", auditLogsHandler.GetAuditLogByID)
					})
				}
			})
		})
	})

	return router
}
