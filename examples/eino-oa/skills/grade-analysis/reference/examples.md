# 成绩分析示例

所有示例使用环境变量 `EINO_OA_BASE_URL` 指定后端地址，例如 `http://localhost:8080/api`，
并通过 `EINO_OA_API_TOKEN` 提供只读访问令牌。所有 `curl` 示例已包含：

```bash
-H "Authorization: Bearer $EINO_OA_API_TOKEN"
```

## 示例1：课程成绩分析

**问题：** "分析一下数据结构这门课的成绩分布"

**步骤：**

1. 查找课程：
```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/courses?name=数据结构"
```

2. 假设返回课程 ID 为 1，分析成绩：
```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/1"
```

3. 返回结果：
```json
{
  "course_id": 1,
  "course_name": "数据结构",
  "avg_score": 86.5,
  "max_score": 95,
  "min_score": 72,
  "pass_rate": 100,
  "count": 4
}
```

## 示例2：学业预警

**问题：** "哪些学生的 GPA 低于 2.0？"

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/warning?threshold=2.0"
```

返回结果：
```json
{
  "threshold": 2.0,
  "students": [
    {"student_id": 4, "student_no": "2024004", "student_name": "赵六", "gpa": 1.30, "total_courses": 3},
    {"student_id": 5, "student_no": "2024005", "student_name": "孙七", "gpa": 0.18, "total_courses": 3}
  ]
}
```

## 示例3：学生综合报告

**问题：** "帮我生成张三的成绩报告"

1. 查找学生：
```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students?name=张三"
```

2. 获取详情、成绩和 GPA（假设学生 ID 为 1）：
```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students/1"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades?student_id=1"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

3. 返回包含：
   - 学生基本信息（学号、姓名、院系、专业等）
   - 所有课程成绩列表
   - 加权 GPA 计算结果

## 示例4：录入成绩后查看 GPA 变化

```bash
# 录入新成绩
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X POST "$EINO_OA_BASE_URL/grades" \
  -H "Content-Type: application/json" \
  -d '{"student_id": 1, "course_id": 6, "score": 78, "semester": "2025-2026-1"}'

# 查看更新后的 GPA
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

## 示例5：组合分析

对比多门课程的成绩情况：

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/1"   # 数据结构
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/4"   # 高等数学
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/6"   # 大学物理
```
