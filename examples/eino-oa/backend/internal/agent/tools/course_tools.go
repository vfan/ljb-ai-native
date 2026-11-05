package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type queryCourseInput struct {
	CourseCode string `json:"course_code,omitempty"`
	Name       string `json:"name,omitempty"`
	Teacher    string `json:"teacher,omitempty"`
	Semester   string `json:"semester,omitempty"`
}

func NewQueryCourseTool(svc *service.CourseService) tool.InvokableTool {
	return utils.NewTool(
		&schema.ToolInfo{
			Name: "query_course",
			Desc: "查询课程信息。可按课程编号、课程名称、教师、学期进行搜索。",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"course_code": {Type: "string", Desc: "课程编号（支持模糊匹配）"},
				"name":        {Type: "string", Desc: "课程名称（支持模糊匹配）"},
				"teacher":     {Type: "string", Desc: "任课教师（支持模糊匹配）"},
				"semester":    {Type: "string", Desc: "学期"},
			}),
		},
		func(ctx context.Context, input *queryCourseInput) (output string, err error) {
			result, err := svc.List(repository.CourseQuery{
				CourseCode: input.CourseCode,
				Name:       input.Name,
				Teacher:    input.Teacher,
				Semester:   input.Semester,
				PageSize:   50,
			})
			if err != nil {
				return "", fmt.Errorf("查询失败: %w", err)
			}
			data, _ := json.Marshal(result)
			return string(data), nil
		},
	)
}
