package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/config"
	"corporate.admsr.ru/backend/internal/db"
	"corporate.admsr.ru/backend/internal/handlers"
	"corporate.admsr.ru/backend/internal/httpx"
)

func main() {
	config.LoadDotEnv(".env")
	config.LoadDotEnv("backend/.env")
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v (set DB_* in backend/.env — see .env.example)", err)
	}

	authSvc := &auth.Service{Pool: pool, TTLHours: cfg.SessionTTL}
	authH := &handlers.AuthHandlers{Pool: pool, Auth: authSvc, Config: cfg}
	newsH := &handlers.News{Pool: pool, Auth: authSvc}
	eventsH := &handlers.Events{Pool: pool, Auth: authSvc}
	galleryH := &handlers.Gallery{Pool: pool, Auth: authSvc, UploadDir: cfg.UploadDir}
	galleryBaseH := &handlers.GalleryBase{Pool: pool, Auth: authSvc, UploadDir: cfg.UploadDir}
	uploadH := &handlers.Upload{UploadDir: cfg.UploadDir}
	usersH := &handlers.Users{Pool: pool, Auth: authSvc}
	profileH := &handlers.Profile{Pool: pool, Auth: authSvc}
	feedbackH := &handlers.Feedback{Pool: pool, Auth: authSvc}
	ofoH := &handlers.OFO{Pool: pool, Auth: authSvc}
	absenceH := &handlers.Absence{Pool: pool, Auth: authSvc}
	portalH := &handlers.Portal{Pool: pool, Auth: authSvc}
	coursesH := &handlers.CoursesHandler{Pool: pool, Auth: authSvc, UploadDir: cfg.UploadDir}
	testsH := &handlers.TestsHandler{Pool: pool, Auth: authSvc}
	formsH := &handlers.FormsHandler{Pool: pool, Auth: authSvc}
	syncH := &handlers.Sync{Pool: pool, Config: cfg}
	birthdaysH := &handlers.Birthdays{Pool: pool, Auth: authSvc, UploadDir: cfg.UploadDir}
	healthH := &handlers.Health{Pool: pool}

	mux := http.NewServeMux()
	mux.Handle("/api/health.php", healthH)
	mux.HandleFunc("/api/auth.php", authH.Login)
	mux.HandleFunc("/api/logout.php", authH.Logout)
	mux.HandleFunc("/api/check-auth.php", authH.CheckAuth)
	mux.HandleFunc("/api/heartbeat.php", authH.Heartbeat)
	mux.HandleFunc("/api/session_bootstrap.php", authH.SessionBootstrap)
	mux.Handle("/api/news.php", newsH)
	mux.Handle("/api/events.php", eventsH)
	mux.Handle("/api/gallery.php", galleryH)
	mux.Handle("/api/gallery_base.php", galleryBaseH)
	mux.Handle("/api/Upload/upload.php", uploadH)
	mux.Handle("/api/users.php", usersH)
	mux.Handle("/api/profile.php", profileH)
	mux.Handle("/api/feedback.php", feedbackH)
	mux.HandleFunc("/api/ofo.php", ofoH.List)
	mux.HandleFunc("/api/ofo_seats.php", ofoH.Seats)
	mux.HandleFunc("/api/ofo_tree.php", ofoH.Tree)
	mux.HandleFunc("/api/ofo_positions.php", ofoH.Positions)
	mux.Handle("/api/absence_journal.php", absenceH)
	mux.HandleFunc("/api/portal_groups.php", portalH.Groups)
	mux.HandleFunc("/api/portal_my_permissions.php", portalH.MyPermissions)
	mux.HandleFunc("/api/portal_services.php", portalH.Services)
	mux.Handle("/api/sync.php", syncH)
	mux.Handle("/api/birthdays.php", birthdaysH)

	mux.HandleFunc("/api/tests_list.php", testsH.List)
	mux.HandleFunc("/api/tests_save.php", testsH.Save)
	mux.HandleFunc("/api/tests_publish.php", testsH.Publish)
	mux.HandleFunc("/api/tests_unpublish.php", testsH.Unpublish)
	mux.HandleFunc("/api/tests_delete.php", testsH.Delete)
	mux.HandleFunc("/api/tests_direct.php", testsH.Direct)
	mux.HandleFunc("/api/tests_submit.php", testsH.Submit)
	mux.HandleFunc("/api/tests_stats.php", testsH.Stats)
	mux.HandleFunc("/api/tests_participant.php", testsH.Participant)
	mux.HandleFunc("/api/tests_by_token.php", testsH.ByToken)

	mux.HandleFunc("/api/forms.php", formsH.Forms)
	mux.HandleFunc("/api/forms_list.php", formsH.List)
	mux.HandleFunc("/api/forms_publish.php", formsH.Publish)
	mux.HandleFunc("/api/forms_submit.php", formsH.Submit)
	mux.HandleFunc("/api/forms_report.php", formsH.Report)
	mux.HandleFunc("/api/forms_archive.php", formsH.Archive)
	mux.HandleFunc("/api/forms_delete.php", formsH.Delete)

	mux.HandleFunc("/api/courses_list.php", coursesH.List)
	mux.HandleFunc("/api/courses_get.php", coursesH.Get)
	mux.HandleFunc("/api/courses_create.php", coursesH.Create)
	mux.HandleFunc("/api/courses_update.php", coursesH.Update)
	mux.HandleFunc("/api/courses_delete.php", coursesH.Delete)
	mux.HandleFunc("/api/courses_publish.php", coursesH.Publish)
	mux.HandleFunc("/api/courses_unpublish.php", coursesH.Unpublish)
	mux.HandleFunc("/api/course_topics_create.php", coursesH.TopicsCreate)
	mux.HandleFunc("/api/course_topics_update.php", coursesH.TopicsUpdate)
	mux.HandleFunc("/api/course_topics_delete.php", coursesH.TopicsDelete)
	mux.HandleFunc("/api/course_topics_order.php", coursesH.TopicsOrder)
	mux.HandleFunc("/api/course_materials_create.php", coursesH.MaterialsCreate)
	mux.HandleFunc("/api/course_materials_update.php", coursesH.MaterialsUpdate)
	mux.HandleFunc("/api/course_materials_delete.php", coursesH.MaterialsDelete)
	mux.HandleFunc("/api/course_materials_upload.php", coursesH.MaterialsUpload)
	mux.HandleFunc("/api/course_tests_create.php", coursesH.TestsCreate)
	mux.HandleFunc("/api/course_tests_get.php", coursesH.TestsGet)
	mux.HandleFunc("/api/course_tests_update.php", coursesH.TestsUpdate)
	mux.HandleFunc("/api/course_tests_delete.php", coursesH.TestsDelete)
	mux.HandleFunc("/api/course_assign_preview.php", coursesH.AssignPreview)
	mux.HandleFunc("/api/course_assign.php", coursesH.Assign)
	mux.HandleFunc("/api/course_admin_results.php", coursesH.AdminResults)
	mux.HandleFunc("/api/course_admin_participant.php", coursesH.AdminParticipant)
	mux.HandleFunc("/api/course_admin_attempt.php", coursesH.AdminAttemptAnswers)
	mux.HandleFunc("/api/course_enrollment_reset.php", coursesH.EnrollmentReset)
	mux.HandleFunc("/api/courses_for_me.php", coursesH.ForMe)
	mux.HandleFunc("/api/course_enrollment_get.php", coursesH.EnrollmentGet)
	mux.HandleFunc("/api/course_start.php", coursesH.Start)
	mux.HandleFunc("/api/course_topic_get.php", coursesH.TopicGet)
	mux.HandleFunc("/api/course_material_open.php", coursesH.MaterialOpen)
	mux.HandleFunc("/api/course_material_heartbeat.php", coursesH.MaterialHeartbeat)
	mux.HandleFunc("/api/course_material_complete.php", coursesH.MaterialComplete)
	mux.HandleFunc("/api/course_next_action.php", coursesH.NextAction)
	mux.HandleFunc("/api/course_result.php", coursesH.Result)

	mux.HandleFunc("/api/tests_attempt_start.php", testsH.AttemptStart)
	mux.HandleFunc("/api/tests_attempt_save.php", testsH.AttemptSave)
	mux.HandleFunc("/api/tests_attempt_get.php", testsH.AttemptGet)
	mux.HandleFunc("/api/tests_attempt_finish.php", testsH.AttemptFinish)

	handler := httpx.CORS(mux)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
	log.Printf("go api listening on %s (upload_dir=%s)", cfg.HTTPAddr, cfg.UploadDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	if pool != nil {
		pool.Close()
	}
}
