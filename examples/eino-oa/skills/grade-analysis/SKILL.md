---
name: grade-analysis
description: 大学生成绩智能分析 - 成绩统计、GPA 计算、学业预警和综合报告
version: 1.0.0
tags:
  - education
  - grade
  - gpa
  - analysis
metadata:
  openclaw:
    requires:
      bins:
        - curl
      env:
        - EINO_OA_BASE_URL
        - EINO_OA_API_TOKEN
    primaryEnv: EINO_OA_BASE_URL
---

# 成绩分析技能

## 概述

本技能用于对大学生的课程成绩进行智能分析，包括成绩统计、GPA 计算、学业预警和学生综合报告生成。通过 REST API 调用，所有结果以 JSON 格式返回。

## 服务地址

使用环境变量 `EINO_OA_BASE_URL` 指定后端地址，例如 `http://localhost:8080/api`。

## 鉴权（必须）

所有接口都需要访问令牌：

```bash
export EINO_OA_API_TOKEN="eo_pat_xxx"
```

本技能只读取数据，建议创建**只读**令牌：

1. 登录 Web 系统 →「访问令牌」→「新建令牌」→ 权限选择「只读」。
2. 或运行 `eino-oa-cli login --server http://localhost:8080`，授权时只申请 `read`。

本文档所有 `curl` 示例都已包含：

```bash
-H "Authorization: Bearer $EINO_OA_API_TOKEN"
```

没有令牌返回 `401`；只读令牌调用写接口返回 `403`。

## 可用接口

### 成绩查询

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades?student_id=1&course_id=2&semester=2025-2026-1"
```

支持的查询参数（均可选）：`student_id`、`course_id`、`semester`。返回成绩列表，包含学生和课程详情。

### 成绩录入

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X POST "$EINO_OA_BASE_URL/grades" \
  -H "Content-Type: application/json" \
  -d '{"student_id": 1, "course_id": 2, "score": 85, "semester": "2025-2026-1"}'
```

绩点根据百分制自动计算。

**绩点计算规则：**

| 分数段 | 绩点 |
| ------ | ---- |
| 90-100 | 4.0  |
| 85-89  | 3.7  |
| 82-84  | 3.3  |
| 78-81  | 3.0  |
| 75-77  | 2.7  |
| 72-74  | 2.3  |
| 68-71  | 2.0  |
| 64-67  | 1.5  |
| 60-63  | 1.0  |
| <60    | 0    |

### 修改成绩

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X PUT "$EINO_OA_BASE_URL/grades/1" \
  -H "Content-Type: application/json" \
  -d '{"score": 90}'
```

### 课程成绩分析

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/1"
```

返回平均分、最高分、最低分、及格率、参考人数。

输出示例：
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

### 学生 GPA 计算

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

返回加权 GPA（按学分加权）。

### 学业预警

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/warning?threshold=2.0"
```

查找 GPA 低于阈值的学生，默认阈值 2.0。

输出示例：
```json
{
  "threshold": 2.0,
  "students": [
    {"student_id": 4, "student_no": "2024004", "student_name": "赵六", "gpa": 1.30, "total_courses": 3},
    {"student_id": 5, "student_no": "2024005", "student_name": "孙七", "gpa": 0.18, "total_courses": 3}
  ]
}
```

### 学生综合报告

综合报告需要组合多个接口：

```bash
# 1. 获取学生信息
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students/1"

# 2. 获取该学生所有成绩
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades?student_id=1"

# 3. 获取该学生 GPA
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

## 典型分析场景

### 场景1：分析某门课程的成绩分布

```bash
# 1. 查找课程 ID
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/courses?name=数据结构"

# 2. 分析成绩（假设课程 ID 为 1）
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/stats/course/1"
```

### 场景2：学业预警

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/warning?threshold=2.0"
```

### 场景3：生成学生综合报告

```bash
# 1. 搜索学生
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students?name=张三"

# 2. 获取详情 + 成绩 + GPA（假设学生 ID 为 1）
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students/1"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades?student_id=1"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

### 场景4：批量查看某学生的成绩

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades?student_id=1"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/grades/gpa/student/1"
```

详细示例请查看 [reference/examples.md](reference/examples.md)。
