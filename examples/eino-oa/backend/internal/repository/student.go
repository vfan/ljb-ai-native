package repository

import (
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"gorm.io/gorm"
)

type StudentRepo struct {
	db *gorm.DB
}

func NewStudentRepo(db *gorm.DB) *StudentRepo {
	return &StudentRepo{db: db}
}

type StudentQuery struct {
	StudentNo  string
	Name       string
	Department string
	Major      string
	ClassName  string
	Status     string
	Page       int
	PageSize   int
}

type PaginatedResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func (r *StudentRepo) List(q StudentQuery) (*PaginatedResult[model.Student], error) {
	tx := r.db.Model(&model.Student{})

	if q.StudentNo != "" {
		tx = tx.Where("student_no LIKE ?", "%"+q.StudentNo+"%")
	}
	if q.Name != "" {
		tx = tx.Where("name LIKE ?", "%"+q.Name+"%")
	}
	if q.Department != "" {
		tx = tx.Where("department = ?", q.Department)
	}
	if q.Major != "" {
		tx = tx.Where("major = ?", q.Major)
	}
	if q.ClassName != "" {
		tx = tx.Where("class_name = ?", q.ClassName)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}

	var total int64
	tx.Count(&total)

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	var students []model.Student
	err := tx.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Order("id DESC").Find(&students).Error
	if err != nil {
		return nil, err
	}

	return &PaginatedResult[model.Student]{
		Items:    students,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *StudentRepo) GetByID(id uint) (*model.Student, error) {
	var s model.Student
	err := r.db.First(&s, id).Error
	return &s, err
}

func (r *StudentRepo) GetByStudentNo(no string) (*model.Student, error) {
	var s model.Student
	err := r.db.Where("student_no = ?", no).First(&s).Error
	return &s, err
}

func (r *StudentRepo) Create(s *model.Student) error {
	return r.db.Create(s).Error
}

func (r *StudentRepo) Update(s *model.Student) error {
	return r.db.Save(s).Error
}

func (r *StudentRepo) Delete(id uint) error {
	return r.db.Delete(&model.Student{}, id).Error
}
