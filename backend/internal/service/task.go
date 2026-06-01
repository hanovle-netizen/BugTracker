package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Задачи
func (s *Service) GetProjectTasks(ctx context.Context, projectID, userID int) ([]map[string]interface{}, error) {
	globalRole, _ := ctx.Value("role").(string)

	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTasksByProject(ctx, projectID)
}

type CreateTaskRequest struct {
	Title               string  `json:"title"`
	Description         *string `json:"description"`
	ProjectID           int     `json:"project_id"`
	Status              *string `json:"status"`
	Severity            *string `json:"severity"`
	Priority            *string `json:"priority"`
	OS                  *string `json:"os"`
	VersionProduct      *string `json:"version_product"`
	PlaybackDescription *string `json:"playback_description"`
	ExpectedResult      *string `json:"expected_result"`
	ActualResult        *string `json:"actual_result"`
}

func (s *Service) CreateTask(ctx context.Context, userID int, req CreateTaskRequest) (int, error) {
	if req.Title == "" || req.ProjectID == 0 {
		return 0, errors.New("title_and_project_id_required")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, req.ProjectID, userID)
		if err != nil {
			return 0, errors.New("access_denied")
		}
	}

	taskMap := map[string]interface{}{
		"title": req.Title, "description": req.Description, "project_id": req.ProjectID,
		"status": req.Status, "severity": req.Severity, "priority": req.Priority,
		"os": req.OS, "version_product": req.VersionProduct,
		"playback_description": req.PlaybackDescription,
		"expected_result":      req.ExpectedResult, "actual_result": req.ActualResult,
	}

	return s.store.CreateTask(ctx, userID, taskMap)
}

func (s *Service) UpdateTask(ctx context.Context, taskID, userID int, req map[string]interface{}) error {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return errors.New("access_denied")
		}
	}

	// Accept frontend API contract fields and map them to DB columns.
	if v, ok := req["assigned_to"]; ok {
		req["assigned_to_fk"] = v
		delete(req, "assigned_to")
	}
	if v, ok := req["passed_by"]; ok {
		req["passed_by_fk"] = v
		delete(req, "passed_by")
	}
	if v, ok := req["accepted_by"]; ok {
		req["accepted_by_fk"] = v
		delete(req, "accepted_by")
	}

	// These are API-level fields; they should not be used as DB columns directly.
	delete(req, "id_pk")
	delete(req, "project_id")
	delete(req, "owner_id_fk")
	delete(req, "created_by")
	delete(req, "task_id")
	delete(req, "assigned_to_email")

	return s.store.UpdateTask(ctx, taskID, userID, req)
}

func (s *Service) DeleteTask(ctx context.Context, taskID, userID int) error {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return errors.New("access_denied")
		}
	}

	return s.store.DeleteTask(ctx, taskID)
}

func (s *Service) UploadTaskPhoto(ctx context.Context, taskID, userID int, fileName string, fileSize int64, data []byte) error {
	if fileSize > 15*1024*1024 {
		return errors.New("file_too_large")
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	valid := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !valid[ext] {
		return errors.New("invalid_format")
	}

	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return errors.New("access_denied")
		}
	}

	return s.store.UpdateTaskPhoto(ctx, taskID, data)
}

func (s *Service) GetTaskPhoto(ctx context.Context, taskID, userID int) ([]byte, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTaskPhoto(ctx, taskID)
}

func (s *Service) GetComments(ctx context.Context, taskID, userID int) ([]map[string]interface{}, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTaskComments(ctx, taskID)
}

func (s *Service) AddComment(ctx context.Context, taskID, userID int, body string) (int, error) {
	if body == "" {
		return 0, errors.New("comment_body_required")
	}

	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return 0, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return 0, errors.New("access_denied")
		}
	}

	return s.store.CreateComment(ctx, taskID, userID, body)
}

func (s *Service) GetTaskRelations(ctx context.Context, taskID, userID int) ([]map[string]interface{}, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTaskRelations(ctx, taskID)
}

func (s *Service) AddTaskRelation(ctx context.Context, taskID, userID int, req struct {
	RelatedTaskID int    `json:"related_task_id"`
	Type          string `json:"type"`
}) (int, error) {
	if taskID == req.RelatedTaskID {
		return 0, errors.New("cannot_relate_to_self")
	}

	projectA, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return 0, errors.New("task_not_found")
	}

	projectB, err := s.store.GetTaskProjectID(ctx, req.RelatedTaskID)
	if err != nil {
		return 0, errors.New("related_task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		if _, err := s.store.GetUserRoleInProject(ctx, projectA, userID); err != nil {
			return 0, errors.New("access_denied")
		}
		if projectA != projectB {
			if _, err := s.store.GetUserRoleInProject(ctx, projectB, userID); err != nil {
				return 0, errors.New("access_denied_to_related_project")
			}
		}
	}

	return s.store.CreateTaskRelation(ctx, taskID, req.RelatedTaskID, req.Type)
}

