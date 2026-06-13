package handler

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/request"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/response"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
)

type ProductHandler struct {
	productUsecase *usecase.ProductUseCase
}

func NewProductHandler(productUsecase *usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{productUsecase: productUsecase}
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {

	var req request.CreateProduct
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleError(c, apperrors.ParseValidationError(err))
		return
	}

	product := entity.NewProduct(req.Name, req.Description, req.PriceCents, req.Stock, req.CategoryID)

	if err := h.productUsecase.Create(c.Request.Context(), product); err != nil {
		utils.HandleError(c, err)
		return
	}

	response.Success(c, product)
}

func (h *ProductHandler) GetProductByID(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"product": "Invalid Product ID"}})
		return
	}

	product, err := h.productUsecase.GetByID(c.Request.Context(), id)

	if err != nil {
		log.Printf("Server Error: %v", err)
		utils.HandleError(c, err)
		return
	}

	response.Success(c, product)
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	category_id, err := utils.ParseID(c, "category_id")
	if err != nil {
		category_id = 0
	}

	req := utils.ParseQueryParams(c)

	params := entity.GetProductsParams{
		QueryParams: req,
		CategoryID:  category_id,
	}

	products, total, err := h.productUsecase.List(c.Request.Context(), params)

	if err != nil {
		utils.HandleError(c, err)
		return
	}

	if len(products) == 0 {
		products = []entity.Product{}
	}

	response.Paginated(c, products, total, params.Page, params.Limit)
}

func (h *ProductHandler) GetProductsByCategoryID(c *gin.Context) {
	category_id, err := utils.ParseID(c, "category_id")
	if err != nil {
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"category": "Invalid Category ID"}})
		return
	}

	req := utils.ParseQueryParams(c)

	params := entity.GetProductsParams{
		QueryParams: req,
		CategoryID:  category_id,
	}

	products, total, err := h.productUsecase.List(c.Request.Context(), params)

	if err != nil {
		utils.HandleError(c, err)
		return
	}

	response.Paginated(c, products, total, params.Page, params.Limit)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"product": "Invalid Product ID"}})
		return
	}

	var req request.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleError(c, err)
		return
	}

	product, err := h.productUsecase.Update(c.Request.Context(), id, &entity.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
	})

	if err != nil {
		utils.HandleError(c, err)
		return
	}

	response.Success(c, product)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := utils.ParseID(c, "id")
	if err != nil {
		utils.HandleError(c, &apperrors.BadRequestError{Errors: map[string]string{"product": "Invalid Product ID"}})
		return
	}
	err = h.productUsecase.Delete(c.Request.Context(), id)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	response.Message(c, "Product deleted")
}
