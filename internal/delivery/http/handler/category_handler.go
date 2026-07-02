package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/request"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/response"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
)

type CategoryHandler struct {
	categoryUseCase *usecase.CategoryUseCase
}

func NewCategoryHandler(categoryUseCase *usecase.CategoryUseCase) *CategoryHandler {
	return &CategoryHandler{categoryUseCase: categoryUseCase}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var body request.CategoryRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		slog.Error("create category validation failed", slog.Any("error", err))
		utils.HandleError(c, apperrors.ParseValidationError(err))
		return
	}

	var parentIDParam *int64
	if body.ParentID != nil {
		// ParentID was provided in the JSON payload
		parentIDParam = body.ParentID
	} else {
		// ParentID was NOT provided (it is nil)
		// You can explicitly set it to nil, or handle root-category business logic here
		parentIDParam = nil
	}

	category, err := h.categoryUseCase.CreateCategory(c.Request.Context(), body.Name, parentIDParam)

	if err != nil {
		slog.Error("failed to create category", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Success(c, category)
}

func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {

	id, err := utils.ParseID(c, "id")
	if err != nil {
		slog.Error("failed to parse category ID", slog.Any("error", err))
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"category": "Invalid Category ID"}})
		return
	}

	category, err := h.categoryUseCase.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to get category", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Success(c, category)
}

func (h *CategoryHandler) GetCategoryByIDWithProducts(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		slog.Error("failed to parse category ID", slog.Any("error", err))
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"category": "Invalid Category ID"}})
		return
	}

	category, err := h.categoryUseCase.GetCategoryByIDWithProducts(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to GetCategoryByIDWithProducts", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Success(c, category)
}

func (h *CategoryHandler) GetAllCategories(c *gin.Context) {
	parentIDStr := c.Query("parent_id")

	var parentID int64
	var err error

	if parentIDStr != "" {
		// 2. Only parse if the string is actually present
		parsedID, err := utils.ParseID(c, "parent_id")
		if err != nil {
			slog.Error("failed to parse parent categiry ID", slog.Any("error", err))
			utils.HandleError(c, &apperrors.BadRequestError{
				Errors: map[string]string{"category": "Invalid Category ID"},
			})
			return
		}
		parentID = parsedID
	}

	params := entity.GetCategoriesParams{
		QueryParams: utils.ParseQueryParams(c),
		ParentID:    parentID,
	}

	categories, total, err := h.categoryUseCase.GetAllCategories(c.Request.Context(), params)

	if err != nil {
		slog.Error("failed to get all categories", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Paginated(c, categories, total, params.Page, params.Limit)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {

	id, err := utils.ParseID(c, "id")
	if err != nil {
		slog.Error("failed to parse category ID", slog.Any("error", err))
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"category": "Invalid Category ID"}})
		return
	}

	var body request.CategoryRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		slog.Error("update category validation failed", slog.Any("error", err))
		utils.HandleError(c, apperrors.ParseValidationError(err))
		return
	}

	category := &entity.Category{
		ID:       id,
		Name:     body.Name,
		Slug:     utils.GenerateSlug(body.Name),
		ParentID: body.ParentID,
	}

	if err := h.categoryUseCase.UpdateCategory(c.Request.Context(), category); err != nil {
		slog.Error("failed to update category", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Success(c, category)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		slog.Error("failed to parse category ID", slog.Any("error", err))
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"category": "Invalid Category ID"}})
		return
	}

	err = h.categoryUseCase.DeleteCategory(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to delete category", slog.Any("error", err))
		utils.HandleError(c, err)
		return
	}

	response.Message(c, "Category Deleted")
}
