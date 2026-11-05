package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type queryStudentInput struct {
	StudentNo  string `json:"student_no,omitempty"`
	Name       string `json:"name,omitempty"`
	Department string `json:"department,omitempty"`
	Major      string `json:"major,omitempty"`
	Status     string `json:"status,omitempty"`
}

type addStudentInput struct {
	StudentNo      string `json:"student_no"`
	Name           string `json:"name"`
	Gender         string `json:"gender,omitempty"`
	Age            int    `json:"age,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Email          string `json:"email,omitempty"`
	Department     string `json:"department,omitempty"`
	Major          string `json:"major,omitempty"`
	ClassName      string `json:"class_name,omitempty"`
	EnrollmentYear int    `json:"enrollment_year,omitempty"`
}

type updateStudentInput struct {
	ID         uint   `json:"id"`
	Name       string `json:"name,omitempty"`
	Gender     string `json:"gender,omitempty"`
	Age        int    `json:"age,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
	Major      string `json:"major,omitempty"`
	ClassName  string `json:"class_name,omitempty"`
	Status     string `json:"status,omitempty"`
}

type deleteStudentInput struct {
	ID uint `json:"id"`
}

func NewQueryStudentTool(svc *service.StudentService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "query_student",
			Desc: "查询学生信息。可按学号、姓名、院系、专业、状态进行模糊搜索。返回匹配的学生列表。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"student_no": {Type: "string", Desc: "学号（支持模糊匹配）"},
				"name":       {Type: "string", Desc: "姓名（支持模糊匹配）"},
				"department": {Type: "string", Desc: "院系"},
				"major":      {Type: "string", Desc: "专业"},
				"status":     {Type: "string", Desc: "状态：在读、休学、毕业、退学"},
			}),
		},
		func(ctx context.Context, input *queryStudentInput) (output string, err error) {
			result, err := svc.List(repository.StudentQuery{
				StudentNo:  input.StudentNo,
				Name:       input.Name,
				Department: input.Department,
				Major:      input.Major,
				Status:     input.Status,
				PageSize:   50,
			})
			if err != nil {
				return "", fmt.Errorf("查询失败: %w", err)
			}
			data, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("序列化查询结果失败: %w", err)
			}
			return string(data), nil
		},
	)
}

func NewAddStudentTool(svc *service.StudentService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "add_student",
			Desc: "新增一个学生记录。需提供学号和姓名，其他字段可选。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"student_no":      {Type: "string", Desc: "学号（必填）", Required: true},
				"name":            {Type: "string", Desc: "姓名（必填）", Required: true},
				"gender":          {Type: "string", Desc: "性别"},
				"age":             {Type: "integer", Desc: "年龄"},
				"phone":           {Type: "string", Desc: "电话"},
				"email":           {Type: "string", Desc: "邮箱"},
				"department":      {Type: "string", Desc: "院系"},
				"major":           {Type: "string", Desc: "专业"},
				"class_name":      {Type: "string", Desc: "班级"},
				"enrollment_year": {Type: "integer", Desc: "入学年份"},
			}),
		},
		func(ctx context.Context, input *addStudentInput) (output string, err error) {
			s := &model.Student{
				StudentNo:      input.StudentNo,
				Name:           input.Name,
				Gender:         input.Gender,
				Age:            input.Age,
				Phone:          input.Phone,
				Email:          input.Email,
				Department:     input.Department,
				Major:          input.Major,
				ClassName:      input.ClassName,
				EnrollmentYear: input.EnrollmentYear,
				Status:         model.StudentStatusActive,
			}
			if err := svc.Create(s); err != nil {
				return "", fmt.Errorf("创建失败: %w", err)
			}
			data, _ := json.Marshal(s)
			return string(data), nil
		},
	)
}

func NewUpdateStudentTool(svc *service.StudentService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "update_student",
			Desc: "更新学生信息。需提供学生ID，以及要更新的字段。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"id":         {Type: "integer", Desc: "学生ID（必填）", Required: true},
				"name":       {Type: "string", Desc: "姓名"},
				"gender":     {Type: "string", Desc: "性别"},
				"age":        {Type: "integer", Desc: "年龄"},
				"phone":      {Type: "string", Desc: "电话"},
				"email":      {Type: "string", Desc: "邮箱"},
				"department": {Type: "string", Desc: "院系"},
				"major":      {Type: "string", Desc: "专业"},
				"class_name": {Type: "string", Desc: "班级"},
				"status":     {Type: "string", Desc: "状态：在读、休学、毕业、退学"},
			}),
		},
		func(ctx context.Context, input *updateStudentInput) (output string, err error) {
			existing, err := svc.GetByID(input.ID)
			if err != nil {
				return "", fmt.Errorf("学生不存在: %w", err)
			}
			if input.Name != "" {
				existing.Name = input.Name
			}
			if input.Gender != "" {
				existing.Gender = input.Gender
			}
			if input.Age > 0 {
				existing.Age = input.Age
			}
			if input.Phone != "" {
				existing.Phone = input.Phone
			}
			if input.Email != "" {
				existing.Email = input.Email
			}
			if input.Department != "" {
				existing.Department = input.Department
			}
			if input.Major != "" {
				existing.Major = input.Major
			}
			if input.ClassName != "" {
				existing.ClassName = input.ClassName
			}
			if input.Status != "" {
				existing.Status = model.StudentStatus(input.Status)
			}
			if err := svc.Update(existing); err != nil {
				return "", fmt.Errorf("更新失败: %w", err)
			}
			data, _ := json.Marshal(existing)
			return string(data), nil
		},
	)
}

func NewDeleteStudentTool(svc *service.StudentService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "delete_student",
			Desc: "删除一个学生记录。需提供学生ID。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"id": {Type: "integer", Desc: "学生ID（必填）", Required: true},
			}),
		},
		func(ctx context.Context, input *deleteStudentInput) (output string, err error) {
			if err := svc.Delete(input.ID); err != nil {
				return "", fmt.Errorf("删除失败: %w", err)
			}
			return fmt.Sprintf("已成功删除学生 ID=%d", input.ID), nil
		},
	)
}
