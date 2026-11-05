package repository

import (
	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"gorm.io/gorm"
)

type GradeRepo struct {
	db *gorm.DB
}

func NewGradeRepo(db *gorm.DB) *GradeRepo {
	return &GradeRepo{db: db}
}

type GradeQuery struct {
	StudentID uint
	CourseID  uint
	Semester  string
	Status    string
	Page      int
	PageSize  int
}

func (r *GradeRepo) List(q GradeQuery) (*PaginatedResult[model.Grade], error) {
	tx := r.db.Model(&model.Grade{}).Preload("Student").Preload("Course")

	if q.StudentID > 0 {
		tx = tx.Where("student_id = ?", q.StudentID)
	}
	if q.CourseID > 0 {
		tx = tx.Where("course_id = ?", q.CourseID)
	}
	if q.Semester != "" {
		tx = tx.Where("semester = ?", q.Semester)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}

	var total int64
	tx.Count(&total)

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	var grades []model.Grade
	err := tx.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Order("id DESC").Find(&grades).Error
	if err != nil {
		return nil, err
	}

	return &PaginatedResult[model.Grade]{
		Items:    grades,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *GradeRepo) GetByID(id uint) (*model.Grade, error) {
	var g model.Grade
	err := r.db.Preload("Student").Preload("Course").First(&g, id).Error
	return &g, err
}

func (r *GradeRepo) Create(g *model.Grade) error {
	g.GradePoint = model.ScoreToGradePoint(g.Score)
	return r.db.Create(g).Error
}

func (r *GradeRepo) Update(g *model.Grade) error {
	g.GradePoint = model.ScoreToGradePoint(g.Score)
	return r.db.Save(g).Error
}

func (r *GradeRepo) Delete(id uint) error {
	return r.db.Delete(&model.Grade{}, id).Error
}

type GradeStats struct {
	CourseID   uint    `json:"course_id"`
	CourseName string  `json:"course_name"`
	AvgScore   float64 `json:"avg_score"`
	MaxScore   float64 `json:"max_score"`
	MinScore   float64 `json:"min_score"`
	PassRate   float64 `json:"pass_rate"`
	Count      int     `json:"count"`
}

func (r *GradeRepo) StatsByCourse(courseID uint) (*GradeStats, error) {
	var stats GradeStats
	err := r.db.Model(&model.Grade{}).
		Select("course_id, AVG(score) as avg_score, MAX(score) as max_score, MIN(score) as min_score, COUNT(*) as count").
		Where("course_id = ?", courseID).
		Group("course_id").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	var passCount int64
	r.db.Model(&model.Grade{}).Where("course_id = ? AND score >= 60", courseID).Count(&passCount)
	if stats.Count > 0 {
		stats.PassRate = float64(passCount) / float64(stats.Count) * 100
	}
	stats.CourseID = courseID

	var course model.Course
	r.db.First(&course, courseID)
	stats.CourseName = course.Name

	return &stats, nil
}

type StudentGPA struct {
	StudentID    uint    `json:"student_id"`
	StudentNo    string  `json:"student_no"`
	StudentName  string  `json:"student_name"`
	GPA          float64 `json:"gpa"`
	TotalCourses int     `json:"total_courses"`
}

func (r *GradeRepo) CalculateGPA(studentID uint) (*StudentGPA, error) {
	var grades []model.Grade
	err := r.db.Preload("Student").Preload("Course").Where("student_id = ?", studentID).Find(&grades).Error
	if err != nil {
		return nil, err
	}

	if len(grades) == 0 {
		return &StudentGPA{StudentID: studentID}, nil
	}

	var totalPoints, totalCredits float64
	for _, g := range grades {
		totalPoints += g.GradePoint * g.Course.Credits
		totalCredits += g.Course.Credits
	}

	gpa := &StudentGPA{
		StudentID:    studentID,
		StudentNo:    grades[0].Student.StudentNo,
		StudentName:  grades[0].Student.Name,
		TotalCourses: len(grades),
	}
	if totalCredits > 0 {
		gpa.GPA = totalPoints / totalCredits
	}

	return gpa, nil
}

func (r *GradeRepo) GetLowGPAStudents(threshold float64) ([]StudentGPA, error) {
	var studentIDs []uint
	r.db.Model(&model.Grade{}).Distinct("student_id").Pluck("student_id", &studentIDs)

	var result []StudentGPA
	for _, sid := range studentIDs {
		gpa, err := r.CalculateGPA(sid)
		if err != nil {
			continue
		}
		if gpa.GPA < threshold && gpa.TotalCourses > 0 {
			result = append(result, *gpa)
		}
	}
	return result, nil
}
