---
name: student-manage
description: 大学生信息管理系统 - 通过 REST API 管理学生和课程信息
version: 1.0.0
tags:
  - education
  - student
  - course
  - crud
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

# 学生管理技能

## 概述

本技能用于操作大学生信息管理系统，提供学生信息和课程信息的增删改查能力。系统通过 REST API 对外提供操作接口，所有接口返回 JSON 格式结果。

## 服务地址

使用环境变量 `EINO_OA_BASE_URL` 指定后端地址，例如 `http://localhost:8080/api`。

## 鉴权（必须）

所有接口都需要访问令牌：

```bash
export EINO_OA_API_TOKEN="eo_pat_xxx"
```

令牌获取方式：

1. 登录 Web 系统 →「访问令牌」→「新建令牌」，选择「只读」或「读写」。
2. 或运行 `eino-oa-cli login --server http://localhost:8080`，在浏览器中同意授权；
   之后用 `eino-oa-cli token` 获取短期令牌（过期自动续期）。

本文档所有 `curl` 示例都已包含：

```bash
-H "Authorization: Bearer $EINO_OA_API_TOKEN"
```

没有令牌返回 `401`；只读令牌调用新增/修改/删除接口返回 `403`。

## 可用接口

### 学生查询

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students?name=张三&department=计算机学院&status=在读"
```

支持的查询参数（均可选）：`name`、`student_no`、`department`、`major`、`status`、`page`、`page_size`。
状态可选值：在读、休学、毕业、退学。支持模糊匹配。

示例：
```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students?name=张三"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students?department=计算机学院&status=在读"
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students"                # 查询全部
```

### 获取单个学生

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/students/1"
```

### 新增学生

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X POST "$EINO_OA_BASE_URL/students" \
  -H "Content-Type: application/json" \
  -d '{
    "student_no": "2024010",
    "name": "刘明",
    "gender": "男",
    "age": 19,
    "department": "计算机学院",
    "major": "软件工程",
    "class_name": "软工2401",
    "enrollment_year": 2024
  }'
```

`student_no` 和 `name` 为必填，其他可选。

### 修改学生

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X PUT "$EINO_OA_BASE_URL/students/1" \
  -H "Content-Type: application/json" \
  -d '{"name": "张三丰", "phone": "13900001111"}'
```

### 删除学生

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X DELETE "$EINO_OA_BASE_URL/students/1"
```

### 课程查询

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/courses?name=数据结构&teacher=李教授&semester=2025-2026-1"
```

支持的查询参数（均可选）：`name`、`course_code`、`teacher`、`semester`。

### 获取单个课程

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" "$EINO_OA_BASE_URL/courses/1"
```

### 新增课程

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X POST "$EINO_OA_BASE_URL/courses" \
  -H "Content-Type: application/json" \
  -d '{
    "course_code": "CS201",
    "name": "操作系统",
    "credits": 3.5,
    "department": "计算机学院",
    "teacher": "王教授",
    "semester": "2025-2026-2"
  }'
```

### 删除课程

```bash
curl -H "Authorization: Bearer $EINO_OA_API_TOKEN" -X DELETE "$EINO_OA_BASE_URL/courses/1"
```

## 输出格式

查询接口返回分页结果：

```json
{
  "items": [...],
  "total": 8,
  "page": 1,
  "page_size": 20
}
```

## 典型用法

1. 先用 `GET /students?name=X` 搜索学生获取 ID
2. 用 `GET /students/:id` 查看完整信息
3. 需要新增时调用 `POST /students`
4. 需要修改时调用 `PUT /students/:id`
5. 需要删除时调用 `DELETE /students/:id`

详细 API 参考请查看 [reference/api.md](reference/api.md)。
