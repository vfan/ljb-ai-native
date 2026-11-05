package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type GradeHandler struct {
	svc *service.GradeService
}

func NewGradeHandler(svc *service.GradeService) *GradeHandler {
	return &GradeHandler{svc: svc}
}

func (h *GradeHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/grades")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.GET("/stats/course/:courseId", h.StatsByCourse)
	g.GET("/gpa/student/:studentId", h.CalculateGPA)
	g.GET("/warning", h.AcademicWarning)
}

func (h *GradeHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	studentID, _ := strconv.ParseUint(c.Query("student_id"), 10, 64)
	courseID, _ := strconv.ParseUint(c.Query("course_id"), 10, 64)

	q := repository.GradeQuery{
		StudentID: uint(studentID),
		CourseID:  uint(courseID),
		Semester:  c.Query("semester"),
		Status:    c.Query("status"),
		Page:      page,
		PageSize:  pageSize,
	}

	result, err := h.svc.List(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *GradeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	grade, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "成绩不存在"})
		return
	}
	c.JSON(http.StatusOK, grade)
}

func (h *GradeHandler) Create(c *gin.Context) {
	var grade model.Grade
	if err := c.ShouldBindJSON(&grade); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(&grade); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, grade)
}

type gradeUpdateInput struct {
	StudentID *uint    `json:"student_id"`
	CourseID  *uint    `json:"course_id"`
	Score     *float64 `json:"score"`
	Semester  *string  `json:"semester"`
	Status    *string  `json:"status"`
}

func (h *GradeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var input gradeUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "成绩记录不存在"})
		return
	}
	if input.StudentID != nil {
		existing.StudentID = *input.StudentID
	}
	if input.CourseID != nil {
		existing.CourseID = *input.CourseID
	}
	if input.Score != nil {
		existing.Score = *input.Score
	}
	if input.Semester != nil {
		existing.Semester = *input.Semester
	}
	if input.Status != nil {
		existing.Status = model.GradeStatus(*input.Status)
	}

	if err := h.svc.Update(existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *GradeHandler) Delete(c *gin.Context) {
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

func (h *GradeHandler) StatsByCourse(c *gin.Context) {
	courseID, err := strconv.ParseUint(c.Param("courseId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的课程ID"})
		return
	}

	stats, err := h.svc.StatsByCourse(uint(courseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *GradeHandler) CalculateGPA(c *gin.Context) {
	studentID, err := strconv.ParseUint(c.Param("studentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学生ID"})
		return
	}

	gpa, err := h.svc.CalculateGPA(uint(studentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gpa)
}

func (h *GradeHandler) AcademicWarning(c *gin.Context) {
	threshold, _ := strconv.ParseFloat(c.DefaultQuery("threshold", "2.0"), 64)

	students, err := h.svc.GetLowGPAStudents(threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"threshold": threshold, "students": students})
}
