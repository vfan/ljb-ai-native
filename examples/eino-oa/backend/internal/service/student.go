package service

import (
	"errors"
	"strings"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
)

type StudentService struct {
	repo *repository.StudentRepo
}

func NewStudentService(repo *repository.StudentRepo) *StudentService {
	return &StudentService{repo: repo}
}

func (s *StudentService) List(q repository.StudentQuery) (*repository.PaginatedResult[model.Student], error) {
	return s.repo.List(q)
}

func (s *StudentService) GetByID(id uint) (*model.Student, error) {
	return s.repo.GetByID(id)
}

func (s *StudentService) GetByStudentNo(no string) (*model.Student, error) {
	return s.repo.GetByStudentNo(no)
}

func (s *StudentService) Create(student *model.Student) error {
	if student.Status == "" {
		student.Status = model.StudentStatusActive
	}
	if err := validateStudent(student); err != nil {
		return err
	}
	return s.repo.Create(student)
}

func (s *StudentService) Update(student *model.Student) error {
	if student.ID == 0 {
		return errors.New("学生ID不能为空")
	}
	if err := validateStudent(student); err != nil {
		return err
	}
	return s.repo.Update(student)
}

func validateStudent(student *model.Student) error {
	student.StudentNo = strings.TrimSpace(student.StudentNo)
	student.Name = strings.TrimSpace(student.Name)
	if student.StudentNo == "" || student.Name == "" {
		return errors.New("学号和姓名不能为空")
	}
	if student.Age < 0 || student.Age > 150 {
		return errors.New("年龄必须在 0-150 之间")
	}
	switch student.Status {
	case model.StudentStatusActive, model.StudentStatusSuspended,
		model.StudentStatusGraduated, model.StudentStatusDropped:
	default:
		return errors.New("无效的学生状态")
	}
	return nil
}

func (s *StudentService) Delete(id uint) error {
	return s.repo.Delete(id)
}
