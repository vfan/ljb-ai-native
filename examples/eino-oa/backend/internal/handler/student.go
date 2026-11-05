package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type StudentHandler struct {
	svc *service.StudentService
}

func NewStudentHandler(svc *service.StudentService) *StudentHandler {
	return &StudentHandler{svc: svc}
}

func (h *StudentHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/students")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *StudentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	q := repository.StudentQuery{
		StudentNo:  c.Query("student_no"),
		Name:       c.Query("name"),
		Department: c.Query("department"),
		Major:      c.Query("major"),
		ClassName:  c.Query("class_name"),
		Status:     c.Query("status"),
		Page:       page,
		PageSize:   pageSize,
	}

	result, err := h.svc.List(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *StudentHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	student, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func (h *StudentHandler) Create(c *gin.Context) {
	var student model.Student
	if err := c.ShouldBindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, student)
}

// studentUpdateInput 使用指针区分“未传字段”和“显式置空”，避免 PUT 把未提供的字段清零。
type studentUpdateInput struct {
	StudentNo      *string `json:"student_no"`
	Name           *string `json:"name"`
	Gender         *string `json:"gender"`
	Age            *int    `json:"age"`
	Phone          *string `json:"phone"`
	Email          *string `json:"email"`
	Department     *string `json:"department"`
	Major          *string `json:"major"`
	ClassName      *string `json:"class_name"`
	EnrollmentYear *int    `json:"enrollment_year"`
	Status         *string `json:"status"`
}

func (h *StudentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var input studentUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学生不存在"})
		return
	}
	if input.StudentNo != nil {
		existing.StudentNo = *input.StudentNo
	}
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Gender != nil {
		existing.Gender = *input.Gender
	}
	if input.Age != nil {
		existing.Age = *input.Age
	}
	if input.Phone != nil {
		existing.Phone = *input.Phone
	}
	if input.Email != nil {
		existing.Email = *input.Email
	}
	if input.Department != nil {
		existing.Department = *input.Department
	}
	if input.Major != nil {
		existing.Major = *input.Major
	}
	if input.ClassName != nil {
		existing.ClassName = *input.ClassName
	}
	if input.EnrollmentYear != nil {
		existing.EnrollmentYear = *input.EnrollmentYear
	}
	if input.Status != nil {
		existing.Status = model.StudentStatus(*input.Status)
	}

	if err := h.svc.Update(existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *StudentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