func (s *Service) DeleteRelation(ctx context.Context, relID, userID int) error {
	globalRole, _ := ctx.Value("role").(string)

	taskA, taskB, err := s.store.DeleteRelation(ctx, relID)
	if err != nil {
		return err
	}

	if globalRole != "admin" {
		projectA, _ := s.store.GetTaskProjectID(ctx, taskA)
		_, errA := s.store.GetUserRoleInProject(ctx, projectA, userID)

		projectB, _ := s.store.GetTaskProjectID(ctx, taskB)
		_, errB := s.store.GetUserRoleInProject(ctx, projectB, userID)

		if errA != nil && errB != nil {
			return errors.New("access_denied")
		}
	}

	return nil
}

func (s *Service) GetTaskTags(ctx context.Context, taskID, userID int) ([]string, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	return s.store.GetTaskTags(ctx, taskID)
}

func (s *Service) ReplaceTags(ctx context.Context, taskID, userID int, tags []string) error {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return errors.New("access_denied")
		}
	}

	return s.store.ReplaceTaskTags(ctx, taskID, tags)
}

type TemplateResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) GetTemplates(ctx context.Context) ([]TemplateResponse, error) {
	data, err := s.store.GetAllTemplates(ctx)
	if err != nil {
		return nil, err
	}

	res := []TemplateResponse{}
	for _, t := range data {
		res = append(res, TemplateResponse{
			ID:        t["id"].(int),
			Name:      t["name"].(string),
			Body:      t["body"].(string),
			CreatedBy: t["created_by"].(int),
			CreatedAt: t["created_at"].(time.Time),
		})
	}
	return res, nil
}

func (s *Service) CreateTemplate(ctx context.Context, userID int, name, body string) (int, error) {
	if name == "" || body == "" {
		return 0, errors.New("name_and_body_required")
	}

	return s.store.CreateTemplate(ctx, userID, name, body)
}

func (s *Service) DeleteTemplate(ctx context.Context, id int) error {
	return s.store.DeleteTemplate(ctx, id)
}

func (s *Service) GetGlobalStats(ctx context.Context, userID int) ([]StatResponse, error) {
	globalRole, _ := ctx.Value("role").(string)
	isTeacher := globalRole == "admin"

	data, err := s.store.GetTaskStats(ctx, userID, isTeacher)
	if err != nil {
		return nil, err
	}

	var res []StatResponse
	for _, d := range data {

		status, _ := d["status"].(string)

		var count int
		if val, ok := d["count"].(int); ok {
			count = val
		} else if val, ok := d["count"].(int64); ok {
			count = int(val)
		}

		res = append(res, StatResponse{
			Status: status,
			Count:  count,
		})
	}
	return res, nil
}

func (s *Service) GetTaskAnalytics(ctx context.Context, taskID, userID int) ([]StatResponse, error) {
	projectID, err := s.store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task_not_found")
	}

	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		_, err := s.store.GetUserRoleInProject(ctx, projectID, userID)
		if err != nil {
			return nil, errors.New("access_denied")
		}
	}

	data, err := s.store.GetSingleTaskStats(ctx, taskID)
	if err != nil {
		return nil, err
	}

	var res []StatResponse
	for _, d := range data {
		res = append(res, StatResponse{
			Status: d["status"].(string),
			Count:  d["count"].(int),
		})
	}
	return res, nil
}

func buildObjectKey(entityType string, entityID int, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	return fmt.Sprintf("%s/%d/%d%s", entityType, entityID, time.Now().UnixNano(), ext)
}

func publicURLForObjectKey(objectKey string) string {
	if useMinioStorage() {
		publicURL := strings.TrimRight(os.Getenv("MINIO_PUBLIC_URL"), "/")
		bucket := os.Getenv("MINIO_BUCKET")
		if bucket == "" {
			bucket = "tasktracker-photos"
		}
		if publicURL != "" {
			return fmt.Sprintf("%s/%s/%s", publicURL, bucket, strings.TrimPrefix(path.Clean("/"+objectKey), "/"))
		}
	}
	return "/uploads/" + strings.TrimPrefix(path.Clean("/"+objectKey), "/")
}

func useMinioStorage() bool {
	return strings.EqualFold(os.Getenv("PHOTO_STORAGE"), "minio")
}

func minioConfig() (endpoint, accessKey, secretKey, bucket, publicURL string) {
	endpoint = os.Getenv("MINIO_ENDPOINT")
	accessKey = os.Getenv("MINIO_ACCESS_KEY")
	secretKey = os.Getenv("MINIO_SECRET_KEY")
	bucket = os.Getenv("MINIO_BUCKET")
	publicURL = os.Getenv("MINIO_PUBLIC_URL")
	if bucket == "" {
		bucket = "tasktracker-photos"
	}
	return
}

