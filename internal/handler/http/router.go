package http

import (
	"log/slog"
	nethttp "net/http"
	"strings"

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
	dashboardHandler *DashboardHandler,
	coursePackagesHandler *CoursePackagesHandler,
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
	router.Use(chimiddleware.Compress(5, "application/json", "text/plain", "text/html"))
	router.Use(chimiddleware.Throttle(1000))
	router.Use(middleware.RateLimit(middleware.NewIPRateLimiter(100, 200)))
	router.Use(middleware.RequestLogger(logger))

	if uploadsRoot := strings.TrimSpace(uploadsDir); uploadsRoot != "" {
		router.Handle("/uploads/*", nethttp.StripPrefix("/uploads/", nethttp.FileServer(nethttp.Dir(uploadsRoot))))
	}

	router.Get("/health", healthHandler.GetHealth)

	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthHandler.GetHealth)

		if authHandler != nil {
			api.Route("/auth", func(auth chi.Router) {
				auth.Post("/register", authHandler.Register)
				auth.Post("/login", authHandler.Login)
				auth.Get("/google/config", authHandler.GoogleConfig)
				auth.Post("/google", authHandler.LoginWithGoogle)
				auth.Post("/password/forgot", authHandler.ForgotPassword)
				auth.Post("/password/reset", authHandler.ResetPassword)
			})
		}

		if certificatesHandler != nil {
			api.Get("/certificates/verify/{verifyHash}", certificatesHandler.VerifyCertificate)
			api.Get("/certificates/{certificateID}", certificatesHandler.GetPublicCertificateByID)
			api.Get("/certificate/{certificateID}", certificatesHandler.GetPublicCertificateByID)
		}

		if coursesHandler != nil {
			api.Get("/courses", coursesHandler.ListCourses)
			api.Get("/courses/", coursesHandler.ListCourses)
			api.Get("/courses/{courseID}", coursesHandler.GetCourseByID)
		}

		if quizzesHandler != nil {
			api.Get("/quizzes/{quizID}", quizzesHandler.GetQuizByID)
		}

		if authenticator == nil {
			return
		}

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(logger, authenticator))

			if authHandler != nil {
				protected.Get("/auth/me", authHandler.Me)
				protected.Put("/auth/me", authHandler.UpdateMe)
				protected.Post("/auth/password/change", authHandler.ChangePassword)
				protected.Post("/auth/logout", authHandler.Logout)
			}

			if dashboardHandler != nil {
				protected.Get("/dashboard", dashboardHandler.GetDashboard)
				protected.Get("/dashboard/me", dashboardHandler.GetMyDashboard)
				protected.With(middleware.RequireAdmin()).Get("/dashboard/admin", dashboardHandler.GetAdminDashboard)
			}

			if coursesHandler != nil {
				protected.With(middleware.RequireAdmin()).Post("/courses", coursesHandler.CreateCourse)
				protected.With(middleware.RequireAdmin()).Post("/courses/", coursesHandler.CreateCourse)
				protected.With(middleware.RequireAdmin()).Put("/courses/{courseID}", coursesHandler.UpdateCourse)
				protected.With(middleware.RequireAdmin()).Delete("/courses/{courseID}", coursesHandler.ArchiveCourse)
			}

			if coursePackagesHandler != nil {
				protected.With(middleware.RequireAdmin()).
					Post("/course-packages", coursePackagesHandler.CreateCoursePackage)
			}

			if quizzesHandler != nil {
				protected.Get("/quizzes", quizzesHandler.ListQuizzes)
				protected.Get("/quizzes/", quizzesHandler.ListQuizzes)
				protected.With(middleware.RequireAdmin()).Get("/quizzes/{quizID}/answers", quizzesHandler.GetQuizByIDWithAnswers)
				protected.With(middleware.RequireAdmin()).Post("/quizzes", quizzesHandler.CreateQuiz)
				protected.With(middleware.RequireAdmin()).Post("/quizzes/", quizzesHandler.CreateQuiz)
				protected.With(middleware.RequireAdmin()).Put("/quizzes/{quizID}", quizzesHandler.UpdateQuiz)
				protected.With(middleware.RequireAdmin()).Delete("/quizzes/{quizID}", quizzesHandler.ArchiveQuiz)
			}

			if attemptsHandler != nil && coursesHandler != nil {
				protected.Post("/courses/{courseID}/attempts", attemptsHandler.SubmitAttempt)
			}

			if attemptsHandler != nil {
				protected.Route("/attempts", func(attempts chi.Router) {
					attempts.Get("/", attemptsHandler.ListAttempts)
					attempts.Get("/{attemptID}", attemptsHandler.GetAttemptByID)
				})
			}

			if enrollmentsHandler != nil {
				protected.Route("/enrollments", func(enrollments chi.Router) {
					enrollments.Get("/", enrollmentsHandler.ListEnrollments)
					enrollments.Post("/", enrollmentsHandler.CreateEnrollment)
					enrollments.Get("/{enrollmentID}", enrollmentsHandler.GetEnrollmentByID)
					enrollments.With(middleware.RequireAdmin()).Post("/{enrollmentID}/complete", enrollmentsHandler.CompleteEnrollment)
				})
			}

			if certificatesHandler != nil {
				protected.Get("/certificates", certificatesHandler.ListCertificates)
				protected.Get("/certificates/", certificatesHandler.ListCertificates)
				protected.With(middleware.RequireAdmin()).Post("/certificates", certificatesHandler.CreateCertificate)
				protected.With(middleware.RequireAdmin()).Post("/certificates/", certificatesHandler.CreateCertificate)
			}

			if courseModulesHandler != nil {
				protected.Route("/course-modules", func(modules chi.Router) {
					modules.Get("/", courseModulesHandler.ListCourseModules)
					modules.Get("/{moduleID}", courseModulesHandler.GetCourseModuleByID)
					modules.With(middleware.RequireAdmin()).Post("/", courseModulesHandler.CreateCourseModule)
					modules.With(middleware.RequireAdmin()).Put("/{moduleID}", courseModulesHandler.UpdateCourseModule)
					modules.With(middleware.RequireAdmin()).Delete("/{moduleID}", courseModulesHandler.DeleteCourseModule)
				})
			}

			if contentBlocksHandler != nil {
				protected.Route("/content-blocks", func(blocks chi.Router) {
					blocks.Get("/", contentBlocksHandler.ListContentBlocks)
					blocks.Get("/{blockID}", contentBlocksHandler.GetContentBlockByID)
					blocks.With(middleware.RequireAdmin()).Post("/", contentBlocksHandler.CreateContentBlock)
					blocks.With(middleware.RequireAdmin()).Put("/{blockID}", contentBlocksHandler.UpdateContentBlock)
					blocks.With(middleware.RequireAdmin()).Delete("/{blockID}", contentBlocksHandler.DeleteContentBlock)
				})
			}

			if uploadsHandler != nil {
				protected.Post("/uploads", uploadsHandler.CreateUpload)
				protected.Post("/uploads/", uploadsHandler.CreateUpload)
			}

			if reviewsHandler != nil {
				protected.Route("/reviews", func(reviews chi.Router) {
					reviews.Get("/", reviewsHandler.ListReviews)
					reviews.Post("/", reviewsHandler.CreateReview)
					reviews.Get("/{reviewID}", reviewsHandler.GetReviewByID)
					reviews.With(middleware.RequireAdmin()).Post("/{reviewID}/moderate", reviewsHandler.ModerateReview)
				})
			}

			if notificationsHandler != nil {
				protected.Route("/notifications", func(notifications chi.Router) {
					notifications.Get("/", notificationsHandler.ListNotifications)
					notifications.Get("/{notificationID}", notificationsHandler.GetNotificationByID)
					notifications.Post("/{notificationID}/read", notificationsHandler.MarkRead)
					notifications.With(middleware.RequireAdmin()).Post("/", notificationsHandler.CreateNotification)
				})
			}

			protected.Group(func(admin chi.Router) {
				admin.Use(middleware.RequireAdmin())

				if usersHandler != nil {
					admin.With(middleware.RequireSuperAdmin()).Route("/users", func(users chi.Router) {
						users.Get("/", usersHandler.ListUsers)
						users.Post("/", usersHandler.CreateUser)
						users.Get("/{userID}", usersHandler.GetUserByID)
						users.Put("/{userID}", usersHandler.UpdateUser)
						users.Delete("/{userID}", usersHandler.DeactivateUser)
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
