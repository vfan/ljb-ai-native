# 学生管理系统 API 参考

## 服务地址

使用环境变量 `EINO_OA_BASE_URL` 指定，例如 `http://localhost:8080/api`。

## 鉴权

所有接口都需要请求头：

```
Authorization: Bearer $EINO_OA_API_TOKEN
```

- 未携带或令牌失效：`401`
- 只读令牌调用写接口：`403`
- 令牌管理接口（`/auth/tokens`）只接受浏览器登录态

## 学生管理

| 方法   | 路径             | 说明         |
| ------ | ---------------- | ------------ |
| GET    | /students        | 学生列表     |
| GET    | /students/:id    | 学生详情     |
| POST   | /students        | 新增学生     |
| PUT    | /students/:id    | 更新学生     |
| DELETE | /students/:id    | 删除学生     |

### 查询参数（GET /students）

| 参数         | 类型   | 说明                              |
| ------------ | ------ | --------------------------------- |
| name         | string | 按姓名模糊搜索                    |
| student_no   | string | 按学号搜索                        |
| department   | string | 按院系搜索                        |
| major        | string | 按专业搜索                        |
| status       | string | 按状态筛选（在读/休学/毕业/退学） |
| page         | int    | 页码，默认 1                      |
| page_size    | int    | 每页数量，默认 20                 |

### 请求体（POST / PUT）

```json
{
  "student_no": "2024001",
  "name": "张三",
  "gender": "男",
  "age": 20,
  "phone": "13800000001",
  "email": "zhangsan@example.com",
  "department": "计算机学院",
  "major": "软件工程",
  "class_name": "软工2401",
  "enrollment_year": 2024
}
```

## 课程管理

| 方法   | 路径            | 说明         |
| ------ | --------------- | ------------ |
| GET    | /courses        | 课程列表     |
| GET    | /courses/:id    | 课程详情     |
| POST   | /courses        | 新增课程     |
| PUT    | /courses/:id    | 更新课程     |
| DELETE | /courses/:id    | 删除课程     |

### 查询参数（GET /courses）

| 参数        | 类型   | 说明           |
| ----------- | ------ | -------------- |
| name        | string | 按课程名搜索   |
| course_code | string | 按课程代码搜索 |
| teacher     | string | 按教师搜索     |
| semester    | string | 按学期筛选     |

### 请求体（POST / PUT）

```json
{
  "course_code": "CS101",
  "name": "数据结构",
  "credits": 4.0,
  "department": "计算机学院",
  "teacher": "李教授",
  "semester": "2025-2026-1",
  "description": "数据结构与算法基础课程"
}
```

## 成绩管理

| 方法   | 路径                            | 说明         |
| ------ | ------------------------------- | ------------ |
| GET    | /grades                         | 成绩列表     |
| GET    | /grades/:id                     | 成绩详情     |
| POST   | /grades                         | 录入成绩     |
| PUT    | /grades/:id                     | 修改成绩     |
| DELETE | /grades/:id                     | 删除成绩     |
| GET    | /grades/stats/course/:courseId  | 课程成绩统计 |
| GET    | /grades/gpa/student/:studentId  | 学生 GPA     |
| GET    | /grades/warning                 | 学业预警     |

### 查询参数（GET /grades）

| 参数       | 类型 | 说明       |
| ---------- | ---- | ---------- |
| student_id | int  | 按学生筛选 |
| course_id  | int  | 按课程筛选 |
| semester   | string | 按学期筛选 |

### 查询参数（GET /grades/warning）

| 参数      | 类型  | 说明                |
| --------- | ----- | ------------------- |
| threshold | float | GPA 阈值，默认 2.0  |

### 请求体（POST / PUT）

```json
{
  "student_id": 1,
  "course_id": 1,
  "score": 85.0,
  "semester": "2025-2026-1"
}
```

## 数据模型

**学生 (Student)**
```json
{
  "id": 1,
  "student_no": "2024001",
  "name": "张三",
  "gender": "男",
  "age": 20,
  "phone": "13800000001",
  "email": "zhangsan@example.com",
  "department": "计算机学院",
  "major": "软件工程",
  "class_name": "软工2401",
  "enrollment_year": 2024,
  "status": "在读"
}
```

**课程 (Course)**
```json
{
  "id": 1,
  "course_code": "CS101",
  "name": "数据结构",
  "credits": 4.0,
  "department": "计算机学院",
  "teacher": "李教授",
  "semester": "2025-2026-1"
}
```

**成绩 (Grade)**
```json
{
  "id": 1,
  "student_id": 1,
  "course_id": 1,
  "score": 85.0,
  "grade_point": 3.7,
  "semester": "2025-2026-1",
  "status": "正常"
}
```

**成绩统计 (GradeStats)**
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

**学生 GPA**
```json
{
  "student_id": 1,
  "student_no": "2024001",
  "student_name": "张三",
  "gpa": 3.45,
  "total_courses": 5
}
```
