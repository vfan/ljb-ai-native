package model

import "time"

type GradeStatus string

const (
	GradeStatusNormal GradeStatus = "正常"
	GradeStatusRetake GradeStatus = "补考"
	GradeStatusRepeat GradeStatus = "重修"
)

type Grade struct {
	ID         uint        `json:"id" gorm:"primaryKey"`
	StudentID  uint        `json:"student_id" gorm:"index;not null"`
	CourseID   uint        `json:"course_id" gorm:"index;not null"`
	Score      float64     `json:"score"`
	GradePoint float64     `json:"grade_point"`
	Semester   string      `json:"semester" gorm:"size:20"`
	Status     GradeStatus `json:"status" gorm:"size:20;default:正常"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`

	Student Student `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Course  Course  `json:"course,omitempty" gorm:"foreignKey:CourseID"`
}

// ScoreToGradePoint converts a score (0-100) to a grade point (0-4.0).
func ScoreToGradePoint(score float64) float64 {
	switch {
	case score >= 90:
		return 4.0
	case score >= 85:
		return 3.7
	case score >= 82:
		return 3.3
	case score >= 78:
		return 3.0
	case score >= 75:
		return 2.7
	case score >= 72:
		return 2.3
	case score >= 68:
		return 2.0
	case score >= 64:
		return 1.5
	case score >= 60:
		return 1.0
	default:
		return 0
	}
}