func minioClient() (*minio.Client, string, string, error) {
	endpoint, accessKey, secretKey, bucket, publicURL := minioConfig()
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, "", "", errors.New("minio_config_missing")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true"),
	})
	if err != nil {
		return nil, "", "", err
	}
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, "", "", err
	}
	if !exists {
		if err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, "", "", err
		}
	}
	// Frontend expects returned photo URLs to be directly openable in browser.
	// Keep bucket objects publicly readable; this matches API docs ("presigned or public URL").
	policy := fmt.Sprintf(`{
		"Version":"2012-10-17",
		"Statement":[
			{
				"Effect":"Allow",
				"Principal":{"AWS":["*"]},
				"Action":["s3:GetObject"],
				"Resource":["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucket)
	if err := client.SetBucketPolicy(context.Background(), bucket, policy); err != nil {
		return nil, "", "", err
	}
	return client, bucket, publicURL, nil
}

func saveObjectFile(objectKey, fileName string, data []byte) error {
	if useMinioStorage() {
		client, bucket, _, err := minioClient()
		if err != nil {
			return err
		}
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		_, err = client.PutObject(context.Background(), bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
		return err
	}
	fullPath := filepath.Join("uploads", filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0o644)
}

func deleteObjectFile(objectKey string) {
	if useMinioStorage() {
		client, bucket, _, err := minioClient()
		if err != nil {
			return
		}
		_ = client.RemoveObject(context.Background(), bucket, objectKey, minio.RemoveObjectOptions{})
		return
	}
	_ = os.Remove(filepath.Join("uploads", filepath.FromSlash(objectKey)))
}

func checkPhotoAccess(ctx context.Context, taskID, userID int, store interface {
	GetTaskProjectID(context.Context, int) (int, error)
	GetUserRoleInProject(context.Context, int, int) (string, error)
}) error {
	projectID, err := store.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return errors.New("task_not_found")
	}
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		if _, err := store.GetUserRoleInProject(ctx, projectID, userID); err != nil {
			return errors.New("access_denied")
		}
	}
	return nil
}

func checkEntityAccess(ctx context.Context, entityType string, entityID, userID int, store interface {
	GetTaskProjectID(context.Context, int) (int, error)
	GetUserRoleInProject(context.Context, int, int) (string, error)
	GetBugTaskID(context.Context, int) (int, error)
}) error {
	if entityType == "bug" {
		taskID, err := store.GetBugTaskID(ctx, entityID)
		if err != nil {
			if err.Error() == "bug_not_found" {
				return errors.New("bug_not_found")
			}
			return err
		}
		entityID = taskID
	}
	return checkPhotoAccess(ctx, entityID, userID, store)
}

func (s *Service) GetEntityPhotos(ctx context.Context, entityType string, entityID, userID int) ([]map[string]interface{}, error) {
	if err := checkEntityAccess(ctx, entityType, entityID, userID, s.store); err != nil {
		return nil, err
	}
	return s.store.GetPhotos(ctx, entityType, entityID)
}

func (s *Service) AddEntityPhoto(ctx context.Context, entityType string, entityID, userID int, fileName string, fileSize int64, data []byte) (int, string, error) {
	if err := checkEntityAccess(ctx, entityType, entityID, userID, s.store); err != nil {
		return 0, "", err
	}
	if fileSize > 15*1024*1024 {
		return 0, "", errors.New("file_too_large")
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	valid := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !valid[ext] {
		return 0, "", errors.New("invalid_format")
	}
	objectKey := buildObjectKey(entityType, entityID, fileName)
	if err := saveObjectFile(objectKey, fileName, data); err != nil {
		return 0, "", err
	}
	url := publicURLForObjectKey(objectKey)
	id, err := s.store.AddPhoto(ctx, entityType, entityID, objectKey, url, userID)
	if err != nil {
		deleteObjectFile(objectKey)
		return 0, "", err
	}
	return id, url, nil
}

func (s *Service) DeleteEntityPhoto(ctx context.Context, entityType string, entityID, photoID, userID int) error {
	if err := checkEntityAccess(ctx, entityType, entityID, userID, s.store); err != nil {
		return err
	}
	photo, err := s.store.GetPhotoByID(ctx, photoID)
	if err != nil {
		return err
	}
	if photo["entity_type"].(string) != entityType || photo["entity_id"].(int) != entityID {
		return errors.New("photo_not_found")
	}
	globalRole, _ := ctx.Value("role").(string)
	if globalRole != "admin" {
		if uploader, ok := photo["uploaded_by"].(*int); !ok || uploader == nil || *uploader != userID {
			return errors.New("access_denied")
		}
	}
	if err := s.store.DeletePhoto(ctx, photoID); err != nil {
		return err
	}
	deleteObjectFile(photo["object_key"].(string))
	return nil
}

