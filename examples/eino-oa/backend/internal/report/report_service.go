package report

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type Service struct {
	studentSvc *service.StudentService
	gradeSvc   *service.GradeService
	chain      compose.Runnable[map[string]any, Result]
}

func NewService(
	studentSvc *service.StudentService,
	gradeSvc *service.GradeService,
	chain compose.Runnable[map[string]any, Result],
) *Service {
	return &Service{
		studentSvc: studentSvc,
		gradeSvc:   gradeSvc,
		chain:      chain,
	}
}

func (s *Service) Generate(ctx context.Context, studentID uint) (*Result, error) {
	student, err := s.studentSvc.GetByID(studentID)
	if err != nil {
		return nil, fmt.Errorf("查询学生失败: %w", err)
	}

	grades, err := s.gradeSvc.List(repository.GradeQuery{
		StudentID: studentID,
		PageSize:  100,
	})
	if err != nil {
		return nil, fmt.Errorf("查询成绩失败: %w", err)
	}

	gpa, err := s.gradeSvc.CalculateGPA(studentID)
	if err != nil {
		return nil, fmt.Errorf("计算 GPA 失败: %w", err)
	}

	studentJSON, err := json.Marshal(student)
	if err != nil {
		return nil, fmt.Errorf("序列化学生数据失败: %w", err)
	}
	gradesJSON, err := json.Marshal(grades.Items)
	if err != nil {
		return nil, fmt.Errorf("序列化成绩数据失败: %w", err)
	}
	gpaJSON, err := json.Marshal(gpa)
	if err != nil {
		return nil, fmt.Errorf("序列化 GPA 数据失败: %w", err)
	}

	result, err := s.chain.Invoke(ctx, map[string]any{
		"student": string(studentJSON),
		"grades":  string(gradesJSON),
		"gpa":     string(gpaJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("生成学生报告失败: %w", err)
	}
	return &result, nil
}
