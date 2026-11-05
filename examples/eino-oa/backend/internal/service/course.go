package service

import (
	"errors"
	"strings"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
)

type CourseService struct {
	repo *repository.CourseRepo
}

func NewCourseService(repo *repository.CourseRepo) *CourseService {
	return &CourseService{repo: repo}
}

func (s *CourseService) List(q repository.CourseQuery) (*repository.PaginatedResult[model.Course], error) {
	return s.repo.List(q)
}

func (s *CourseService) GetByID(id uint) (*model.Course, error) {
	return s.repo.GetByID(id)
}

func (s *CourseService) Create(course *model.Course) error {
	if err := validateCourse(course); err != nil {
		return err
	}
	return s.repo.Create(course)
}

func (s *CourseService) Update(course *model.Course) error {
	if course.ID == 0 {
		return errors.New("课程ID不能为空")
	}
	if err := validateCourse(course); err != nil {
		return err
	}
	return s.repo.Update(course)
}

func validateCourse(course *model.Course) error {
	course.CourseCode = strings.TrimSpace(course.CourseCode)
	course.Name = strings.TrimSpace(course.Name)
	if course.CourseCode == "" || course.Name == "" {
		return errors.New("课程编号和课程名不能为空")
	}
	if course.Credits < 0 || course.Credits > 30 {
		return errors.New("学分必须在 0-30 之间")
	}
	return nil
}

func (s *CourseService) Delete(id uint) error {
	return s.repo.Delete(id)
}
