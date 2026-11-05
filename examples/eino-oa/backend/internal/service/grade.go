package service

import (
	"errors"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
)

type GradeService struct {
	repo *repository.GradeRepo
}

func NewGradeService(repo *repository.GradeRepo) *GradeService {
	return &GradeService{repo: repo}
}

func (s *GradeService) List(q repository.GradeQuery) (*repository.PaginatedResult[model.Grade], error) {
	return s.repo.List(q)
}

func (s *GradeService) GetByID(id uint) (*model.Grade, error) {
	return s.repo.GetByID(id)
}

func (s *GradeService) Create(grade *model.Grade) error {
	if grade.Status == "" {
		grade.Status = model.GradeStatusNormal
	}
	if err := validateGrade(grade); err != nil {
		return err
	}
	return s.repo.Create(grade)
}

func (s *GradeService) Update(grade *model.Grade) error {
	if grade.ID == 0 {
		return errors.New("成绩ID不能为空")
	}
	if err := validateGrade(grade); err != nil {
		return err
	}
	return s.repo.Update(grade)
}

func validateGrade(grade *model.Grade) error {
	if grade.StudentID == 0 || grade.CourseID == 0 {
		return errors.New("学生ID和课程ID不能为空")
	}
	if grade.Score < 0 || grade.Score > 100 {
		return errors.New("成绩必须在 0-100 之间")
	}
	switch grade.Status {
	case model.GradeStatusNormal, model.GradeStatusRetake, model.GradeStatusRepeat:
	default:
		return errors.New("无效的成绩状态")
	}
	return nil
}

func (s *GradeService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *GradeService) StatsByCourse(courseID uint) (*repository.GradeStats, error) {
	return s.repo.StatsByCourse(courseID)
}

func (s *GradeService) CalculateGPA(studentID uint) (*repository.StudentGPA, error) {
	return s.repo.CalculateGPA(studentID)
}

func (s *GradeService) GetLowGPAStudents(threshold float64) ([]repository.StudentGPA, error) {
	return s.repo.GetLowGPAStudents(threshold)
}
