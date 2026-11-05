package repository

import (
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"gorm.io/gorm"
)

type CourseRepo struct {
	db *gorm.DB
}

func NewCourseRepo(db *gorm.DB) *CourseRepo {
	return &CourseRepo{db: db}
}

type CourseQuery struct {
	CourseCode string
	Name       string
	Department string
	Teacher    string
	Semester   string
	Page       int
	PageSize   int
}

func (r *CourseRepo) List(q CourseQuery) (*PaginatedResult[model.Course], error) {
	tx := r.db.Model(&model.Course{})

	if q.CourseCode != "" {
		tx = tx.Where("course_code LIKE ?", "%"+q.CourseCode+"%")
	}
	if q.Name != "" {
		tx = tx.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.Department != "" {
		tx = tx.Where("department = ?", q.Department)
	}
	if q.Teacher != "" {
		tx = tx.Where("teacher LIKE ?", "%"+q.Teacher+"%")
	}
	if q.Semester != "" {
		tx = tx.Where("semester = ?", q.Semester)
	}

	var total int64
	tx.Count(&total)

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	var courses []model.Course
	err := tx.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Order("id DESC").Find(&courses).Error
	if err != nil {
		return nil, err
	}

	return &PaginatedResult[model.Course]{
		Items:    courses,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *CourseRepo) GetByID(id uint) (*model.Course, error) {
	var c model.Course
	err := r.db.First(&c, id).Error
	return &c, err
}

func (r *CourseRepo) Create(c *model.Course) error {
	return r.db.Create(c).Error
}

func (r *CourseRepo) Update(c *model.Course) error {
	return r.db.Save(c).Error
}

func (r *CourseRepo) Delete(id uint) error {
	return r.db.Delete(&model.Course{}, id).Error
}
