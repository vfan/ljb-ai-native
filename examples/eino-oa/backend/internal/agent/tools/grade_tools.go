package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	reportservice "github.com/wuhaitao/eino-oa/backend/internal/report"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type queryGradesInput struct {
	StudentID uint   `json:"student_id,omitempty"`
	CourseID  uint   `json:"course_id,omitempty"`
	Semester  string `json:"semester,omitempty"`
}

type addGradeInput struct {
	StudentID uint    `json:"student_id"`
	CourseID  uint    `json:"course_id"`
	Score     float64 `json:"score"`
	Semester  string  `json:"semester,omitempty"`
	Status    string  `json:"status,omitempty"`
}

type updateGradeInput struct {
	ID    uint    `json:"id"`
	Score float64 `json:"score"`
}

type gradeAnalysisInput struct {
	CourseID uint `json:"course_id"`
}

type academicWarningInput struct {
	Threshold float64 `json:"threshold,omitempty"`
}

type studentReportInput struct {
	StudentID uint `json:"student_id"`
}

func NewQueryGradesTool(svc *service.GradeService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "query_grades",
			Desc: "查询成绩记录。可按学生ID、课程ID、学期筛选。返回成绩列表，包含学生和课程详情。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"student_id": {Type: "integer", Desc: "学生ID"},
				"course_id":  {Type: "integer", Desc: "课程ID"},
				"semester":   {Type: "string", Desc: "学期，如 2025-2026-1"},
			}),
		},
		func(ctx context.Context, input *queryGradesInput) (output string, err error) {
			result, err := svc.List(repository.GradeQuery{
				StudentID: input.StudentID,
				CourseID:  input.CourseID,
				Semester:  input.Semester,
				PageSize:  100,
			})
			if err != nil {
				return "", fmt.Errorf("查询失败: %w", err)
			}
			data, _ := json.Marshal(result)
			return string(data), nil
		},
	)
}

func NewAddGradeTool(svc *service.GradeService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "add_grade",
			Desc: "录入一条成绩记录。需提供学生ID、课程ID和分数。绩点会自动计算。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"student_id": {Type: "integer", Desc: "学生ID（必填）", Required: true},
				"course_id":  {Type: "integer", Desc: "课程ID（必填）", Required: true},
				"score":      {Type: "number", Desc: "成绩分数 0-100（必填）", Required: true},
				"semester":   {Type: "string", Desc: "学期"},
				"status":     {Type: "string", Desc: "状态：正常、补考、重修"},
			}),
		},
		func(ctx context.Context, input *addGradeInput) (output string, err error) {
			g := &model.Grade{
				StudentID: input.StudentID,
				CourseID:  input.CourseID,
				Score:     input.Score,
				Semester:  input.Semester,
				Status:    model.GradeStatus(input.Status),
			}
			if g.Status == "" {
				g.Status = model.GradeStatusNormal
			}
			if err := svc.Create(g); err != nil {
				return "", fmt.Errorf("录入失败: %w", err)
			}
			data, _ := json.Marshal(g)
			return string(data), nil
		},
	)
}

func NewUpdateGradeTool(svc *service.GradeService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "update_grade",
			Desc: "修改一条成绩记录的分数。需提供成绩ID和新分数。绩点会自动重新计算。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"id":    {Type: "integer", Desc: "成绩记录ID（必填）", Required: true},
				"score": {Type: "number", Desc: "新的成绩分数 0-100（必填）", Required: true},
			}),
		},
		func(ctx context.Context, input *updateGradeInput) (output string, err error) {
			existing, err := svc.GetByID(input.ID)
			if err != nil {
				return "", fmt.Errorf("成绩记录不存在: %w", err)
			}
			existing.Score = input.Score
			if err := svc.Update(existing); err != nil {
				return "", fmt.Errorf("修改失败: %w", err)
			}
			data, _ := json.Marshal(existing)
			return string(data), nil
		},
	)
}

func NewGradeAnalysisTool(svc *service.GradeService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "grade_analysis",
			Desc: "分析某门课程的成绩分布情况。返回平均分、最高分、最低分、及格率、参考人数。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"course_id": {Type: "integer", Desc: "课程ID（必填）", Required: true},
			}),
		},
		func(ctx context.Context, input *gradeAnalysisInput) (output string, err error) {
			stats, err := svc.StatsByCourse(input.CourseID)
			if err != nil {
				return "", fmt.Errorf("分析失败: %w", err)
			}
			data, _ := json.Marshal(stats)
			return string(data), nil
		},
	)
}

func NewAcademicWarningTool(svc *service.GradeService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "academic_warning",
			Desc: "学业预警：查找GPA低于指定阈值的学生列表。默认阈值为2.0。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"threshold": {Type: "number", Desc: "GPA阈值，默认2.0"},
			}),
		},
		func(ctx context.Context, input *academicWarningInput) (output string, err error) {
			threshold := input.Threshold
			if threshold <= 0 {
				threshold = 2.0
			}
			students, err := svc.GetLowGPAStudents(threshold)
			if err != nil {
				return "", fmt.Errorf("查询失败: %w", err)
			}
			data, _ := json.Marshal(map[string]interface{}{
				"threshold": threshold,
				"count":     len(students),
				"students":  students,
			})
			return string(data), nil
		},
	)
}

func NewStudentReportTool(reportSvc *reportservice.Service) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "student_report",
			Desc: "生成结构化学生综合报告，包含总结、学习优势、薄弱课程和下一步建议。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"student_id": {Type: "integer", Desc: "学生ID（必填）", Required: true},
			}),
		},
		func(ctx context.Context, input *studentReportInput) (output string, err error) {
			result, err := reportSvc.Generate(ctx, input.StudentID)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("序列化报告失败: %w", err)
			}
			return string(data), nil
		},
	)
}
