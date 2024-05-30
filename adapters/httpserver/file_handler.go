package httpserver

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/SeaCloudHub/backend/domain/notification"

	"github.com/SeaCloudHub/backend/pkg/app"
	"github.com/SeaCloudHub/backend/pkg/apperror"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"

	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/domain/permission"

	"github.com/labstack/echo/v4"
)

// GetMetadata godoc
// @Summary GetMetadata
// @Description GetMetadata
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.GetMetadataRequest true "Get metadata request"
// @Success 200 {object} model.SuccessResponse{data=model.GetMetadataResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/metadata [get]
func (s *Server) GetMetadata(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.GetMetadataRequest
		parents []file.SimpleFile
		users   []permission.FileUser
		canView bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	f, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !f.IsDir {
		canView, err = s.PermissionService.CanViewFile(ctx, id.ID, f.ID.String())
	} else {
		canView, err = s.PermissionService.CanViewDirectory(ctx, id.ID, f.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	if parentPaths := f.Parents(); len(parentPaths) > 0 {
		parents, err = s.FileStore.ListByFullPaths(ctx, parentPaths)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	// get who has access to the file
	if f.IsDir {
		users, err = s.PermissionService.GetDirectoryUsers(ctx, f.ID.String())
	} else {
		users, err = s.PermissionService.GetFileUsers(ctx, f.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	userRoles, err := s.PermissionService.GetFileUserRoles(ctx, id.ID, f.ID.String(), f.IsDir)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	isStarred, err := s.FileStore.IsStarred(ctx, f.ID, uuid.MustParse(id.ID))
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	userIDs := lo.Map(users, func(user permission.FileUser, _ int) string {
		return user.UserID
	})

	userDetails, err := s.UserStore.ListByIDs(ctx, userIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	userDetalMap := lo.KeyBy(userDetails, func(userDetail identity.User) string {
		return userDetail.ID.String()
	})

	users = lo.Map(users, func(user permission.FileUser, _ int) permission.FileUser {
		u := userDetalMap[user.UserID]

		user.Email = u.Email
		user.FirstName = u.FirstName
		user.LastName = u.LastName
		user.AvatarURL = u.AvatarURL

		return user
	})

	return s.success(c, model.GetMetadataResponse{
		File:    *f.Response().WithUserRoles(userRoles).WithIsStarred(isStarred),
		Parents: parents,
		Users:   users,
	})
}

// Download godoc
// @Summary Download
// @Description Download
// @Tags file
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.DownloadRequest true "Download file request"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/download [get]
func (s *Server) Download(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.DownloadRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	id, _ := c.Get(ContextKeyIdentity).(*identity.Identity)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canView, err := s.PermissionService.CanViewDirectory(ctx, id.ID, e.ID.String())
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if !canView {
			return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
		}

		return s.downloadZip(c, e.Path, []string{e.ID.String()})
	}

	canView, err := s.PermissionService.CanViewFile(ctx, id.ID, e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	f, mime, err := s.FileService.DownloadFile(ctx, e.ID.String())
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}
	defer f.Close()

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, uuid.MustParse(id.ID), file.LogActionOpen)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return c.Stream(http.StatusOK, mime, f)
}

// DownloadBatch godoc
// @Summary DownloadBatch
// @Description DownloadBatch
// @Tags file
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request body model.DownloadBatchRequest true "Download batch request"
// @Success 200 {file} file
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/download [post]
func (s *Server) DownloadBatch(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.DownloadBatchRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has view permission to the parent directory
	parent, err := s.FileStore.GetByID(ctx, req.ParentID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canView, err := s.PermissionService.CanViewDirectory(ctx, user.ID.String(), parent.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	return s.downloadZip(c, parent.FullPath(), req.IDs)
}

// UploadFiles godoc
// @Summary UploadFiles
// @Description UploadFiles
// @Tags file
// @Accept multipart/form-data
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request formData model.UploadFilesRequest true "Upload files request"
// @Param files formData file true "Files"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files [post]
func (s *Server) UploadFiles(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.UploadFilesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Check if user has permission to upload files
	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// Files
	form, err := c.MultipartForm()
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	files := form.File["files"]

	totalSize := lo.Reduce(files, func(agg uint64, file *multipart.FileHeader, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+e.Owner.StorageUsage > e.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	wp := workerpool.New(10)
	var m sync.Mutex
	var resp []file.File

	for _, file := range files {
		// open file
		src, err := file.Open()
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
		defer src.Close()

		wp.Submit(func() {
			// save files
			f, err := s.createFile(ctx, e, src, file.Filename, user.ID, false)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	newStorageUsage := lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, e.Owner.StorageUsage)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, e.OwnerID, newStorageUsage); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	payload := lo.Map(resp, func(f file.File, index int) map[string]string {
		return map[string]string{"id": f.ID.String(), "mime": f.MimeType}
	})

	message, err := json.Marshal(payload)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.PubSubService.Publish(ctx, "thumbnails", string(message)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	logs := lo.Map(resp, func(f file.File, index int) file.Log {
		return file.NewLog(f.ID, user.ID, file.LogActionCreate)
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// UploadChunk godoc
// @Summary UploadChunk
// @Description UploadChunk
// @Tags file
// @Accept multipart/form-data
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request formData model.UploadChunkRequest true "Upload chunk request"
// @Param file formData file true "File"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/chunks [post]
func (s *Server) UploadChunk(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.UploadChunkRequest
		f   *file.File
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	// Check if user has permission to upload files
	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	mpFile, fileHeader, err := c.Request().FormFile("file")
	if err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}
	defer mpFile.Close()

	// chec storage limit from file size
	if uint64(fileHeader.Size)+e.Owner.StorageUsage > e.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	if req.FileID != "" {
		// append chunk to file
		f, err = s.FileStore.GetUnfinishedByID(ctx, req.FileID)
		if err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.error(c, apperror.ErrEntityNotFound(err))
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}

		f, err = s.appendChunk(ctx, f.ID.String(), mpFile, req.Last)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if req.Last {
			payload := []map[string]string{
				{"id": f.ID.String(), "mime": f.MimeType},
			}

			message, err := json.Marshal(payload)
			if err != nil {
				return s.error(c, apperror.ErrInternalServer(err))
			}

			if err := s.PubSubService.Publish(ctx, "thumbnails", string(message)); err != nil {
				return s.error(c, apperror.ErrInternalServer(err))
			}

			// write log
			if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(f.ID, user.ID, file.LogActionCreate)}); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
			}
		}
	} else {
		// check storage limit from client input
		if req.TotalSize+e.Owner.StorageUsage > e.Owner.StorageCapacity {
			return s.error(c, apperror.ErrStorageCapacityExceeded())
		}

		// create file
		f, err = s.createFile(ctx, e, mpFile, fileHeader.Filename, user.ID, true)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, e.OwnerID, e.Owner.StorageUsage+uint64(fileHeader.Size)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, f.Response())
}

// ListEntries godoc
// @Summary ListEntries
// @Description ListEntries
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param id path string true "Directory ID"
// @Param request query model.ListEntriesRequest true "List entries request"
// @Success 200 {object} model.SuccessResponse{data=model.ListEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id} [get]
func (s *Server) ListEntries(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListEntriesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !e.IsDir {
		return s.error(c, apperror.ErrDirectoryOnlyOperation())
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	canView, err := s.PermissionService.CanViewDirectory(ctx, user.ID.String(), e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After)

	files, err := s.FileStore.ListCursor(ctx, e.FullPath(), cursor, filter)
	if err != nil {
		if errors.Is(err, file.ErrInvalidCursor) {
			return s.error(c, apperror.ErrInvalidParam(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range files {
		files[i] = *files[i].Response()
	}

	// write log
	if !e.IsRoot() {
		if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionOpen)}); err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
		}
	}

	files = s.mapUserRolesAndStarred(ctx, user, files)

	return s.success(c, model.ListEntriesResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
	})
}

// ListPageEntries godoc
// @Summary ListPageEntries
// @Description ListPageEntries
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param id path string true "Directory ID"
// @Param request query model.ListPageEntriesRequest true "List page entries request"
// @Success 200 {object} model.SuccessResponse{data=model.ListPageEntriesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/page [get]
func (s *Server) ListPageEntries(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListPageEntriesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !e.IsDir {
		return s.error(c, apperror.ErrDirectoryOnlyOperation())
	}

	canView, err := s.PermissionService.CanViewDirectory(ctx, user.ID.String(), e.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	pager := pagination.NewPager(req.Page, req.Limit)
	filter := file.NewFilter(req.Type, req.After)
	files, err := s.FileStore.ListPager(ctx, e.FullPath(), pager, filter, req.Query)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	files = s.mapUserRolesAndStarred(ctx, user, files)

	return s.success(c, model.ListPageEntriesResponse{
		Entries:    files,
		Pagination: pager.PageInfo(),
	})
}

// ListTrash godoc
// @Summary ListTrash
// @Description ListTrash
// @Tags file
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.ListTrashRequest true "List trash request"
// @Success 200 {object} model.SuccessResponse{data=model.ListTrashResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/trash [get]
func (s *Server) ListTrash(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListTrashRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// get trash directory
	trash, err := s.FileStore.GetTrashByUserID(ctx, user.ID)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After)
	files, err := s.FileStore.ListTrash(ctx, trash.FullPath(), cursor, filter)
	if err != nil {
		if errors.Is(err, file.ErrInvalidCursor) {
			return s.error(c, apperror.ErrInvalidParam(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i := range files {
		files[i] = *files[i].Response()
	}

	files = s.mapUserRolesAndStarred(ctx, user, files)

	return s.success(c, model.ListTrashResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
	})
}

// CreateDirectory godoc
// @Summary CreateDirectory
// @Description CreateDirectory
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CreateDirectoryRequest true "Create directory request"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/directories [post]
func (s *Server) CreateDirectory(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.CreateDirectoryRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	parent, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), parent.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	f := file.NewDirectory(req.Name).WithID(uuid.New()).WithPath(parent.FullPath()).WithOwnerID(user.ID)
	if err := s.FileStore.Create(ctx, f); err != nil {
		if errors.Is(err, file.ErrDirAlreadyExists) {
			return s.error(c, apperror.ErrDirAlreadyExists(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.PermissionService.CreateDirectoryPermissions(ctx, user.ID.String(), f.ID.String(), parent.ID.String()); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(f.ID, user.ID, file.LogActionCreate)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, f.Response())
}

// Share godoc
// @Summary Share
// @Description Share
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.ShareRequest true "Share request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/share [post]
func (s *Server) Share(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.ShareRequest
		canEdit bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	} else {
		canEdit, err = s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	users, err := s.UserStore.ListByEmails(ctx, req.Emails)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	userIDs := lo.Map(users, func(u identity.User, index int) uuid.UUID {
		return u.ID
	})

	for _, userID := range userIDs {
		if userID == user.ID {
			return s.error(c, apperror.ErrInvalidRequest(errors.New("cannot share with yourself")))
		}
	}

	if err := s.FileStore.UpsertShare(ctx, e.ID, userIDs, req.Role); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	token := *c.Get(ContextKeyIdentity).(*identity.Identity).Session.Token

	go func() {
		notifications := lo.Map(users, func(u identity.User, index int) notification.Notification {
			content := map[string]interface{}{
				"file":         e.Name,
				"file_id":      e.ID.String(),
				"is_dir":       e.IsDir,
				"role":         req.Role,
				"owner_avatar": user.AvatarURL,
				"owner_name":   fmt.Sprint(user.FirstName, " ", user.LastName),
			}

			contentBytes, _ := json.Marshal(content)

			return notification.Notification{
				UserID:  u.ID.String(),
				Content: string(contentBytes),
			}
		})

		// Send all notifications in one batch
		if err := s.NotificationService.SendNotification(context.Background(), notifications, user.ID.String(), token); err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
		}
	}()

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionShare)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, nil)
}

// Access godoc
// @Summary Access
// @Description Access
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request path model.AccessRequest true "Access request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/access [get]
func (s *Server) Access(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.AccessRequest
		err error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	var role string

	switch e.GeneralAccess {
	case "everyone-can-view":
		role = "viewer"
	case "everyone-can-edit":
		role = "editor"

	case "restricted":
		fallthrough
	default:
		share, err := s.FileStore.GetShare(ctx, e.ID, user.ID)
		if err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.success(c, nil)
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}

		role = share.Role
	}

	// clear permissions
	if e.IsDir {
		if err := s.PermissionService.ClearDirectoryPermissions(ctx, e.ID.String(), user.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	} else {
		if err := s.PermissionService.ClearFilePermissions(ctx, e.ID.String(), user.ID.String()); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	// add permissions
	if err := s.PermissionService.CreatePermission(ctx, permission.NewCreatePermission(
		user.ID.String(), e.ID.String(), e.IsDir, role)); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// remove share
	if err := s.FileStore.DeleteShare(ctx, e.ID, user.ID); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, nil)
}

// UpdateGeneralAccess godoc
// @Summary UpdateGeneralAccess
// @Description UpdateGeneralAccess
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.UpdateGeneralAccessRequest true "Update general access request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/general-access [patch]
func (s *Server) UpdateGeneralAccess(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.UpdateGeneralAccessRequest
		canEdit bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	} else {
		canEdit, err = s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	if err := s.FileStore.UpdateGeneralAccess(ctx, e.ID, req.GeneralAccess); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionUpdate)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, nil)
}

// UpdateAccess godoc
// @Summary UpdateAccess
// @Description UpdateAccess
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.UpdateAccessRequest true "Update access request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/access [patch]
func (s *Server) UpdateAccess(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.UpdateAccessRequest
		isOwner bool
		err     error
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		isOwner, err = s.PermissionService.IsDirectoryOwner(ctx, user.ID.String(), e.ID.String())
	} else {
		isOwner, err = s.PermissionService.IsFileOwner(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !isOwner {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	wp := workerpool.New(10)

	for _, a := range req.Access {
		wp.Submit(func() {
			if a.UserID == user.ID.String() {
				return
			}

			// clear permissions
			if e.IsDir {
				if err := s.PermissionService.ClearDirectoryPermissions(ctx, e.ID.String(), a.UserID); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			} else {
				if err := s.PermissionService.ClearFilePermissions(ctx, e.ID.String(), a.UserID); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			// add permissions
			if a.Role == "revoked" {
				return
			}

			if err := s.PermissionService.CreatePermission(ctx, permission.NewCreatePermission(
				a.UserID, e.ID.String(), e.IsDir, a.Role)); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}
		})
	}

	wp.StopWait()

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionUpdate)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, nil)
}

// CopyFiles godoc
// @Summary CopyFiles
// @Description CopyFiles
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.CopyFilesRequest true "Copy files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/copy [post]
func (s *Server) CopyFiles(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.CopyFilesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the destination directory
	dest, err := s.FileStore.GetByID(ctx, req.To)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListByIDs(ctx, req.IDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		// copy directory is not allowed
		if e.IsDir {
			return s.error(c, apperror.ErrFileOnlyOperation())
		}

		// check if user has view permission to the file
		canView, err := s.PermissionService.CanViewFile(ctx, user.ID.String(), e.ID.String())
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if !canView {
			return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
		}

		// copy file
		wp.Submit(func() {
			src, _, err := s.FileService.DownloadFile(ctx, e.ID.String())
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}
			defer src.Close()

			newName := fmt.Sprintf("Copy of %s", e.Name)

			f, err := s.createFile(ctx, dest, src, newName, user.ID, false)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			userRoles, err := s.PermissionService.GetFileUserRoles(ctx, user.ID.String(), f.ID.String(), f.IsDir)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response().WithUserRoles(userRoles))
		})
	}

	wp.StopWait()

	newStorageUsage := lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, dest.Owner.StorageUsage)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, newStorageUsage); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	logs := lo.Map(files, func(f file.File, index int) file.Log {
		return file.NewLog(f.ID, user.ID, file.LogActionCreate)
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// Move godoc
// @Summary Move
// @Description Move
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.MoveRequest true "Move files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/move [post]
func (s *Server) Move(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.MoveRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the source directory
	src, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// check if user has edit permission to the destination directory
	dest, err := s.FileStore.GetByID(ctx, req.To)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelectedChildren(ctx, src.FullPath(), req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			dstPath := strings.Replace(e.Path, src.FullPath(), dest.FullPath(), 1)

			if e.Path == src.FullPath() {
				// update parent relationship
				if e.IsDir {
					err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				} else {
					err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				}

				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			f := e.WithPath(dstPath)
			if err := s.FileStore.UpdatePath(ctx, e.ID, dstPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	if dest.OwnerID == src.OwnerID {
		return s.success(c, resp)
	}

	totalSize = lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	logs := lo.FilterMap(resp, func(f file.File, index int) (file.Log, bool) {
		if f.Path != src.FullPath() {
			return file.Log{}, false
		}

		return file.NewLog(f.ID, user.ID, file.LogActionMove), true
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// Rename godoc
// @Summary Rename
// @Description Rename
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.RenameFileRequest true "Rename file request"
// @Success 200 {object} model.SuccessResponse{data=file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/rename [patch]
func (s *Server) Rename(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.RenameFileRequest
		canEdit bool
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	e, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	if e.IsDir {
		canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), e.ID.String())
	} else {
		canEdit, err = s.PermissionService.CanEditFile(ctx, user.ID.String(), e.ID.String())
	}

	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	newPath := strings.Replace(e.Path, e.Name, req.Name, 1)

	if err := s.FileStore.UpdateName(ctx, e.ID, req.Name); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	resp := *e.WithName(req.Name).WithPath(newPath).Response()

	// write log
	if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionUpdate)}); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// MoveToTrash godoc
// @Summary MoveToTrash
// @Description MoveToTrash
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.MoveToTrashRequest true "Move to trash request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/move/trash [post]
func (s *Server) MoveToTrash(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.MoveToTrashRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the source directory
	src, err := s.FileStore.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	// get trash directory
	dest, err := s.FileStore.GetTrashByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err = s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelectedOwnedChildren(ctx, user.ID, src, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	totalSize := lo.Reduce(files, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	if totalSize+dest.Owner.StorageUsage > dest.Owner.StorageCapacity {
		return s.error(c, apperror.ErrStorageCapacityExceeded())
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			dstPath := strings.Replace(e.Path, src.FullPath(), dest.FullPath(), 1)

			if e.Path == src.FullPath() {
				// update parent relationship
				if e.IsDir {
					err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				} else {
					err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
				}

				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			f := e.WithPath(dstPath)
			if err := s.FileStore.MoveToTrash(ctx, e.ID, dstPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *f.Response())
		})
	}

	wp.StopWait()

	if dest.OwnerID == src.OwnerID {
		return s.success(c, resp)
	}

	totalSize = lo.Reduce(resp, func(agg uint64, file file.File, index int) uint64 {
		return agg + uint64(file.Size)
	}, 0)

	// update user storage usage
	if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	// write log
	logs := lo.FilterMap(resp, func(f file.File, index int) (file.Log, bool) {
		if f.Path != src.FullPath() {
			return file.Log{}, false
		}

		return file.NewLog(f.ID, user.ID, file.LogActionMove), true
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// RestoreFromTrash godoc
// @Summary RestoreFromTrash
// @Description RestoreFromTrash
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.RestoreFromTrashRequest true "Restore from trash request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/restore [post]
func (s *Server) RestoreFromTrash(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.RestoreFromTrashRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	// check if user has edit permission to the trash directory
	src, err := s.FileStore.GetTrashByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return s.error(c, apperror.ErrEntityNotFound(err))
		}

		return s.error(c, apperror.ErrInternalServer(err))
	}

	canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), src.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canEdit {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToEdit))
	}

	files, err := s.FileStore.ListSelected(ctx, src, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	var resp []file.File

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		if e.PreviousPath == nil || *e.PreviousPath == "" {
			continue
		}

		wp.Submit(func() {
			dest, err := s.FileStore.GetByFullPath(ctx, *e.PreviousPath)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			// check if user has edit permission to the destination directory
			canEdit, err := s.PermissionService.CanEditDirectory(ctx, user.ID.String(), dest.ID.String())
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if !canEdit {
				// if user has no edit permission to the destination directory, restore to root directory
				dest, err = s.FileStore.GetByID(ctx, e.Owner.RootID.String())
				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			dstPath := strings.Replace(e.Path, src.FullPath(), dest.FullPath(), 1)
			path := e.Path

			// update parent relationship
			if e.IsDir {
				err = s.PermissionService.UpdateDirectoryParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
			} else {
				err = s.PermissionService.UpdateFileParent(ctx, e.ID.String(), dest.ID.String(), src.ID.String())
			}

			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			f := e.WithPath(dstPath)
			if err := s.FileStore.RestoreFromTrash(ctx, e.ID, dstPath); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			totalSize := e.Size

			if e.IsDir {
				children, err := s.FileStore.RestoreChildrenFromTrash(ctx, path, dstPath)
				if err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				totalSize = lo.Reduce(children, func(agg uint64, file file.File, index int) uint64 {
					return agg + uint64(file.Size)
				}, totalSize)
			}

			resp = append(resp, *f.Response())
			if dest.OwnerID == src.OwnerID {
				return
			}

			// update user storage usage
			if err := s.UserStore.UpdateStorageUsage(ctx, dest.OwnerID, dest.Owner.StorageUsage+totalSize); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if err := s.UserStore.UpdateStorageUsage(ctx, src.OwnerID, src.Owner.StorageUsage-totalSize); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			m.Lock()
			defer m.Unlock()
		})
	}

	wp.StopWait()

	// write log
	logs := lo.Map(files, func(f file.File, index int) file.Log {
		return file.NewLog(f.ID, user.ID, file.LogActionMove)
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// Delete godoc
// @Summary Delete
// @Description Delete
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.DeleteRequest true "Delete files request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/delete [post]
func (s *Server) Delete(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.DeleteRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	files, err := s.FileStore.ListByIDs(ctx, req.SourceIDs)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	var (
		resp      []file.File
		totalSize uint64
	)

	wp := workerpool.New(10)
	var m sync.Mutex

	for _, e := range files {
		wp.Submit(func() {
			// check if user can delete the file
			var (
				canDelete bool
				err       error
			)

			if e.IsDir {
				canDelete, err = s.PermissionService.CanDeleteDirectory(ctx, user.ID.String(), e.ID.String())
			} else {
				canDelete, err = s.PermissionService.CanDeleteFile(ctx, user.ID.String(), e.ID.String())
			}

			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if !canDelete {
				s.Logger.Errorw(permission.ErrNotPermittedToDelete.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			if err := s.FileService.Delete(ctx, e.ID.String()); err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			files, err := s.FileStore.Delete(ctx, e)
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
				return
			}

			// delete file permissions
			if e.IsDir {
				if err := s.PermissionService.DeleteDirectoryPermissions(ctx, e.ID.String()); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}

				for _, f := range files {
					totalSize += uint64(f.Size)

					if f.IsDir {
						if err := s.PermissionService.DeleteDirectoryPermissions(ctx, f.ID.String()); err != nil {
							s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
							return
						}

						return
					}

					if err := s.PermissionService.DeleteFilePermissions(ctx, f.ID.String()); err != nil {
						s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
						return
					}
				}
			} else {
				totalSize = uint64(e.Size)

				if err := s.PermissionService.DeleteFilePermissions(ctx, e.ID.String()); err != nil {
					s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
					return
				}
			}

			m.Lock()
			defer m.Unlock()
			resp = append(resp, *e.Response())
		})
	}

	// update user storage usage
	if totalSize > 0 {
		if err := s.UserStore.UpdateStorageUsage(ctx, files[0].Owner.ID, files[0].Owner.StorageUsage-totalSize); err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	wp.StopWait()

	// write log
	logs := lo.Map(files, func(f file.File, index int) file.Log {
		return file.NewLog(f.ID, user.ID, file.LogActionDelete)
	})

	if err := s.FileStore.WriteLogs(ctx, logs); err != nil {
		s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
	}

	return s.success(c, resp)
}

// GetShared godoc
// @Summary GetShared
// @Description GetShared
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.GetSharedRequest true "Get shared request"
// @Success 200 {object} model.SuccessResponse{data=model.GetSharedResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/share [get]
func (s *Server) GetShared(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.GetSharedRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	g, _ := errgroup.WithContext(ctx)
	var (
		m       sync.Mutex
		fileIDs []string
	)

	userRoleByFileID := make(map[string][]string)

	g.Go(func() error {
		ids, err := s.PermissionService.GetSharedPermissions(ctx, user.ID.String(), "Directory", "editors")
		if err != nil {
			return fmt.Errorf("get shared permissions (d-e): %w", err)
		}

		m.Lock()
		defer m.Unlock()
		fileIDs = append(fileIDs, ids...)
		for _, id := range ids {
			userRoleByFileID[id] = append(userRoleByFileID[id], "editor")
		}

		return nil
	})

	g.Go(func() error {
		ids, err := s.PermissionService.GetSharedPermissions(ctx, user.ID.String(), "Directory", "viewers")
		if err != nil {
			return fmt.Errorf("get shared permissions (d-v): %w", err)
		}

		m.Lock()
		defer m.Unlock()
		fileIDs = append(fileIDs, ids...)
		for _, id := range ids {
			userRoleByFileID[id] = append(userRoleByFileID[id], "viewer")
		}

		return nil
	})

	g.Go(func() error {
		ids, err := s.PermissionService.GetSharedPermissions(ctx, user.ID.String(), "File", "editors")
		if err != nil {
			return fmt.Errorf("get shared permissions (f-e): %w", err)
		}

		m.Lock()
		defer m.Unlock()
		fileIDs = append(fileIDs, ids...)
		for _, id := range ids {
			userRoleByFileID[id] = append(userRoleByFileID[id], "editor")
		}

		return nil
	})

	g.Go(func() error {
		ids, err := s.PermissionService.GetSharedPermissions(ctx, user.ID.String(), "File", "viewers")
		if err != nil {
			return fmt.Errorf("get shared permissions (f-v): %w", err)
		}

		m.Lock()
		defer m.Unlock()
		fileIDs = append(fileIDs, ids...)
		for _, id := range ids {
			userRoleByFileID[id] = append(userRoleByFileID[id], "viewer")
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After)

	files, err := s.FileStore.ListByIDsAndCursor(ctx, fileIDs, cursor, filter)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	for i, f := range files {
		files[i].WithUserRoles(userRoleByFileID[f.ID.String()])

		IsStarred, err := s.FileStore.IsStarred(ctx, f.ID, user.ID)
		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		files[i].WithIsStarred(IsStarred)
	}

	return s.success(c, model.GetSharedResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
	})
}

// Star godoc
// @Summary Star
// @Description Star
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.StarRequest true "Star request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/star [patch]
func (s *Server) Star(c echo.Context) error {
	var (
		ctx     = app.NewEchoContextAdapter(c)
		req     model.StarRequest
		canView bool
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	for _, id := range req.FileIDs {
		e, err := s.FileStore.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.error(c, apperror.ErrEntityNotFound(err))
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}

		if e.IsDir {
			canView, err = s.PermissionService.CanViewDirectory(ctx, user.ID.String(), e.ID.String())
		} else {
			canView, err = s.PermissionService.CanViewFile(ctx, user.ID.String(), e.ID.String())
		}

		if err != nil {
			return s.error(c, apperror.ErrInternalServer(err))
		}

		if !canView {
			return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
		}

		if err := s.FileStore.Star(ctx, e.ID, user.ID); err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.error(c, apperror.ErrEntityNotFound(err))
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}

		// write log
		if err := s.FileStore.WriteLogs(ctx, []file.Log{file.NewLog(e.ID, user.ID, file.LogActionStar)}); err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
		}
	}
	return s.success(c, nil)
}

// Unstar godoc
// @Summary Unstar
// @Description Unstar
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param payload body model.UnstarRequest true "Unstar request"
// @Success 200 {object} model.SuccessResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/unstar [patch]
func (s *Server) Unstar(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.UnstarRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)
	for _, id := range req.FileIDs {
		fileId, err := uuid.Parse(id)
		if err != nil {
			return s.error(c, apperror.ErrInvalidParam(err))
		}

		if err := s.FileStore.Unstar(ctx, fileId, user.ID); err != nil {
			if errors.Is(err, file.ErrNotFound) {
				return s.error(c, apperror.ErrEntityNotFound(err))
			}

			return s.error(c, apperror.ErrInternalServer(err))
		}
	}

	return s.success(c, nil)
}

// ListStarred godoc
// @Summary ListStarred
// @Description ListStarred
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.ListStarredRequest true "List starred request"
// @Success 200 {object} model.SuccessResponse{data=model.ListStarredResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/starred [get]
func (s *Server) ListStarred(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListStarredRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After)

	files, err := s.FileStore.ListStarred(ctx, user.ID, cursor, filter)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	files = s.mapUserRolesAndStarred(ctx, user, files)

	return s.success(c, model.ListStarredResponse{
		Entries: files,
		Cursor:  cursor.NextToken(),
	})
}

// Search godoc
// @Summary Search
// @Description Search
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.SearchRequest true "Search request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/search [get]
func (s *Server) Search(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.SearchRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	if req.ParentID == "" {
		req.ParentID = user.RootID.String()
	}

	parent, err := s.FileStore.GetByID(ctx, req.ParentID)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	canView, err := s.PermissionService.CanViewDirectory(ctx, user.ID.String(), parent.ID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if !canView {
		return s.error(c, apperror.ErrForbidden(permission.ErrNotPermittedToView))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After).WithPath(parent.FullPath())

	files, err := s.FileStore.Search(ctx, req.Query, cursor, filter)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	fullPaths := lo.Map(files, func(file file.File, index int) string {
		return file.Path
	})

	parents, err := s.FileStore.ListByFullPaths(ctx, fullPaths)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	entries := s.MapperService.FileWithParents(files, parents)
	entries = s.mapUserRolesAndStarred(ctx, user, entries)

	return s.success(c, model.SearchResponse{
		Entries: entries,
		Cursor:  cursor.NextToken(),
	})
}

// ListSuggested godoc
// @Summary ListSuggested
// @Description ListSuggested
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.ListSuggestedRequest true "List suggested request"
// @Success 200 {object} model.SuccessResponse{data=[]file.File}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/suggested [get]
func (s *Server) ListSuggested(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListSuggestedRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	files, err := s.FileStore.ListSuggested(ctx, user.ID, req.Limit, req.Dir)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	fullPaths := lo.Map(files, func(file file.File, index int) string {
		return file.Path
	})

	parents, err := s.FileStore.ListByFullPaths(ctx, fullPaths)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	entries := s.MapperService.FileWithParents(files, parents)

	entries = s.mapUserRolesAndStarred(ctx, user, entries)

	return s.success(c, entries)
}

// ListActivities godoc
// @Summary ListActivities
// @Description ListActivities
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param id path string true "File ID"
// @Param request query model.ListActivitiesRequest true "List activities request"
// @Success 200 {object} model.SuccessResponse{data=model.ListActivitiesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/{id}/activities [get]
func (s *Server) ListActivities(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListActivitiesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	activities, err := s.FileStore.ListActivities(ctx, uuid.MustParse(req.ID), cursor)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	return s.success(c, model.ListActivitiesResponse{
		Activities: activities,
		Cursor:     cursor.NextToken(),
	})
}

// GetStorage godoc
// @Summary GetStorage
// @Description GetStorage
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Success 200 {object} model.SuccessResponse{data=model.GetStorageResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/storage [get]
func (s *Server) GetStorage(c echo.Context) error {
	var ctx = app.NewEchoContextAdapter(c)
	user, _ := c.Get(ContextKeyUser).(*identity.User)

	root, err := s.FileStore.GetByID(ctx, user.RootID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	files, err := s.FileStore.GetAllFiles(ctx, root.FullPath())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	storage := file.NewStorage(files)

	return s.success(c, model.GetStorageResponse{
		Storage:  storage,
		Capacity: user.StorageCapacity,
	})
}

// ListFileSizes godoc
// @Summary ListFileSizes
// @Description ListFileSizes
// @Tags file
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" default(Bearer <session_token>)
// @Param request query model.ListFileSizesRequest true "List file sizes request"
// @Success 200 {object} model.SuccessResponse{data=model.ListFileSizesResponse}
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /files/sizes [get]
func (s *Server) ListFileSizes(c echo.Context) error {
	var (
		ctx = app.NewEchoContextAdapter(c)
		req model.ListFileSizesRequest
	)

	if err := c.Bind(&req); err != nil {
		return s.error(c, apperror.ErrInvalidRequest(err))
	}

	if err := req.Validate(ctx); err != nil {
		return s.error(c, apperror.ErrInvalidParam(err))
	}

	user, _ := c.Get(ContextKeyUser).(*identity.User)

	root, err := s.FileStore.GetByID(ctx, user.RootID.String())
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	cursor := pagination.NewCursor(req.Cursor, req.Limit)
	filter := file.NewFilter(req.Type, req.After)

	sizes, err := s.FileStore.ListFiles(ctx, root.FullPath(), cursor, filter, req.Asc)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	sizes = s.mapUserRolesAndStarred(ctx, user, sizes)

	return s.success(c, model.ListFileSizesResponse{
		Entries: sizes,
		Cursor:  cursor.NextToken(),
	})
}

func (s *Server) RegisterFileRoutes(router *echo.Group) {
	router.Use(s.passwordChangedAtMiddleware)
	router.GET("/trash", s.ListTrash)
	router.GET("/share", s.GetShared)
	router.GET("/search", s.Search)
	router.GET("/starred", s.ListStarred)
	router.GET("/suggested", s.ListSuggested)
	router.GET("/storage", s.GetStorage)
	router.GET("/sizes", s.ListFileSizes)
	router.POST("/download", s.DownloadBatch)
	router.POST("/share", s.Share) // share file or directory with some users
	router.POST("/directories", s.CreateDirectory)
	router.POST("/copy", s.CopyFiles)
	router.POST("/move", s.Move)
	router.PATCH("/rename", s.Rename)
	router.POST("/move/trash", s.MoveToTrash)
	router.POST("/restore", s.RestoreFromTrash)
	router.POST("/delete", s.Delete)
	router.POST("", s.UploadFiles)
	router.POST("/chunks", s.UploadChunk)
	router.PATCH("/general-access", s.UpdateGeneralAccess)
	router.PATCH("/access", s.UpdateAccess)
	router.GET("/:id", s.ListEntries)
	router.GET("/:id/page", s.ListPageEntries)
	router.GET("/:id/metadata", s.GetMetadata)
	router.GET("/:id/download", s.Download)
	router.GET("/:id/access", s.Access) // get access to the shared file or directory
	router.GET("/:id/activities", s.ListActivities)
	router.PATCH("/star", s.Star)
	router.PATCH("/unstar", s.Unstar)

}

func (s *Server) mapUserRolesAndStarred(ctx context.Context, user *identity.User, entries []file.File) []file.File {
	return lop.Map(entries, func(e file.File, i int) file.File {
		userRoles, err := s.PermissionService.GetFileUserRoles(ctx, user.ID.String(), e.ID.String(), e.IsDir)
		if err != nil {
			return e
		}

		isStarred, err := s.FileStore.IsStarred(ctx, e.ID, user.ID)
		if err != nil {
			return e
		}

		return *e.WithUserRoles(userRoles).WithIsStarred(isStarred)
	})
}

func (s *Server) createFile(ctx context.Context, parent *file.File, reader io.Reader, filename string, ownerID uuid.UUID, more bool) (*file.File, error) {
	id := uuid.New()

	contentType, src, err := app.DetectContentType(reader)
	if err != nil {
		return nil, fmt.Errorf("detect content type: %w", err)
	}

	_, err = s.FileService.CreateFile(ctx, src, id.String(), contentType)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	entry, err := s.FileService.GetMetadata(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	f := entry.ToFile(filename).WithID(id).WithPath(parent.FullPath()).WithOwnerID(ownerID).WithMore(more)
	if err := s.FileStore.Create(ctx, f); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	// create file permissions
	if err := s.PermissionService.CreateFilePermissions(ctx, ownerID.String(), f.ID.String(), parent.ID.String()); err != nil {
		return nil, fmt.Errorf("create file permissions: %w", err)
	}

	return f, nil
}

func (s *Server) appendChunk(ctx context.Context, fileID string, reader io.Reader, last bool) (*file.File, error) {
	_, err := s.FileService.AppendFile(ctx, reader, fileID)
	if err != nil {
		return nil, fmt.Errorf("append file: %w", err)
	}

	entry, err := s.FileService.GetMetadata(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	f, err := s.FileStore.UpdateChunk(ctx, uuid.MustParse(fileID), entry.Size, last)
	if err != nil {
		return nil, fmt.Errorf("update chunk: %w", err)
	}

	return f, nil
}

func (s *Server) downloadZip(c echo.Context, path string, ids []string) error {
	var ctx = app.NewEchoContextAdapter(c)

	// list selected children
	files, err := s.FileStore.ListSelectedChildren(ctx, path, ids)
	if err != nil {
		return s.error(c, apperror.ErrInternalServer(err))
	}

	if len(files) == 0 {
		return s.error(c, apperror.ErrNoFilesSelected(nil))
	}

	zw := zip.NewWriter(c.Response().Writer)
	defer zw.Close()

	for _, f := range files {
		path := strings.TrimPrefix(f.FullPath(), path+"/")

		if f.IsDir {
			_, err := zw.Create(path + "/")
			if err != nil {
				s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
			}

			continue
		}

		r, _, err := s.FileService.DownloadFile(ctx, f.ID.String())
		if err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
			continue
		}
		defer r.Close()

		w, err := zw.Create(path)
		if err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
			continue
		}

		if _, err := io.Copy(w, r); err != nil {
			s.Logger.Errorw(err.Error(), zap.String("request_id", s.requestID(c)))
			continue
		}
	}

	c.Response().Header().Set(echo.HeaderContentType, "application/zip")

	return nil
}
